package controller

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	quota "k8s.io/apiserver/pkg/quota/v1"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	kubefeatures "k8s.io/kubernetes/pkg/features"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ElasticQuotaReconciler reconciles a ElasticQuota object
type ElasticQuotaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ElasticQuotaReconciler) updateStatus(ctx context.Context, instance *v1alpha1.ElasticQuota, logger logr.Logger) error {
	var currentEq v1alpha1.ElasticQuota
	namespacedName := types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}

	if err := r.Get(ctx, namespacedName, &currentEq); err != nil {
		logger.Error(err, "unable to fetch ElasticQuota")
		return err
	}
	if equality.Semantic.DeepEqual(currentEq.Status, instance.Status) {
		logger.V(1).Info("current status and desired status of ElasticQuota are equal, skipping update")
		return nil
	}
	logger.V(1).Info("updating ElasticQuota status", "Status", instance.Status)
	if err := r.Status().Update(ctx, instance); err != nil {
		logger.Error(err, "unable to update ElasticQuota status")
		return err
	}

	return nil
}

//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=elasticquotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=elasticquotas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=elasticquotas/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;

func (r *ElasticQuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var instance v1alpha1.ElasticQuota
	if err := r.Client.Get(ctx, req.NamespacedName, &instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	resourceList, err := r.computeElasticQuotaUsed(ctx, req.Namespace, &instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	instance.Status.Used = resourceList

	err = r.updateStatus(ctx, &instance, logger)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ElasticQuotaReconciler) computeElasticQuotaUsed(ctx context.Context, namespace string, eq *v1alpha1.ElasticQuota) (v1.ResourceList, error) {
	logger := log.FromContext(ctx)

	used := newZeroUsed(eq)
	var podList v1.PodList
	if err := r.Client.List(ctx, &podList, client.InNamespace(namespace)); err != nil {
		logger.Error(err, "unable to list Pods")
		return used, err
	}
	for _, p := range podList.Items {
		if p.Status.Phase == v1.PodRunning {
			used = quota.Add(used, computePodResourceRequest(&p))
		}
	}
	return used, nil
}

// computePodResourceRequest returns a v1.ResourceList that covers the largest
// width in each resource dimension. Because init-containers run sequentially, we collect
// the max in each dimension iteratively. In contrast, we sum the resource vectors for
// regular containers since they run simultaneously.
//
// If Pod Overhead is specified and the feature gate is set, the resources defined for Overhead
// are added to the calculated Resource request sum
//
// Example:
//
// Pod:
//
//	InitContainers
//	  IC1:
//	    CPU: 2
//	    Memory: 1G
//	  IC2:
//	    CPU: 2
//	    Memory: 3G
//	Containers
//	  C1:
//	    CPU: 2
//	    Memory: 1G
//	  C2:
//	    CPU: 1
//	    Memory: 1G
//
// Result: CPU: 3, Memory: 3G
func computePodResourceRequest(pod *v1.Pod) v1.ResourceList {
	result := v1.ResourceList{}
	for _, container := range pod.Spec.Containers {
		result = quota.Add(result, container.Resources.Requests)
	}
	initRes := v1.ResourceList{}
	// take max_resource for init_containers
	for _, container := range pod.Spec.InitContainers {
		initRes = quota.Max(initRes, container.Resources.Requests)
	}
	// If Overhead is being utilized, add to the total requests for the pod
	if pod.Spec.Overhead != nil && utilfeature.DefaultFeatureGate.Enabled(kubefeatures.PodOverhead) {
		quota.Add(result, pod.Spec.Overhead)
	}
	// take max_resource for init_containers and containers
	return quota.Max(result, initRes)
}

// newZeroUsed will return the zero value of the union of min and max
func newZeroUsed(eq *v1alpha1.ElasticQuota) v1.ResourceList {
	minResources := quota.ResourceNames(eq.Spec.Min)
	maxResources := quota.ResourceNames(eq.Spec.Max)
	res := v1.ResourceList{}
	for _, v := range minResources {
		res[v] = *resource.NewQuantity(0, resource.DecimalSI)
	}
	for _, v := range maxResources {
		res[v] = *resource.NewQuantity(0, resource.DecimalSI)
	}
	return res
}

func (r *ElasticQuotaReconciler) findElasticQuotaForPod(pod client.Object) []reconcile.Request {
	ctx := context.Background()
	logger := log.FromContext(ctx)

	var eqList v1alpha1.ElasticQuotaList
	err := r.Client.List(ctx, &eqList, client.InNamespace(pod.GetNamespace()))
	if err != nil {
		logger.Error(err, "unable to list ElasticQuotas")
		return []reconcile.Request{}
	}

	if len(eqList.Items) > 0 {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Name:      eqList.Items[0].Name,
				Namespace: eqList.Items[0].Namespace,
			},
		}}
	}

	return []reconcile.Request{}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ElasticQuotaReconciler) SetupWithManager(mgr ctrl.Manager, name string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ElasticQuota{}).
		Named(name).
		Watches(
			&source.Kind{Type: &v1.Pod{}},
			handler.EnqueueRequestsFromMapFunc(r.findElasticQuotaForPod),
			builder.WithPredicates(
				predicate.Funcs{
					CreateFunc: func(_ event.CreateEvent) bool {
						return false
					},
					DeleteFunc: func(_ event.DeleteEvent) bool {
						return true
					},
					UpdateFunc: func(updateEvent event.UpdateEvent) bool {
						// If new Pod is not Running, then do not trigger Reconcile
						newPod := updateEvent.ObjectNew.(*v1.Pod)
						if newPod.Status.Phase != v1.PodRunning {
							return false
						}
						// Trigger Reconcile only if the Pod updated from phase <any> to Running
						oldPod := updateEvent.ObjectOld.(*v1.Pod)
						return oldPod.Status.Phase != newPod.Status.Phase
					},
					GenericFunc: func(_ event.GenericEvent) bool {
						return false
					},
				},
			),
		).
		Complete(r)
}
