package core

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sort"
)

type Controller struct {
	client.Client
	Scheme       *runtime.Scheme
	partitioner  Partitioner
	clusterState *state.ClusterState
}

func NewController(client client.Client, scheme *runtime.Scheme, clusterState *state.ClusterState, partitioner Partitioner) Controller {
	return Controller{
		Client:       client,
		Scheme:       scheme,
		partitioner:  partitioner,
		clusterState: clusterState,
	}
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	pendingPods, err := getPendingPods()
	if err != nil {
		logger.Error(err, "unable to fetch pending pods")
		return ctrl.Result{}, err
	}

	// Keep only pending pods that could benefit from
	// extra resources created through GPU partitioning
	pendingCandidates := make([]v1.Pod, 0)
	for _, p := range pendingPods {
		if pod.ExtraResourcesCouldHelpScheduling(p) {
			pendingCandidates = append(pendingCandidates, p)
		}
	}
	if len(pendingCandidates) == 0 {
		logger.Info("there are no pending pods that could be helped through GPU partitioning")
		return ctrl.Result{}, nil
	}

	// Sort Pods by importance
	sort.Slice(pendingCandidates, func(i, j int) bool {
		return pod.IsMoreImportant(pendingCandidates[i], pendingCandidates[j])
	})

	// Run partitioning
	if err := c.partitioner.RunPartitioning(ctx, pendingCandidates); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func getPendingPods() ([]v1.Pod, error) {
	return nil, nil
}

func (c *Controller) SetupWithManager(mgr ctrl.Manager, name string) error {
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Pod{}, constant.PodPhaseKey, func(rawObj client.Object) []string {
		p := rawObj.(*v1.Pod)
		return []string{string(p.Status.Phase)}
	})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pod{}).
		Named(name).
		Complete(c)
}
