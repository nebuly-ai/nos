package state

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PodController struct {
	client.Client
	Scheme       *runtime.Scheme
	clusterState *ClusterState
}

func NewPodController(client client.Client, scheme *runtime.Scheme, state *ClusterState) PodController {
	return PodController{
		Client:       client,
		Scheme:       scheme,
		clusterState: state,
	}
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (c *PodController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch instance
	var instance v1.Pod
	objKey := client.ObjectKey{Namespace: req.Namespace, Name: req.Name}
	err := c.Client.Get(ctx, objKey, &instance)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "unable to fetch pod")
		return ctrl.Result{}, err
	}

	// If Pod does not exist then remove it from Cluster State
	if apierrors.IsNotFound(err) {
		if err := c.clusterState.deletePod(req.NamespacedName); err != nil {
			logger.Error(err, "unable to remove pod from cluster state")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// If Pod is not assigned to any node then skip state update
	if instance.Spec.NodeName == "" {
		return ctrl.Result{}, nil
	}

	// If node does not exist already exists in cluster state we need to add it
	if _, found := c.clusterState.GetNode(instance.Spec.NodeName); !found {
		var podNode v1.Node
		nodeKey := client.ObjectKey{Namespace: "", Name: instance.Spec.NodeName}
		if err = c.Client.Get(ctx, nodeKey, &podNode); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		var podList v1.PodList
		if err = c.Client.List(ctx, &podList, client.MatchingFields{constant.PodNodeNameKey: instance.Spec.NodeName}); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		c.clusterState.updateNode(podNode, podList.Items)

		return ctrl.Result{}, nil
	}

	c.clusterState.updateUsage(instance)
	return ctrl.Result{}, nil
}

func (c *PodController) SetupWithManager(mgr ctrl.Manager, name string) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1.Pod{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Complete(c)
}
