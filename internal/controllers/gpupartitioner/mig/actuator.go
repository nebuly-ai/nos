package mig

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Actuator struct {
}

func NewActuator() *Actuator {
	return nil
}

func (a *Actuator) newLogger(ctx context.Context) klog.Logger {
	return log.FromContext(ctx).WithName("MigActuator")
}

func (a *Actuator) Apply(ctx context.Context, plan state.PartitioningState) error {
	var err error
	logger := a.newLogger(ctx)
	logger.Info(
		"applying plan",
		"createOperations",
		//plan.CreateOperations,
		"deleteOperations",
		//plan.DeleteOperations,
	)

	return err
}
