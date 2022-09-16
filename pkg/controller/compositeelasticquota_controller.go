package controller

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CompositeElasticQuotaReconciler reconciles a CompositeElasticQuota object
type CompositeElasticQuotaReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	resourceCalculator util.ResourceCalculator
}

func NewCompositeElasticQuotaReconciler(client client.Client, scheme *runtime.Scheme, nvidiaGPUResourceMemoryGB int64) CompositeElasticQuotaReconciler {
	return CompositeElasticQuotaReconciler{
		Client:             client,
		Scheme:             scheme,
		resourceCalculator: util.ResourceCalculator{NvidiaGPUDeviceMemoryGB: nvidiaGPUResourceMemoryGB},
	}
}

//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=compositeelasticquotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=compositeelasticquotas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=compositeelasticquotas/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;patch

func (r *CompositeElasticQuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
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
