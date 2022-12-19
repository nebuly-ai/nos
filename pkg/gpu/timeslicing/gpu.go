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
	Replicas     int
	MemoryGB     int
	usedMemoryGB int
}

func (g *GPU) Clone() GPU {
	return GPU{
		Model:    g.Model,
		Index:    g.Index,
		Replicas: g.Replicas,
		MemoryGB: g.MemoryGB,
	}
}

func (g *GPU) ReserveMemory(memoryGB int) error {
	if g.usedMemoryGB+memoryGB > g.MemoryGB {
		return fmt.Errorf("not enough memory: requested %d, available %d", memoryGB, g.MemoryGB-g.usedMemoryGB)
	}
	g.usedMemoryGB += memoryGB
	return nil
}

func (g *GPU) GetSliceSize() {

}
