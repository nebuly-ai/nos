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
)

type GPU struct {
	Model        gpu.Model
	Index        int
	MemoryGB     int
	freeProfiles map[ProfileName]int
	usedProfiles map[ProfileName]int
}

func NewFullGPU(model gpu.Model, index int, memoryGB int) GPU {
	return GPU{
		Model:        model,
		Index:        index,
		MemoryGB:     memoryGB,
		freeProfiles: make(map[ProfileName]int),
		usedProfiles: make(map[ProfileName]int),
	}
}

func NewGPU(model gpu.Model, index int, memoryGB int, usedProfiles, freeProfiles map[ProfileName]int) (GPU, error) {
	g := GPU{
		Model:        model,
		Index:        index,
		MemoryGB:     memoryGB,
		freeProfiles: freeProfiles,
		usedProfiles: usedProfiles,
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
	for p, q := range g.usedProfiles {
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
	for p, q := range g.freeProfiles {
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

func (g *GPU) Clone() GPU {
	return GPU{
		Model:    g.Model,
		Index:    g.Index,
		MemoryGB: g.MemoryGB,
	}
}
