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
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type Node struct {
	Name     string
	nodeInfo framework.NodeInfo
	GPUs     []GPU
}

// NewNode creates a new MIG Node starting from the node provided as argument.
//
// The function constructs the MIG GPUs of the provided node using both the n8s.nebuly.ai MIG status annotations
// and the labels exposed by the NVIDIA gpu-feature-discovery tool. Specifically, the following labels are used:
// - GPU product ("nvidia.com/gpu.product")
// - GPU count ("nvidia.com/gpu.count")
//
// If the v1.Node provided as arg does not have the GPU Product label, returned node will not contain any mig.GPU.
func NewNode(n framework.NodeInfo) (Node, error) {
	if n.Node() == nil {
		return Node{}, fmt.Errorf("node is nil")
	}
	node := *n.Node()
	gpuModel, err := gpu.GetModel(node)
	if err != nil {
		return Node{
			Name:     node.Name,
			GPUs:     make([]GPU, 0),
			nodeInfo: n,
		}, nil
	}
	gpuCount, _ := gpu.GetCount(node)

	gpus, err := extractGPUs(node, gpuModel, gpuCount)
	if err != nil {
		return Node{}, err
	}
	return Node{
		Name:     node.Name,
		GPUs:     gpus,
		nodeInfo: n,
	}, nil
}

func extractGPUs(node v1.Node, gpuModel gpu.Model, gpuCount int) ([]GPU, error) {
	result := make([]GPU, 0)

	// Init GPUs from annotation
	statusAnnotations, _ := gpu.ParseNodeAnnotations(node)
	for gpuIndex, gpuAnnotations := range statusAnnotations.GroupByGpuIndex() {
		usedMigDevices := make(map[ProfileName]int)
		freeMigDevices := make(map[ProfileName]int)
		for _, a := range gpuAnnotations {
			profileName := ProfileName(a.ProfileName)
			if a.IsUsed() {
				usedMigDevices[profileName] = a.Quantity
			}
			if a.IsFree() {
				freeMigDevices[profileName] = a.Quantity
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

func (n *Node) GetName() string {
	return n.Name
}

// Geometry returns the overall MIG geometry of the node, which corresponds to the sum of the MIG geometry of all
// the GPUs present in the Node.
func (n *Node) Geometry() map[gpu.Slice]int {
	res := make(map[gpu.Slice]int)
	for _, g := range n.GPUs {
		for p, q := range g.GetGeometry() {
			res[p] += q
		}
	}
	return res
}

func (n *Node) NodeInfo() framework.NodeInfo {
	return n.nodeInfo
}

// HasFreeCapacity returns true if the Node has at least one GPU with free MIG capacity, namely it either has a
// free MIG device or its allowed MIG geometries allow to create at least one more MIG device.
func (n *Node) HasFreeCapacity() bool {
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

// UpdateGeometryFor tries to update the MIG geometry of each single GPU of the node in order to create the MIG profiles
// provided as argument.
//
// The method returns true if it updates the MIG geometry of any GPU, false otherwise.
func (n *Node) UpdateGeometryFor(slices map[gpu.Slice]int) (bool, error) {
	// If there are no GPUs, then there's nothing to do
	if len(n.GPUs) == 0 {
		return false, nil
	}
	if len(slices) == 0 {
		return false, nil
	}

	// Copy slices
	var requiredProfiles = make(map[gpu.Slice]int, len(slices))
	for k, v := range slices {
		requiredProfiles[k] = v
	}
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

	// Update node info
	scalarResources := n.computeScalarResources()
	n.nodeInfo.Allocatable.ScalarResources = scalarResources

	return anyGpuUpdated, nil
}

func (n *Node) computeScalarResources() map[v1.ResourceName]int64 {
	res := make(map[v1.ResourceName]int64)

	// Set all non-MIG scalar resources
	for r, v := range n.nodeInfo.Allocatable.ScalarResources {
		if !IsNvidiaMigDevice(r) {
			res[r] = v
		}
	}
	// Set MIG scalar resources
	for r, v := range n.Geometry() {
		resource := r.(ProfileName).AsResourceName()
		res[resource] = int64(v)
	}

	return res
}

// AddPod adds a Pod to the node by updating the free and used MIG devices of the Node GPUs according to the
// MIG requested required by the Pod.
//
// AddPod returns an error if the node does not have any GPU providing enough free MIG resources for the Pod.
func (n *Node) AddPod(pod v1.Pod) error {
	for _, g := range n.GPUs {
		if err := g.AddPod(pod); err == nil {
			nodeInfo := n.NodeInfo()
			nodeInfo.AddPod(&pod)
			return nil
		}
	}
	return fmt.Errorf("not enough free MIG devices")
}

func (n *Node) Clone() interface{} {
	cloned := Node{
		Name:     n.GetName(),
		GPUs:     make([]GPU, len(n.GPUs)),
		nodeInfo: *n.nodeInfo.Clone(),
	}
	for i := range n.GPUs {
		cloned.GPUs[i] = n.GPUs[i].Clone()
	}
	return &cloned
}
