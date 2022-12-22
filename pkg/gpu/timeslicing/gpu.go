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

package timeslicing

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
	"sort"
)

type GPU struct {
	Model        gpu.Model
	Index        int
	MemoryGB     int
	UsedProfiles map[ProfileName]int
	FreeProfiles map[ProfileName]int
}

func NewFullGPU(model gpu.Model, index int, memoryGB int) GPU {
	return GPU{
		Model:        model,
		Index:        index,
		MemoryGB:     memoryGB,
		UsedProfiles: make(map[ProfileName]int),
		FreeProfiles: make(map[ProfileName]int),
	}
}

func NewGPU(model gpu.Model, index int, memoryGB int, usedProfiles, freeProfiles map[ProfileName]int) (GPU, error) {
	g := GPU{
		Model:        model,
		Index:        index,
		MemoryGB:     memoryGB,
		UsedProfiles: usedProfiles,
		FreeProfiles: freeProfiles,
	}
	if err := g.Validate(); err != nil {
		return GPU{}, err
	}
	return g, nil
}

func NewGpuOrPanic(model gpu.Model, index int, memoryGB int, usedProfiles, freeProfiles map[ProfileName]int) GPU {
	g, err := NewGPU(model, index, memoryGB, usedProfiles, freeProfiles)
	if err != nil {
		panic(err)
	}
	return g
}

func (g *GPU) Validate() error {
	var totalMemoryGB int
	for p, q := range g.UsedProfiles {
		mem := p.GetMemorySizeGB()
		if mem < MinSliceMemoryGB {
			return fmt.Errorf(
				"min allowed slice size is %dGB, but profile %s has %dGB",
				MinSliceMemoryGB,
				p,
				mem,
			)
		}
		totalMemoryGB += mem * q
	}
	for p, q := range g.FreeProfiles {
		mem := p.GetMemorySizeGB()
		if mem < MinSliceMemoryGB {
			return fmt.Errorf(
				"min allowed slice size is %dGB, but profile %s has %dGB",
				MinSliceMemoryGB,
				p,
				mem,
			)
		}
		totalMemoryGB += mem * q
	}
	if totalMemoryGB > g.MemoryGB {
		return fmt.Errorf("total memory of profiles (%d) exceeds GPU memory (%d)", totalMemoryGB, g.MemoryGB)
	}
	return nil
}

func (g *GPU) GetGeometry() gpu.Geometry {
	geometry := make(gpu.Geometry)
	for p, q := range g.UsedProfiles {
		geometry[p] += q
	}
	for p, q := range g.FreeProfiles {
		geometry[p] += q
	}
	return geometry
}

func (g *GPU) Clone() GPU {
	cloned := GPU{
		Model:    g.Model,
		Index:    g.Index,
		MemoryGB: g.MemoryGB,
	}
	if g.UsedProfiles != nil {
		cloned.UsedProfiles = make(map[ProfileName]int)
		for k, v := range g.UsedProfiles {
			cloned.UsedProfiles[k] = v
		}
	}
	if g.FreeProfiles != nil {
		cloned.FreeProfiles = make(map[ProfileName]int)
		for k, v := range g.FreeProfiles {
			cloned.FreeProfiles[k] = v
		}
	}
	return cloned
}

func (g *GPU) HasFreeCapacity() bool {
	if len(g.FreeProfiles) > 0 {
		return true
	}
	return g.canCreateMoreSlices()
}

// AddPod adds a Pod to the GPU by updating the free and used slices according to the ones
// requested by the Pod.
//
// AddPod returns an error if the GPU does not have enough free slices for the Pod.
func (g *GPU) AddPod(pod v1.Pod) error {
	for r, q := range GetRequestedProfiles(pod) {
		if g.FreeProfiles[r] < q {
			return fmt.Errorf(
				"not enough free slices (pod requests %d %s, but GPU only has %d)",
				q,
				r,
				g.FreeProfiles[r],
			)
		}
		g.FreeProfiles[r] -= q
		g.UsedProfiles[r] += q
	}
	return nil
}

// UpdateGeometryFor tries to update the geometry of the GPU in order to create the highest possible number of required
// slices provided as argument, without deleting any of the used slices.
//
// The method returns true if the GPU geometry gets updated, false otherwise.
func (g *GPU) UpdateGeometryFor(slices map[gpu.Slice]int) bool {
	var missingSlices = g.getMissingSlices(slices)

	// If the GPU already provides the required slices then there's nothing to do
	if len(missingSlices) == 0 {
		return false
	}

	var updated bool
	var originalFreeProfiles = util.CopyMap(g.FreeProfiles)

	// Sort missing slices by size (smaller first)
	sortedMissingSlices := make([]gpu.Slice, 0, len(missingSlices))
	for slice := range missingSlices {
		sortedMissingSlices = append(sortedMissingSlices, slice)
	}
	sort.SliceStable(sortedMissingSlices, func(i, j int) bool {
		first := sortedMissingSlices[i].(ProfileName)
		second := sortedMissingSlices[j].(ProfileName)
		return first.GetMemorySizeGB() < second.GetMemorySizeGB()
	})

	for _, s := range sortedMissingSlices {
		missingProfile := s.(ProfileName)
		missingProfileMemory := missingProfile.GetMemorySizeGB()
		// first try to create the missing slices by using spare capacity
		if g.canCreateMoreSlices() {
			q := missingSlices[missingProfile]
			for i := 0; i < q; i++ {
				if err := g.createSlice(missingProfileMemory); err != nil {
					break
				}
				missingSlices[missingProfile]--
				updated = true
			}
		}
		// then try to free up space by deleting the initial free slices
		q := missingSlices[missingProfile]
		for k := range originalFreeProfiles {
			delete(g.FreeProfiles, k)
		}
		for i := 0; i < q; i++ {
			if !g.canCreateMoreSlices() {
				break
			}
			if err := g.createSlice(missingProfileMemory); err != nil {
				break
			}
			missingSlices[missingProfile]--
			updated = true
		}
		// try to restore the original free slices
		for k, v := range originalFreeProfiles {
			_ = g.createSlices(k.GetMemorySizeGB(), v)
		}
	}

	return updated
}

func (g *GPU) getMissingSlices(required map[gpu.Slice]int) map[gpu.Slice]int {
	var missingSlices = make(map[gpu.Slice]int)
	for requiredSlice, requiredQuantity := range required {
		requiredProfile := requiredSlice.(ProfileName)
		diff := requiredQuantity - g.FreeProfiles[requiredProfile]
		if diff > 0 {
			missingSlices[requiredProfile] = diff
		}
	}
	return missingSlices
}

func (g *GPU) createSlice(sizeGb int) error {
	return g.createSlices(sizeGb, 1)
}

func (g *GPU) createSlices(sizeGb, num int) error {
	totSlicesMemory := g.getTotSlicesMemory()
	spareMemory := g.MemoryGB - totSlicesMemory
	if spareMemory < sizeGb*num {
		return fmt.Errorf("not enough spare memory to create %d slices of size %dGB", num, sizeGb)
	}
	sliceProfile := NewProfile(sizeGb)
	g.FreeProfiles[sliceProfile] += num
	return nil
}

// canCreateMoreSlices returns true if the GPU has enough free space to create more slices, false otherwise
func (g *GPU) canCreateMoreSlices() bool {
	totSlicesMemory := g.getTotSlicesMemory()
	spareMemory := g.MemoryGB - totSlicesMemory
	return spareMemory >= MinSliceMemoryGB
}

func (g *GPU) getTotSlicesMemory() int {
	var totSlicesMemory int
	for p, q := range g.UsedProfiles {
		totSlicesMemory += p.GetMemorySizeGB() * q
	}
	for p, q := range g.FreeProfiles {
		totSlicesMemory += p.GetMemorySizeGB() * q
	}
	return totSlicesMemory
}
