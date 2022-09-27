package core

import (
	"context"
	v1 "k8s.io/api/core/v1"
)

type Planner interface {
	GetNodesPartitioningPlan(ctx context.Context, pendingPods []v1.Pod) (map[string]v1.ResourceList, error)
}

type Actuator interface {
	ApplyPartitioning(ctx context.Context, plan map[string]v1.ResourceList) error
}

type Partitioner interface {
	RunPartitioning(ctx context.Context, pendingCandidates []v1.Pod) error
}
