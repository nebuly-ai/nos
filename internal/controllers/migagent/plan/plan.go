package plan

import (
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
)

type MigConfigPlan struct {
	DeleteOperations DeleteOperationList
	CreateOperations CreateOperationList
}

func NewMigConfigPlan(state MigState, desired mig.GPUSpecAnnotationList) MigConfigPlan {
	plan := MigConfigPlan{}

	// Get resources present in current state which MIG profile is not included in spec
	for _, resourceList := range getResourcesNotIncludedInSpec(state, desired).GroupByMigProfile() {
		op := DeleteOperation{Resources: resourceList}
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

		//for _, res := range actualResources{
		//	if res in desired && res.Status == resource.StatusFree {
		//
		//	}
		//}

		diff := totalDesiredQuantity - len(actualResources)
		if diff > 0 {
			op := CreateOperation{
				MigProfile: migProfile,
				Quantity:   diff,
			}
			plan.addCreateOp(op)
		}
		if diff < 0 {
			toDelete := extractCandidatesForDeletion(actualResources, util.Abs(diff))
			op := DeleteOperation{Resources: toDelete}
			plan.addDeleteOp(op)
		}
	}

	return plan
}

func extractCandidatesForDeletion(resources mig.DeviceResourceList, nToDelete int) mig.DeviceResourceList {
	deleteCandidates := make(mig.DeviceResourceList, 0)
	// add free devices first
	for _, r := range resources {
		if r.IsFree() {
			deleteCandidates = append(deleteCandidates, r)
		}
		if len(deleteCandidates) == util.Abs(nToDelete) {
			break
		}
	}
	// if candidates are not enough, add not-free resources too
	if len(deleteCandidates) < nToDelete {
		for _, r := range resources {
			if !r.IsFree() {
				deleteCandidates = append(deleteCandidates, r)
			}
			if len(deleteCandidates) == util.Abs(nToDelete) {
				break
			}
		}
	}
	return deleteCandidates
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
