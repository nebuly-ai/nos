package migagent

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/migagent/types"
	migtypes "github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
)

type migProfilePlan struct {
	migProfile      string
	gpuIndex        int
	desiredQuantity int
	actualResources []migtypes.MigDeviceResource
}

type migConfigPlan []migProfilePlan

func (p migConfigPlan) summary() string {
	toCreate := make([]migtypes.MigProfile, 0)
	toDelete := make([]migtypes.MigProfile, 0)
	for _, plan := range p {
		diff := plan.desiredQuantity - len(plan.actualResources)
		if diff > 0 {
			for i := 0; i < diff; i++ {
				toCreate = append(toCreate, migtypes.MigProfile{Name: plan.migProfile, GpuIndex: plan.gpuIndex})
			}
		}
		if diff < 0 {
			for i := 0; i < util.Abs(diff); i++ {
				toDelete = append(toDelete, migtypes.MigProfile{Name: plan.migProfile, GpuIndex: plan.gpuIndex})
			}
		}
	}
	return fmt.Sprintf("MIG profiles to create: %v MIG profiles to delete: %v", toCreate, toDelete)
}

func (p migConfigPlan) isEmpty() bool {
	return len(p) == 0
}

func computePlan(state types.MigState, desired types.GPUSpecAnnotationList) migConfigPlan {
	plan := make(migConfigPlan, 0)

	// Get resources present in current state which MIG profile is not included in spec
	for migProfile, resourceList := range getResourcesNotIncludedInSpec(state, desired).GroupByMigProfile() {
		p := migProfilePlan{
			gpuIndex:        migProfile.GpuIndex,
			migProfile:      migProfile.Name,
			desiredQuantity: 0, // we want these resources to be deleted
			actualResources: resourceList,
		}
		plan = append(plan, p)
	}

	// Compute plan for resources contained in spec annotations
	stateResources := state.Flatten().GroupByMigProfile()
	for migProfile, annotations := range desired.GroupByMigProfile() {
		totalDesiredQuantity := 0
		for _, a := range annotations {
			totalDesiredQuantity += a.Quantity
		}

		actualResources := stateResources[migProfile]
		if actualResources == nil {
			actualResources = make(migtypes.MigDeviceResourceList, 0)
		}

		plan = append(
			plan,
			migProfilePlan{
				migProfile:      migProfile.Name,
				gpuIndex:        migProfile.GpuIndex,
				desiredQuantity: totalDesiredQuantity,
				actualResources: actualResources,
			},
		)
	}

	return plan
}

func getResourcesNotIncludedInSpec(state types.MigState, specAnnotations types.GPUSpecAnnotationList) migtypes.MigDeviceResourceList {
	lookup := specAnnotations.GroupByGpuIndex()

	updatedState := state
	for gpuIndex, annotations := range lookup {
		migProfiles := make([]string, 0)
		for _, a := range annotations {
			migProfiles = append(migProfiles, a.GetMigProfileName())
		}
		updatedState = updatedState.WithoutMigProfiles(gpuIndex, migProfiles)
	}

	return updatedState.Flatten()
}
