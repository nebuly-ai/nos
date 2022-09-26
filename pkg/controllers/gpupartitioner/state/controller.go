package state

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Controller struct {
	client.Client
	Scheme       *runtime.Scheme
	clusterState ClusterState
}

func NewController(client client.Client, scheme *runtime.Scheme, state ClusterState) Controller {
	return Controller{
		Client:       client,
		Scheme:       scheme,
		clusterState: state,
	}
}

//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (c *Controller) SetupWithManager(mgr ctrl.Manager, name string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Node{}).
		Named(name).
		Complete(c)
}
