package core

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"github.com/nebuly-ai/nebulnetes/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"time"
)

type Controller struct {
	client.Client
	Scheme       *runtime.Scheme
	logger       logr.Logger
	podBatcher   util.Batcher[v1.Pod]
	clusterState *state.ClusterState
	currentBatch map[string]v1.Pod
	planner      Planner
	actuator     Actuator
}

func NewController(
	scheme *runtime.Scheme,
	client client.Client,
	logger logr.Logger,
	podBatcher util.Batcher[v1.Pod],
	clusterState *state.ClusterState,
	planner Planner,
	actuator Actuator) Controller {
	return Controller{
		Scheme:       scheme,
		Client:       client,
		logger:       logger,
		clusterState: clusterState,
		currentBatch: make(map[string]v1.Pod),
		podBatcher:   podBatcher,
		planner:      planner,
		actuator:     actuator,
	}
}

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	c.logger.V(3).Info("*** start reconcile ***")
	defer c.logger.V(3).Info("*** end reconcile ***")
	var requeueNecessary bool

	// Fetch instance
	var instance v1.Pod
	if err := c.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If Pod is not pending then don't add it to the current batch
	if instance.Status.Phase != v1.PodPending {
		c.logger.V(3).Info("pod is not pending, skipping it", "pod", instance.Name, "namespace", instance.Namespace)
		return ctrl.Result{}, nil
	}

	// Add Pod to current batch only if not already present
	if _, ok := c.currentBatch[util.GetNamespacedName(&instance).String()]; !ok {
		c.podBatcher.Add(instance)
		c.currentBatch[util.GetNamespacedName(&instance).String()] = instance
		c.logger.V(1).Info("batch updated", "pod", instance.Name, "namespace", instance.Namespace)
		// pod has been added to current batch, requeue in order to process it in the next cycle
		requeueNecessary = true
	}

	// If batch is ready then process pending pods
	select {
	case batch := <-c.podBatcher.Ready():
		c.currentBatch = make(map[string]v1.Pod)
		return c.processPendingPods(ctx, batch)
	default:
		c.logger.V(1).Info("batch not ready")
	}

	if requeueNecessary {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

func (c *Controller) processPendingPods(ctx context.Context, pods []v1.Pod) (ctrl.Result, error) {
	c.logger.V(1).Info("*** processing pending pods ***")
	defer c.logger.V(1).Info("*** end processing pending pods ***")

	c.logger.Info(fmt.Sprintf("processing %d pods", len(pods)))
	snapshot := c.clusterState.GetSnapshot()

	// Keep only pending pods that could benefit from
	// extra resources created through GPU partitioning
	pendingCandidates := make([]v1.Pod, 0)
	for _, p := range pods {
		if p.Status.Phase != v1.PodPending {
			continue
		}
		if pod.ExtraResourcesCouldHelpScheduling(p) {
			pendingCandidates = append(pendingCandidates, p)
		}
	}

	nPendingCandidates := len(pendingCandidates)
	c.logger.Info(fmt.Sprintf("found %d pendiong candidate pods", nPendingCandidates))
	if nPendingCandidates == 0 {
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
