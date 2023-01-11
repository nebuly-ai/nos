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
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/mig"
	"github.com/nebuly-ai/nos/pkg/util"
)

type MigConfigPlan struct {
	DeleteOperations DeleteOperationList
	CreateOperations CreateOperationList
}

func NewMigConfigPlan(state MigState, desired gpu.SpecAnnotationList) MigConfigPlan {
	plan := MigConfigPlan{
		DeleteOperations: make(DeleteOperationList, 0),
		CreateOperations: make(CreateOperationList, 0),
	}

	// Delete resources not included in spec
	for _, resourceList := range mig.GroupDevicesByMigProfile(getResourcesNotIncludedInSpec(state, desired)) {
		op := DeleteOperation{Resources: resourceList}
		plan.addDeleteOp(op)
	}

	// Compute plan for resources contained in spec annotations
	stateResourcesByGpu := state.Flatten().SortByDeviceId().GroupByGpuIndex()
	for gpuIndex, gpuAnnotations := range desired.GroupByGpuIndex() {
		gpuStateResources := mig.GroupDevicesByMigProfile(stateResourcesByGpu[gpuIndex])
		nCreateOp := 0
		for migProfile, migProfileAnnotations := range mig.GroupSpecAnnotationsByMigProfile(gpuAnnotations) {
			// init actual resources of current GPU and current MIG profile
			actualMigProfileResources := gpuStateResources[migProfile]
			if actualMigProfileResources == nil {
				actualMigProfileResources = make(gpu.DeviceList, 0)
			}

			// compute total desired quantity
			totalDesiredQuantity := 0
			for _, a := range migProfileAnnotations {
				totalDesiredQuantity += a.Quantity
			}

			diff := totalDesiredQuantity - len(actualMigProfileResources)
			if diff > 0 {
				plan.addCreateOp(CreateOperation{MigProfile: migProfile, Quantity: diff})
				nCreateOp++
			}
			if diff < 0 {
				toDelete := extractCandidatesForDeletion(actualMigProfileResources, util.Abs(diff))
				op := DeleteOperation{Resources: toDelete}
				plan.addDeleteOp(op)
			}
		}

		// no create operations on this GPU, we don't need to clean up free devices
		if nCreateOp == 0 {
			continue
		}

		// if there's any create op on the GPU, then re-create existing *free* resources so that
		// when applying the create operations the number of possible MIG permutations to try is larger
		resourcesToRecreate := extractResourcesToRecreate(stateResourcesByGpu[gpuIndex], plan)
		if len(resourcesToRecreate) > 0 {
			// delete free resources not already included in plan
			plan.addDeleteOp(DeleteOperation{Resources: resourcesToRecreate})
			// re-create free resources
			for profile, resources := range mig.GroupDevicesByMigProfile(resourcesToRecreate) {
				plan.addCreateOp(CreateOperation{MigProfile: profile, Quantity: len(resources)})
			}
		}
	}

	return plan
}

func extractResourcesToRecreate(resources gpu.DeviceList, currentPlan MigConfigPlan) gpu.DeviceList {
	// lookup
	alreadyToBeDeletedLookup := make(map[string]gpu.Device)
	for _, r := range currentPlan.getResourcesToDelete() {
		alreadyToBeDeletedLookup[r.DeviceId] = r
	}
	// extract free resources not already included in plan delete operations
	resourcesToRecreate := make(gpu.DeviceList, 0)
	for _, r := range resources.GetFree() {
		if _, toBeDeleted := alreadyToBeDeletedLookup[r.DeviceId]; !toBeDeleted {
			resourcesToRecreate = append(resourcesToRecreate, r)
		}
	}

	return resourcesToRecreate
}

func extractCandidatesForDeletion(resources gpu.DeviceList, nToDelete int) gpu.DeviceList {
	deleteCandidates := make(gpu.DeviceList, 0)
	// add free devices first
	for _, r := range resources {
		if r.IsFree() {
			deleteCandidates = append(deleteCandidates, r)
		}
		if len(deleteCandidates) == util.Abs(nToDelete) {
			break
		}
	}
	// if free devices are not enough, add used resources too
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

func (p *MigConfigPlan) getResourcesToDelete() gpu.DeviceList {
	resources := make(gpu.DeviceList, 0)
	for _, o := range p.DeleteOperations {
		resources = append(resources, o.Resources...)
	}
	return resources
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

func getResourcesNotIncludedInSpec(state MigState, specAnnotations gpu.SpecAnnotationList) gpu.DeviceList {
	lookup := specAnnotations.GroupByGpuIndex()

	updatedState := state
	for gpuIndex, annotations := range lookup {
		migProfiles := make([]mig.ProfileName, 0)
		for _, a := range annotations {
			migProfiles = append(migProfiles, mig.ProfileName(a.ProfileName))
		}
		updatedState = updatedState.WithoutMigProfiles(gpuIndex, migProfiles)
	}

	return updatedState.Flatten()
}
