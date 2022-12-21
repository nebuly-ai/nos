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
	v1 "k8s.io/api/core/v1"
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
	return GPU{
		Model:    g.Model,
		Index:    g.Index,
		MemoryGB: g.MemoryGB,
	}
}

func (g *GPU) HasFreeCapacity() bool {
	if len(g.FreeProfiles) > 0 {
		return true
	}
	// Check if there is space to create more slices
	var slicesMemory int
	for p, q := range g.UsedProfiles {
		slicesMemory += p.GetMemorySizeGB() * q
	}
	freeMemory := g.MemoryGB - slicesMemory
	return freeMemory >= MinSliceMemoryGB
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

func (g *GPU) UpdateGeometryFor(slices map[gpu.Slice]int) bool {
	return false
}
