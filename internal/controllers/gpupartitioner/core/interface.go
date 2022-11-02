package core

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	v1 "k8s.io/api/core/v1"
)

type Planner interface {
	Plan(ctx context.Context, snapshot state.ClusterSnapshot, pendingPods []v1.Pod) (state.ClusterPartitioning, error)
}

type Actuator interface {
	Apply(ctx context.Context, partitioning state.ClusterPartitioning) error
}
