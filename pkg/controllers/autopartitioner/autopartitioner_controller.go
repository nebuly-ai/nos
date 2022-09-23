package autopartitioner

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/util/pod"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Controller struct {
	client.Client
	Scheme             *runtime.Scheme
	resourceCalculator *resource.Calculator
}

func NewController(client client.Client, scheme *runtime.Scheme) Controller {
	return Controller{
		Client: client,
		Scheme: scheme,
	}
}

//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=elasticquotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=elasticquotas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=elasticquotas/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;patch

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//logger := log.FromContext(ctx)

	// Fetch instance
	var instance v1.Pod
	if err := c.Client.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If extra resources generated from partitioning don't help with
	// scheduling then we don't need to reconcile this Pod
	if !pod.ExtraResourcesCouldHelpScheduling(instance) {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (c *Controller) SetupWithManager(mgr ctrl.Manager, name string) error {
	return nil
}
