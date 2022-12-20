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
			freeProfiles,
			usedProfiles,
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
	//TODO implement me
	panic("implement me")
}

func (n *Node) GetName() string {
	//TODO implement me
	panic("implement me")
}

func (n *Node) Geometry() map[gpu.Slice]int {
	//TODO implement me
	panic("implement me")
}

func (n *Node) NodeInfo() framework.NodeInfo {
	//TODO implement me
	panic("implement me")
}

func (n *Node) AddPod(pod v1.Pod) error {
	//TODO implement me
	panic("implement me")
}

func (n *Node) HasFreeCapacity() bool {
	//TODO implement me
	panic("implement me")
}
