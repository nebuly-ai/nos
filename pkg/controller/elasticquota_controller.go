package controller

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	schedulerplugins "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"
)

// ElasticQuotaReconciler reconciles a ElasticQuota object
type ElasticQuotaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sigs.k8s.io,resources=elasticquotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sigs.k8s.io,resources=pods,verbs=get;list;watch;
//+kubebuilder:rbac:groups=sigs.k8s.io,resources=elasticquotas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sigs.k8s.io,resources=elasticquotas/finalizers,verbs=update

func (r *ElasticQuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ElasticQuotaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulerplugins.ElasticQuota{}).
		Complete(r)
}
