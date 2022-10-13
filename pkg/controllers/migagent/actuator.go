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

type migProfilePlan struct {
	gpuIndex        int
	migProfile      string
	desiredQuantity int
	actualResources []types.MigDeviceResource
}

type migConfigPlan []migProfilePlan

func (a *MigActuator) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	var res ctrl.Result

	logger := log.FromContext(ctx).WithName("Actuator")
	logger.Info("Actuating desired MIG config")

	// Retrieve instance
	var instance v1.Node
	if err := a.Client.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, err
	}

	// Check if reported status already matches spec
	statusAnnotations, specAnnotations := types.GetGPUAnnotationsFromNode(instance)
	if mig.SpecMatchesStatus(specAnnotations, statusAnnotations) {
		logger.Info("Reported status matches desired MIG config, nothing to do")
		return ctrl.Result{}, nil
	}

	// Compute current state
	migDeviceResources, err := a.migClient.GetMigDeviceResources(ctx)
	if err != nil {
		logger.Error(err, "unable to get MIG device resources")
		return ctrl.Result{}, nil
	}
	state := types.NewMigState(migDeviceResources)

	// Check if actual state already matches spec
	if state.Matches(specAnnotations) {
		logger.Info("Actual status matches desired MIG config, nothing to do")
		return ctrl.Result{}, nil
	}

	// MIG resources which MIG profile is not present in spec annotations must be deleted
	//toDelete := getResourcesToDelete(specAnnotations, state)
	//logger.V(1).Info("MIG resources to delete", "resources", toDelete)

	//res, err = a.deleteMigResources(ctx, toDelete)
	//
	// Create MIG resources
	// todo

	return res, err
}

func (a *MigActuator) deleteMigResources(ctx context.Context, toDelete []types.MigDeviceResource) (ctrl.Result, error) {
	var err error
	var res ctrl.Result
	logger := klog.FromContext(ctx)

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

func plan(state types.MigState, desired []types.GPUSpecAnnotation) migConfigPlan {
	result := make(migConfigPlan, 0)

	//resources := getResourcesNotIncludedInSpec(state, desired)
	//groupedByGpuIndexAndProfile := make(map[string][]types.MigDeviceResource)
	//for _, r := range resources {
	//	groupedByGpuIndexAndProfile[]
	//}

	return result
}

func getResourcesNotIncludedInSpec(state types.MigState, specAnnotations []types.GPUSpecAnnotation) []types.MigDeviceResource {
	// Group by GPU index
	lookup := make(map[int]types.GPUAnnotationList)
	for _, annotation := range specAnnotations {
		gpuIndex := annotation.GetGPUIndex()
		if lookup[gpuIndex] == nil {
			lookup[gpuIndex] = make(types.GPUAnnotationList, 0)
		}
		lookup[gpuIndex] = append(lookup[gpuIndex], annotation)
	}

	updatedState := state
	for gpuIndex, annotations := range lookup {
		migProfiles := make([]string, 0)
		for _, a := range annotations {
			migProfiles = append(migProfiles, a.GetMigProfile())
		}
		updatedState = updatedState.WithoutMigProfiles(gpuIndex, migProfiles)
	}

	return updatedState.Flatten()
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
