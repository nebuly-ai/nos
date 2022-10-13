package migagent

import "github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"

type migProfilePlan struct {
	migProfile      string
	gpuIndex        int
	desiredQuantity int
	actualResources []types.MigDeviceResource
}

type migConfigPlan []migProfilePlan

func (p migConfigPlan) summary() string {
	return ""
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
		p := migProfilePlan{
			migProfile:      migProfile.Name,
			gpuIndex:        migProfile.GpuIndex,
			desiredQuantity: totalDesiredQuantity,
			actualResources: stateResources[migProfile],
		}
		plan = append(plan, p)
	}

	return plan
}

func getResourcesNotIncludedInSpec(state types.MigState, specAnnotations types.GPUSpecAnnotationList) types.MigDeviceResourceList {
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
