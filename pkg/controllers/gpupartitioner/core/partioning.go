package core

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/state"
	v1 "k8s.io/api/core/v1"
)

type PartitioningPlan map[string]v1.ResourceList

type Planner interface {
	GetNodesPartitioningPlan(ctx context.Context, snapshot state.ClusterSnapshot, pendingPods []v1.Pod) (map[string]PartitioningPlan, error)
}

type Actuator interface {
	ApplyPartitioning(ctx context.Context, plan map[string]PartitioningPlan) error
}
