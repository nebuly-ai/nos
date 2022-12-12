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

package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
	"strconv"
)

type Node struct {
	Name string
	GPUs []GPU
}

// NewNode creates a new MIG Node starting from the node provided as argument.
//
// The function constructs the MIG GPUs of the provided node using both the n8s.nebuly.ai MIG status annotations
// and the labels exposed by the NVIDIA gpu-feature-discovery tool. Specifically, the following labels are used:
// - GPU product ("nvidia.com/gpu.product")
// - GPU count ("nvidia.com/gpu.count")
//
// If the v1.Node provided as arg does not have the GPU Product label, returned node will not contain any mig.GPU.
func NewNode(n v1.Node) (Node, error) {
	gpuModel, ok := getGPUModel(n)
	if !ok {
		return Node{Name: n.Name, GPUs: make([]GPU, 0)}, nil
	}
	gpuCount, _ := getGPUCount(n)

	gpus, err := extractGPUs(n, gpuModel, gpuCount)
	if err != nil {
		return Node{}, err
	}
	return Node{Name: n.Name, GPUs: gpus}, nil
}

func extractGPUs(node v1.Node, gpuModel gpu.Model, gpuCount int) ([]GPU, error) {
	result := make([]GPU, 0)

	// Init GPUs from annotation
	statusAnnotations, _ := GetGPUAnnotationsFromNode(node)
	for gpuIndex, gpuAnnotations := range statusAnnotations.GroupByGpuIndex() {
		usedMigDevices := make(map[ProfileName]int)
		freeMigDevices := make(map[ProfileName]int)
		for _, a := range gpuAnnotations {
			if a.IsUsed() {
				usedMigDevices[a.GetMigProfileName()] = a.Quantity
			}
			if a.IsFree() {
				freeMigDevices[a.GetMigProfileName()] = a.Quantity
			}
		}
		g, err := NewGPU(gpuModel, gpuIndex, usedMigDevices, freeMigDevices)
		if err != nil {
			return nil, err
		}
		result = append(result, g)
	}

	// Add missing GPUs not included in node annotations (e.g. GPUs with MIG enabled but without any MIG device)
	nGpus := len(result)
	for i := nGpus; i < gpuCount; i++ {
		g, err := NewGPU(gpuModel, i, make(map[ProfileName]int), make(map[ProfileName]int))
		if err != nil {
			return nil, err
		}
		result = append(result, g)
	}

	return result, nil
}

func getGPUModel(node v1.Node) (gpu.Model, bool) {
	if val, ok := node.Labels[constant.LabelNvidiaProduct]; ok {
		return gpu.Model(val), true
	}
	return "", false
}

func getGPUCount(node v1.Node) (int, bool) {
	if val, ok := node.Labels[constant.LabelNvidiaCount]; ok {
		if valAsInt, err := strconv.Atoi(val); err == nil {
			return valAsInt, true
		}
	}
	return 0, false
}

// UpdateGeometryFor tries to update the MIG geometry of each single GPU of the node in order to create the MIG profiles
// provided as argument.
//
// The method returns true if it updates the MIG geometry of any GPU, false otherwise.
func (n *Node) UpdateGeometryFor(profiles map[ProfileName]int) bool {
	// If there are no GPUs, then there's nothing to do
	if len(n.GPUs) == 0 {
		return false
	}
	if len(profiles) == 0 {
		return false
	}

	var requiredProfiles = util.CopyMap(profiles)
	var anyGpuUpdated bool

	for _, g := range n.GPUs {
		updated := g.UpdateGeometryFor(requiredProfiles)
		anyGpuUpdated = anyGpuUpdated || updated
		for profile, quantity := range g.GetFreeMigDevices() {
			requiredProfiles[profile] -= quantity
			if requiredProfiles[profile] <= 0 {
				delete(requiredProfiles, profile)
			}
		}
	}

	return anyGpuUpdated
}

// GetGeometry returns the overall MIG geometry of the node, which corresponds to the sum of the MIG geometry of all
// the GPUs present in the Node.
func (n *Node) GetGeometry() Geometry {
	res := make(Geometry)
	for _, g := range n.GPUs {
		for p, q := range g.GetGeometry() {
			res[p] += q
		}
	}
	return res
}

// HasFreeMigCapacity returns true if the Node has at least one GPU with free MIG capacity, namely it either has a
// free MIG device or its allowed MIG geometries allow to create at least one more MIG device.
func (n *Node) HasFreeMigCapacity() bool {
	if len(n.GPUs) == 0 {
		return false
	}
	for _, g := range n.GPUs {
		if g.HasFreeMigDevices() {
			return true
		}
		// If the GPU is not in a valid Geometry it means that we can create new free MIG devices
		// by applying any valid MIG geometry
		if !g.AllowsGeometry(g.GetGeometry()) {
			return true
		}
	}
	return false
}

// AddPod adds a Pod to the node by updating the free and used MIG devices of the Node GPUs according to the
// MIG requested required by the Pod.
//
// AddPod returns an error if the node does not have any GPU providing enough free MIG resources for the Pod.
func (n *Node) AddPod(pod v1.Pod) error {
	for _, g := range n.GPUs {
		if err := g.AddPod(pod); err == nil {
			return nil
		}
	}
	return fmt.Errorf("not enough free MIG devices")
}

func (n *Node) Clone() Node {
	cloned := Node{
		Name: n.Name,
		GPUs: make([]GPU, len(n.GPUs)),
	}
	for i := range n.GPUs {
		cloned.GPUs[i] = n.GPUs[i].Clone()
	}
	return cloned
}
