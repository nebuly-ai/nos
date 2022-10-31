package plan

import (
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
)

type CreateOperation struct {
	MigProfile mig.Profile
	Quantity   int
}

type DeleteOperation struct {
	// Resources are the possible device resources that can be deleted. Must be >= Quantity.
	Resources mig.DeviceResourceList
	// Quantity is the amount of resources that need to be deleted. Must be <= len(Resources).
	Quantity int
}

func (o DeleteOperation) GetMigProfileName() mig.ProfileName {
	if len(o.Resources) > 0 {
		return o.Resources[0].GetMigProfileName()
	}
	return ""
}

type MigConfigPlan struct {
	DeleteOperations []DeleteOperation
	CreateOperations []CreateOperation
}

func NewMigConfigPlan(state MigState, desired mig.GPUSpecAnnotationList) MigConfigPlan {
	plan := MigConfigPlan{}

	// Get resources present in current state which MIG profile is not included in spec
	for _, resourceList := range getResourcesNotIncludedInSpec(state, desired).GroupByMigProfile() {
		op := DeleteOperation{
			Resources: resourceList,
			Quantity:  len(resourceList), // we want all of these resources to be deleted
		}
		plan.addDeleteOp(op)
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
			actualResources = make(mig.DeviceResourceList, 0)
		}

		diff := totalDesiredQuantity - len(actualResources)
		if diff > 0 {
			op := CreateOperation{
				MigProfile: migProfile,
				Quantity:   diff,
			}
			plan.addCreateOp(op)
		}
		if diff < 0 {
			op := DeleteOperation{
				Quantity:  util.Abs(diff),
				Resources: actualResources,
			}
			plan.addDeleteOp(op)
		}
	}

	return plan
}

func (p *MigConfigPlan) addDeleteOp(op DeleteOperation) {
	p.DeleteOperations = append(p.DeleteOperations, op)
}

func (p *MigConfigPlan) addCreateOp(op CreateOperation) {
	p.CreateOperations = append(p.CreateOperations, op)
}

func (p *MigConfigPlan) IsEmpty() bool {
	return len(p.DeleteOperations) == 0 && len(p.CreateOperations) == 0
}

func getResourcesNotIncludedInSpec(state MigState, specAnnotations mig.GPUSpecAnnotationList) mig.DeviceResourceList {
	lookup := specAnnotations.GroupByGpuIndex()

	updatedState := state
	for gpuIndex, annotations := range lookup {
		migProfiles := make([]mig.ProfileName, 0)
		for _, a := range annotations {
			migProfiles = append(migProfiles, a.GetMigProfileName())
		}
		updatedState = updatedState.WithoutMigProfiles(gpuIndex, migProfiles)
	}

	return updatedState.Flatten()
}
