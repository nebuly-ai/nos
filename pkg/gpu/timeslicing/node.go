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
	deviceplugin "github.com/NVIDIA/k8s-device-plugin/api/config/v1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	v1 "k8s.io/api/core/v1"
	"strconv"
)

type Node struct {
	Name string
	GPUs []GPU
}

func NewNode(n v1.Node, config deviceplugin.TimeSlicing) (Node, error) {
	// Extract common GPU info from node labels
	gpuModel, err := gpu.GetModel(n)
	if err != nil {
		return Node{}, err
	}
	gpuCount, err := gpu.GetCount(n)
	if err != nil {
		return Node{}, err
	}
	memoryGB, err := gpu.GetMemoryGB(n)
	if err != nil {
		return Node{}, err
	}

	// Init GPUs from nvidia device plugin time-slicing config
	gpus := make([]GPU, 0)
	for _, r := range config.Resources {
		for _, d := range r.Devices.List {
			if !d.IsGPUIndex() {
				return Node{}, fmt.Errorf("time-slicing config should use GPU indexes, found: %s", d)
			}
			index, _ := strconv.Atoi(string(d))
			g := GPU{
				Model:    gpuModel,
				Index:    index,
				MemoryGB: memoryGB,
				Replicas: r.Replicas,
			}
			gpus = append(gpus, g)
		}
	}

	// Add missing GPUs not included in time-slicing config
	for i := len(gpus); i < gpuCount; i++ {
		g := GPU{
			Model:    gpuModel,
			Index:    i,
			MemoryGB: memoryGB,
			Replicas: 1,
		}
		gpus = append(gpus, g)
	}

	return Node{
		Name: n.Name,
		GPUs: gpus,
	}, nil
}

func (n *Node) Clone() Node {
	gpus := make([]GPU, len(n.GPUs))
	for i, g := range n.GPUs {
		gpus[i] = g.Clone()
	}
	return Node{
		Name: n.Name,
		GPUs: gpus,
	}
}
