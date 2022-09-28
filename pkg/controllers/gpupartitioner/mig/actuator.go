package mig

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/core"
)

type Actuator struct {
}

func NewActuator() *Actuator {
	return nil
}

func (a Actuator) ApplyPartitioning(ctx context.Context, plan map[string]core.PartitioningPlan) error {
	//TODO implement me
	panic("implement me")
}
