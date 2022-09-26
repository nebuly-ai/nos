package core

import (
	"context"
	v1 "k8s.io/api/core/v1"
)

type Planner interface {
	GetNodesPartitioningPlan(pendingPods []v1.Pod) (map[string]PartitioningPlan, error)
}

type Actuator interface {
	ApplyPartitioning(ctx context.Context, plan map[string]PartitioningPlan) error
}

type Partitioner interface {
	RunPartitioning(ctx context.Context, pendingCandidates []v1.Pod) error
}

type PartitioningPlan struct {
	ResourceName v1.ResourceName
	Quantity     int
}
