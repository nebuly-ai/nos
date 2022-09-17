package controller

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CompositeElasticQuotaReconciler reconciles a CompositeElasticQuota object
type CompositeElasticQuotaReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	resourceCalculator *util.ResourceCalculator
	podsReconciler     *elasticQuotaPodsReconciler
}

func NewCompositeElasticQuotaReconciler(client client.Client, scheme *runtime.Scheme, nvidiaGPUResourceMemoryGB int64) CompositeElasticQuotaReconciler {
	resourceCalculator := util.ResourceCalculator{
		NvidiaGPUDeviceMemoryGB: nvidiaGPUResourceMemoryGB,
	}
	return CompositeElasticQuotaReconciler{
		Client:             client,
		Scheme:             scheme,
		resourceCalculator: &resourceCalculator,
		podsReconciler: &elasticQuotaPodsReconciler{
			c:                  client,
			resourceCalculator: &resourceCalculator,
		},
	}
}

//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=compositeelasticquotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=compositeelasticquotas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=compositeelasticquotas/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;patch

func (r *CompositeElasticQuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch CEQ instance
	var instance v1alpha1.CompositeElasticQuota
	if err := r.Client.Get(ctx, req.NamespacedName, &instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//Fetch running Pods in the namespaces specified by the EQ
	pods, err := r.fetchRunningPods(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Update pods in EQ namespaces and compute used quota
	used, err := r.podsReconciler.PatchPodsAndComputeUsedQuota(
		ctx,
		pods,
		instance.Spec.Min,
		instance.Spec.Max,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Update status
	instance.Status.Used = used
	if err = r.updateStatus(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *CompositeElasticQuotaReconciler) fetchRunningPods(ctx context.Context,
	eq v1alpha1.CompositeElasticQuota) ([]v1.Pod, error) {

	logger := log.FromContext(ctx)
	var result = make([]v1.Pod, 0)

	var namespaceRunningPods v1.PodList
	for _, namespace := range eq.Spec.Namespaces {
		opts := []client.ListOption{
			client.InNamespace(namespace),
			client.MatchingFields{podPhaseKey: string(v1.PodRunning)},
		}
		if err := r.Client.List(ctx, &namespaceRunningPods, opts...); err != nil {
			logger.Error(err, "unable to list running Pods", "namespace", namespace)
			return nil, err
		}
		result = append(result, namespaceRunningPods.Items...)
	}
	return result, nil
}

func (r *CompositeElasticQuotaReconciler) updateStatus(ctx context.Context, instance v1alpha1.CompositeElasticQuota) error {
	var logger = log.FromContext(ctx)
	var currentEq v1alpha1.CompositeElasticQuota
	namespacedName := types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}

	if err := r.Get(ctx, namespacedName, &currentEq); err != nil {
		logger.Error(err, "unable to fetch ElasticQuota")
		return err
	}
	if equality.Semantic.DeepEqual(currentEq.Status, instance.Status) {
		logger.V(1).Info("current status and desired status of ElasticQuota are equal, skipping update")
		return nil
	}
	logger.V(1).Info("updating CompositeElasticQuota status", "Status", instance.Status)
	if err := r.Status().Patch(ctx, &instance, client.MergeFrom(&currentEq)); err != nil {
		logger.Error(err, "unable to update CompositeElasticQuota status")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CompositeElasticQuotaReconciler) SetupWithManager(mgr ctrl.Manager, name string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CompositeElasticQuota{}).
		Named(name).
		//Watches(
		//	&source.Kind{Type: &v1.Pod{}},
		//	handler.EnqueueRequestsFromMapFunc(r.findCompositeElasticQuotaForPod),
		//	builder.WithPredicates(
		//		predicate.Funcs{
		//			CreateFunc: func(_ event.CreateEvent) bool {
		//				return false
		//			},
		//			DeleteFunc: func(_ event.DeleteEvent) bool {
		//				return true
		//			},
		//			UpdateFunc: func(updateEvent event.UpdateEvent) bool {
		//				// Reconcile only if Pod changed phase, and either old or new phase status is Running
		//				newPod := updateEvent.ObjectNew.(*v1.Pod)
		//				oldPod := updateEvent.ObjectOld.(*v1.Pod)
		//				statusChanged := newPod.Status.Phase != oldPod.Status.Phase
		//				anyRunning := (newPod.Status.Phase == v1.PodRunning) || (oldPod.Status.Phase == v1.PodRunning)
		//				return statusChanged && anyRunning
		//			},
		//			GenericFunc: func(_ event.GenericEvent) bool {
		//				return false
		//			},
		//		},
		//	),
		//).
		Complete(r)
}
