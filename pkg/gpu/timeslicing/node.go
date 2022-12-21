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
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type Node struct {
	Name     string
	GPUs     []GPU
	nodeInfo framework.NodeInfo
}

func NewNode(n framework.NodeInfo) (Node, error) {
	if n.Node() == nil {
		return Node{}, fmt.Errorf("node is nil")
	}
	node := *n.Node()
	gpus, err := extractGPUs(node)
	if err != nil {
		return Node{}, err
	}
	return Node{
		Name:     node.Name,
		GPUs:     gpus,
		nodeInfo: n,
	}, nil
}

func extractGPUs(n v1.Node) ([]GPU, error) {
	// Extract common GPU info from node labels
	gpuModel, err := gpu.GetModel(n)
	if err != nil {
		return nil, err
	}
	gpuCount, err := gpu.GetCount(n)
	if err != nil {
		return nil, err
	}
	gpuMemoryGB, err := gpu.GetMemoryGB(n)
	if err != nil {
		return nil, err
	}

	result := make([]GPU, 0)

	// Init GPUs from annotation
	statusAnnotations, _ := gpu.ParseNodeAnnotations(n)
	for gpuIndex, gpuAnnotations := range statusAnnotations.GroupByGpuIndex() {
		usedProfiles := make(map[ProfileName]int)
		freeProfiles := make(map[ProfileName]int)
		for _, a := range gpuAnnotations {
			profileName := ProfileName(a.ProfileName)
			if a.IsUsed() {
				usedProfiles[profileName] = a.Quantity
			}
			if a.IsFree() {
				freeProfiles[profileName] = a.Quantity
			}
		}
		g, err := NewGPU(
			gpuModel,
			gpuIndex,
			gpuMemoryGB,
			usedProfiles,
			freeProfiles,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, g)
	}

	// Add missing GPUs not included in node annotations
	// (e.g. GPUs enabled but without any time-slicing replica/profile)
	nGpus := len(result)
	for i := nGpus; i < gpuCount; i++ {
		g := NewFullGPU(
			gpuModel,
			i,
			gpuMemoryGB,
		)
		result = append(result, g)
	}

	return result, nil
}

func (n *Node) Clone() interface{} {
	gpus := make([]GPU, len(n.GPUs))
	for i, g := range n.GPUs {
		gpus[i] = g.Clone()
	}
	return &Node{
		Name: n.Name,
		GPUs: gpus,
	}
}

func (n *Node) UpdateGeometryFor(slices map[gpu.Slice]int) (bool, error) {
	// If there are no GPUs, then there's nothing to do
	if len(n.GPUs) == 0 {
		return false, nil
	}
	if len(slices) == 0 {
		return false, nil
	}

	// Copy slices
	var requiredSlices = make(map[gpu.Slice]int, len(slices))
	for k, v := range slices {
		requiredSlices[k] = v
	}

	var anyGpuUpdated bool
	for _, g := range n.GPUs {
		updated := g.UpdateGeometryFor(requiredSlices)
		anyGpuUpdated = anyGpuUpdated || updated
		for profile, quantity := range g.FreeProfiles {
			requiredSlices[profile] -= quantity
			if requiredSlices[profile] <= 0 {
				delete(requiredSlices, profile)
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

	// Set all non-time-slicing scalar resources
	for r, v := range n.nodeInfo.Allocatable.ScalarResources {
		if !IsTimeSlicingResource(r) {
			res[r] = v
		}
	}
	// Set time-slicing scalar resources
	for r, v := range n.Geometry() {
		resource := r.(ProfileName).AsResourceName()
		res[resource] = int64(v)
	}

	return res
}

func (n *Node) GetName() string {
	return n.Name
}

// Geometry returns the overall geometry of the node, which corresponds to the sum of the geometries of all
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

// AddPod adds a Pod to the node by updating the free and used time-slicing slices of the Node GPUs according to the
// slices requested by the Pod.
//
// AddPod returns an error if the node does not have any GPU providing enough free slices resources for the Pod.
func (n *Node) AddPod(pod v1.Pod) error {
	for _, g := range n.GPUs {
		if err := g.AddPod(pod); err == nil {
			nodeInfo := n.NodeInfo()
			nodeInfo.AddPod(&pod)
			return nil
		}
	}
	return fmt.Errorf("not enough free GPU slices")
}

// HasFreeCapacity returns true if any of the GPUs of the node has enough free capacity for hosting more pods.
func (n *Node) HasFreeCapacity() bool {
	for _, g := range n.GPUs {
		if g.HasFreeCapacity() {
			return true
		}
	}
	return false
}
