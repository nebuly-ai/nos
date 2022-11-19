/*
 * Copyright 2022 Nebuly.ai
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package plan

import (
	"github.com/google/go-cmp/cmp"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
)

type MigConfigPlan struct {
	DeleteOperations DeleteOperationList
	CreateOperations CreateOperationList
}

func NewMigConfigPlan(state MigState, desired mig.GPUSpecAnnotationList) MigConfigPlan {
	plan := MigConfigPlan{
		DeleteOperations: make(DeleteOperationList, 0),
		CreateOperations: make(CreateOperationList, 0),
	}

	// Get resources present in current state which MIG profile is not included in spec
	for _, resourceList := range getResourcesNotIncludedInSpec(state, desired).GroupByMigProfile() {
		op := DeleteOperation{Resources: resourceList}
		plan.addDeleteOp(op)
	}

	// Compute plan for resources contained in spec annotations
	stateResourcesByGpu := state.Flatten().SortByDeviceId().GroupByGpuIndex()
	for gpuIndex, gpuAnnotations := range desired.GroupByGpuIndex() {
		gpuStateResources := stateResourcesByGpu[gpuIndex].GroupByMigProfile()
		for migProfile, migProfileAnnotations := range gpuAnnotations.GroupByMigProfile() {
			// init actual resources of current GPU and current MIG profile
			actualMigProfileResources := gpuStateResources[migProfile]
			if actualMigProfileResources == nil {
				actualMigProfileResources = make(mig.DeviceResourceList, 0)
			}

			// compute total desired quantity
			totalDesiredQuantity := 0
			for _, a := range migProfileAnnotations {
				totalDesiredQuantity += a.Quantity
			}

			diff := totalDesiredQuantity - len(actualMigProfileResources)
			if diff > 0 {
				// create missing MIG profiles and delete and re-create possible existing *free* resources
				// corresponding to the same MIG profile, so that when applying the create operations the number
				// of possible MIG permutations to try is larger
				freeResources := actualMigProfileResources.GetFree()
				if len(freeResources) > 0 {
					plan.addDeleteOp(DeleteOperation{Resources: freeResources})
					for freeProfile, freeProfileResources := range freeResources.GroupByMigProfile() {
						plan.addCreateOp(CreateOperation{MigProfile: freeProfile, Quantity: len(freeProfileResources)})
					}
				}
				plan.addCreateOp(CreateOperation{MigProfile: migProfile, Quantity: diff})
			}
			if diff < 0 {
				toDelete := extractCandidatesForDeletion(actualMigProfileResources, util.Abs(diff))
				op := DeleteOperation{Resources: toDelete}
				plan.addDeleteOp(op)
			}
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

func (p *MigConfigPlan) Equal(other *MigConfigPlan) bool {
	if other == nil || p == nil {
		return p == other
	}
	if !cmp.Equal(other.DeleteOperations, p.DeleteOperations) {
		return false
	}
	if !cmp.Equal(other.CreateOperations, p.CreateOperations) {
		return false
	}
	return true
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
