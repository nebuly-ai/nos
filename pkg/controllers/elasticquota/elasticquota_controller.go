package elasticquota

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	Scheme             *runtime.Scheme
	resourceCalculator *gpu.Calculator
	podsReconciler     *elasticQuotaPodsReconciler
}

func NewElasticQuotaReconciler(client client.Client, scheme *runtime.Scheme, nvidiaGPUResourceMemoryGB int64) ElasticQuotaReconciler {
	resourceCalculator := gpu.Calculator{NvidiaGPUDeviceMemoryGB: nvidiaGPUResourceMemoryGB}
	return ElasticQuotaReconciler{
		Client:             client,
		Scheme:             scheme,
		resourceCalculator: &resourceCalculator,
		podsReconciler: &elasticQuotaPodsReconciler{
			c:                  client,
			resourceCalculator: &resourceCalculator,
		},
	}
}

//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=elasticquotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=elasticquotas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=elasticquotas/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;patch

func (r *ElasticQuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch EQ instance
	var instance v1alpha1.ElasticQuota
	if err := r.Client.Get(ctx, req.NamespacedName, &instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch running Pods in the EQ namespace
	var runningPodList v1.PodList
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingFields{constant.PodPhaseKey: string(v1.PodRunning)},
	}
	if err := r.Client.List(ctx, &runningPodList, opts...); err != nil {
		logger.Error(err, "unable to list running Pods")
		return ctrl.Result{}, err
	}

	// Update pods in EQ namespaces and compute used quota
	used, err := r.podsReconciler.PatchPodsAndComputeUsedQuota(
		ctx,
		runningPodList.Items,
		instance.Spec.Min,
		instance.Spec.Max,
	)
	if err != nil {
		return ctrl.Result{}, nil
	}

	// Update EQ status
	instance.Status.Used = used
	if err = r.updateStatus(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ElasticQuotaReconciler) updateStatus(ctx context.Context, instance v1alpha1.ElasticQuota) error {
	var logger = log.FromContext(ctx)
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
	if err := r.Status().Patch(ctx, &instance, client.MergeFrom(&currentEq)); err != nil {
		logger.Error(err, "unable to update ElasticQuota status")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ElasticQuotaReconciler) SetupWithManager(mgr ctrl.Manager, name string) error {
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Pod{}, constant.PodPhaseKey, func(rawObj client.Object) []string {
		pod := rawObj.(*v1.Pod)
		return []string{string(pod.Status.Phase)}
	})
	if err != nil {
		return err
	}

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
						// Reconcile only if Pod changed phase, and either old or new phase status is Running
						newPod := updateEvent.ObjectNew.(*v1.Pod)
						oldPod := updateEvent.ObjectOld.(*v1.Pod)
						statusChanged := newPod.Status.Phase != oldPod.Status.Phase
						anyRunning := (newPod.Status.Phase == v1.PodRunning) || (oldPod.Status.Phase == v1.PodRunning)
						return statusChanged && anyRunning
					},
					GenericFunc: func(_ event.GenericEvent) bool {
						return false
					},
				},
			),
		).
		Complete(r)
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
