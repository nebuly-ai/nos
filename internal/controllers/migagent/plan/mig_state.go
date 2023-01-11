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
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/mig"
	"github.com/nebuly-ai/nos/pkg/util"
)

// MigState represents the current state in terms of MIG resources of each GPU (which index is stored as key
// in the map)
type MigState map[int]gpu.DeviceList

func NewMigState(resources gpu.DeviceList) MigState {
	res := make(MigState)
	for _, r := range resources {
		if res[r.GpuIndex] == nil {
			res[r.GpuIndex] = make([]gpu.Device, 0)
		}
		res[r.GpuIndex] = append(res[r.GpuIndex], r)
	}
	return res
}

func (s MigState) Matches(specAnnotations gpu.SpecAnnotationList) bool {
	getKey := func(name mig.ProfileName, gpuIndex int) string {
		return fmt.Sprintf("%d-%s", gpuIndex, name)
	}

	specGpuIndexWithMigProfileQuantities := make(map[string]int)
	for _, a := range specAnnotations {
		key := getKey(mig.ProfileName(a.ProfileName), a.Index)
		specGpuIndexWithMigProfileQuantities[key] += a.Quantity
	}

	stateGpuIndexWithMigProfileQuantities := make(map[string]int)
	groupedBy := s.Flatten().GroupBy(func(r gpu.Device) string {
		return getKey(mig.GetMigProfileName(r), r.GpuIndex)
	})
	for k, v := range groupedBy {
		stateGpuIndexWithMigProfileQuantities[k] = len(v)
	}

	return cmp.Equal(specGpuIndexWithMigProfileQuantities, stateGpuIndexWithMigProfileQuantities)
}

func (s MigState) Flatten() gpu.DeviceList {
	allResources := make(gpu.DeviceList, 0)
	for _, r := range s {
		allResources = append(allResources, r...)
	}
	return allResources
}

func (s MigState) DeepCopy() MigState {
	return NewMigState(s.Flatten())
}

// WithoutMigProfiles returns the state obtained after removing all the resources matching the MIG profiles
// on the GPU index provided as inputs
func (s MigState) WithoutMigProfiles(gpuIndex int, migProfiles []mig.ProfileName) MigState {
	res := s.DeepCopy()
	res[gpuIndex] = make(gpu.DeviceList, 0)
	for _, r := range s[gpuIndex] {
		if !util.InSlice(mig.GetMigProfileName(r), migProfiles) {
			res[gpuIndex] = append(res[gpuIndex], r)
		}
	}
	return res
}
