package mig

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/core"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/state"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Partitioner struct {
	clusterState *state.ClusterState
	client       client.Client
	k8sClient    kubernetes.Interface
	actuator     core.Actuator
}

func NewPartitioner(state *state.ClusterState, client client.Client, k8sClient kubernetes.Interface) Partitioner {
	return Partitioner{
		clusterState: state,
		client:       client,
		k8sClient:    k8sClient,
		actuator:     NewActuator(),
	}
}

func (p Partitioner) RunPartitioning(ctx context.Context, pendingCandidates []v1.Pod) error {
	logger := log.FromContext(ctx).WithName("mig-partitioner")

	// Init planner
	planner, err := NewPlanner(p.k8sClient, p.clusterState.GetSnapshot())
	if err != nil {
		return err
	}

	// Compute partitioning plan
	plan, err := planner.GetNodesPartitioningPlan(ctx, pendingCandidates)
	if err != nil {
		logger.Error(err, "unable to compute partitioning plan")
		return err
	}
	if len(plan) == 0 {
		logger.Info(
			"Partitioning plan is empty, nothing to do",
			"nPendingCandidatePods",
			len(pendingCandidates),
		)
		return nil
	}

	// Apply partitioning plan
	if err := p.actuator.ApplyPartitioning(ctx, plan); err != nil {
		logger.Error(err, "unable to apply partitioning plan")
		return err
	}

	return nil
}
