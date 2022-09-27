package mig

import (
	"context"
	v1 "k8s.io/api/core/v1"
)

type Actuator struct {
}

func NewActuator() *Actuator {
	return nil
}

func (a Actuator) ApplyPartitioning(ctx context.Context, plan map[string]v1.ResourceList) error {
	//TODO implement me
	panic("implement me")
}
