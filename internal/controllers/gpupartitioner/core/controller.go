package core

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"github.com/nebuly-ai/nebulnetes/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sort"
	"time"
)

type Controller struct {
	client.Client
	Scheme       *runtime.Scheme
	logger       logr.Logger
	podBatcher   util.Batcher[v1.Pod]
	clusterState *state.ClusterState
	planner      Planner
	actuator     Actuator
}

func NewController(
	scheme *runtime.Scheme,
	client client.Client,
	logger logr.Logger,
	podBatcher util.Batcher[v1.Pod],
	planner Planner,
	actuator Actuator) Controller {
	return Controller{
		Scheme:     scheme,
		Client:     client,
		logger:     logger,
		podBatcher: podBatcher,
		planner:    planner,
		actuator:   actuator,
	}
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;patch

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("Controller")
	logger.V(1).Info("*** start reconcile ***")
	defer logger.V(1).Info("*** end reconcile ***")

	// Fetch instance
	var instance v1.Pod
	if err := c.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If Pod is not pending then don't add it to the current batch
	if instance.Status.Phase != v1.PodPending {
		return ctrl.Result{}, nil
	}

	// Add Pod to current batch
	c.podBatcher.Add(instance)

	// If Pods batch is ready, then process it
	select {
	case <-c.podBatcher.Ready():
		return c.processPendingPods(ctx)
	default:
		c.logger.V(1).Info("batch not ready")
	}

	// Pod has been added to current batch, requeue in order to process it in the next cycle
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (c *Controller) processPendingPods(ctx context.Context) (ctrl.Result, error) {
	c.logger.V(1).Info("*** processing pending pods ***")
	defer c.logger.V(1).Info("*** end processing pending pods ***")

	snapshot := c.clusterState.GetSnapshot()

	// Keep only pending pods that could benefit from
	// extra resources created through GPU partitioning
	pendingCandidates := make([]v1.Pod, 0)
	for _, p := range snapshot.GetPendingPods() {
		if pod.ExtraResourcesCouldHelpScheduling(p) {
			pendingCandidates = append(pendingCandidates, p)
		}
	}
	if len(pendingCandidates) == 0 {
		c.logger.Info("there are no pending pods to help with GPU partitioning")
		return ctrl.Result{}, nil
	}

	// Sort Pods by importance
	sort.Slice(pendingCandidates, func(i, j int) bool {
		return pod.IsMoreImportant(pendingCandidates[i], pendingCandidates[j])
	})

	// Compute desired state
	desiredState, err := c.planner.Plan(ctx, snapshot, pendingCandidates)
	if err != nil {
		c.logger.Error(err, "unable to plan desired partitioning state")
		return ctrl.Result{}, err
	}

	// Apply partitioning plan
	if err = c.actuator.Apply(ctx, snapshot, desiredState); err != nil {
		c.logger.Error(err, "unable to apply desired partitioning state")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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
