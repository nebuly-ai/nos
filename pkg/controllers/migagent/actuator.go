package migagent

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

type MigActuator struct {
	client.Client
	migClient mig.Client
}

func NewActuator(client client.Client, migClient mig.Client) MigActuator {
	reporter := MigActuator{
		Client:    client,
		migClient: migClient,
	}
	return reporter
}

func (a *MigActuator) newLogger(ctx context.Context) klog.Logger {
	return log.FromContext(ctx).WithName("Actuator")
}

func (a *MigActuator) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := a.newLogger(ctx)
	logger.Info("actuating desired MIG config")

	// Retrieve instance
	var instance v1.Node
	if err := a.Client.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, err
	}

	// Check if reported status already matches spec
	statusAnnotations, specAnnotations := types.GetGPUAnnotationsFromNode(instance)
	if mig.SpecMatchesStatus(specAnnotations, statusAnnotations) {
		logger.Info("reported status matches desired MIG config, nothing to do")
		return ctrl.Result{}, nil
	}

	// Compute MIG config plan
	plan, err := a.plan(ctx, specAnnotations)
	if err != nil {
		return ctrl.Result{}, err
	}
	if plan.isEmpty() {
		logger.Info("MIG config plan is empty, nothing to do")
		return ctrl.Result{}, nil
	}

	// Apply MIG config plan
	logger.Info("applying MIG config plan", "plan", plan.summary())
	return a.apply(plan)
}

func (a *MigActuator) plan(ctx context.Context, specAnnotations types.GPUSpecAnnotationList) (migConfigPlan, error) {
	logger := a.newLogger(ctx)

	// Compute current state
	migDeviceResources, err := a.migClient.GetMigDeviceResources(ctx)
	if err != nil {
		logger.Error(err, "unable to get MIG device resources")
		return nil, err
	}
	state := types.NewMigState(migDeviceResources)

	// Check if actual state already matches spec
	if state.Matches(specAnnotations) {
		logger.Info("actual state matches desired MIG config")
		return migConfigPlan{}, nil
	}

	// Compute MIG config plan
	return computePlan(state, specAnnotations), nil
}

func (a *MigActuator) apply(plan migConfigPlan) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (a *MigActuator) deleteMigResources(ctx context.Context, toDelete []types.MigDeviceResource) (ctrl.Result, error) {
	var err error
	var res ctrl.Result
	logger := a.newLogger(ctx)

	for _, r := range toDelete {
		// consider only free resources
		if r.Status != resource.StatusFree {
			logger.Info(
				"Cannot delete MIG resource because it is not in status 'free'",
				"status",
				r.Status,
				"GPU",
				r.GpuIndex,
				"resource",
				r.ResourceName,
			)
			continue
		}
		// try to delete device
		logger.Info("Deleting MIG resource", "GPU", r.GpuIndex, "resource", r.ResourceName)
		if err = a.migClient.DeleteMigResource(ctx, r); err != nil {
			logger.Error(
				err,
				"unable to delete MIG resource",
				"GPU",
				r.GpuIndex,
				"resource",
				r.ResourceName,
				"UUID",
				r.DeviceId,
			)
			// keep running, but reschedule for execution
			res = ctrl.Result{RequeueAfter: 10 * time.Second} // todo: use exponential backoff
		}
	}

	return res, err
}

func (a *MigActuator) SetupWithManager(mgr ctrl.Manager, controllerName string, nodeName string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&v1.Node{},
			builder.WithPredicates(
				excludeDeletePredicate{},
				matchingNamePredicate{Name: nodeName},
				annotationsChangedPredicate{},
			),
		).
		Named(controllerName).
		Complete(a)
}
