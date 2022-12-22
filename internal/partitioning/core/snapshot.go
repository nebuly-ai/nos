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

package core

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sort"
)

type snapshotData struct {
	nodes map[string]PartitionableNode
}

func (d snapshotData) clone() *snapshotData {
	res := snapshotData{
		nodes: make(map[string]PartitionableNode),
	}
	for k, v := range d.nodes {
		res.nodes[k] = v.Clone().(PartitionableNode)
	}
	return &res
}

type clusterSnapshot struct {
	data               *snapshotData
	forkedData         *snapshotData
	partitioner        PartitionCalculator
	profilesCalculator gpu.SliceCalculator
	profilesFilter     gpu.SliceFilter
}

func NewClusterSnapshot(
	nodes map[string]PartitionableNode,
	partitioner PartitionCalculator,
	sliceCalculator gpu.SliceCalculator,
	sliceFilter gpu.SliceFilter,
) Snapshot {
	data := snapshotData{nodes: nodes}
	return &clusterSnapshot{
		data:               &data,
		partitioner:        partitioner,
		profilesCalculator: sliceCalculator,
		profilesFilter:     sliceFilter,
	}
}

func (c *clusterSnapshot) GetPartitioningState() state.PartitioningState {
	partitioningState := make(map[string]state.NodePartitioning)
	for name, node := range c.GetNodes() {
		partitioningState[name] = c.partitioner.GetPartitioning(node)
	}
	return partitioningState
}

func (c *clusterSnapshot) GetLackingSlices(pod v1.Pod) map[gpu.Slice]int {
	return c.profilesFilter.ExtractSlices(c.getLackingResources(pod).ScalarResources)
}

func (c *clusterSnapshot) getData() *snapshotData {
	if c.forkedData != nil {
		return c.forkedData
	}
	return c.data
}

func (c *clusterSnapshot) Fork() error {
	if c.forkedData != nil {
		return fmt.Errorf("snapshot already forked")
	}
	c.forkedData = c.getData().clone()
	return nil
}

func (c *clusterSnapshot) Clone() Snapshot {
	cloned := &clusterSnapshot{
		partitioner:        c.partitioner,
		profilesCalculator: c.profilesCalculator,
		profilesFilter:     c.profilesFilter,
	}
	if c.forkedData != nil {
		cloned.forkedData = c.forkedData.clone()
	}
	if c.data != nil {
		cloned.data = c.data.clone()
	}
	return cloned
}

func (c *clusterSnapshot) Commit() {
	if c.forkedData != nil {
		c.data = c.forkedData
		c.forkedData = nil
	}
}

func (c *clusterSnapshot) Revert() {
	c.forkedData = nil
}

func (c *clusterSnapshot) GetCandidateNodes() []PartitionableNode {
	result := make([]PartitionableNode, 0)
	for _, n := range c.getData().nodes {
		if n.HasFreeCapacity() {
			result = append(result, n)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].GetName() < result[j].GetName()
	})
	return result
}

func (c *clusterSnapshot) getLackingResources(pod v1.Pod) framework.Resource {
	podRequest := resource.ComputePodRequest(pod)
	totalAllocatable := framework.Resource{}
	totalRequested := framework.Resource{}
	for _, n := range c.GetNodes() {
		totalAllocatable = resource.Sum(totalAllocatable, *n.NodeInfo().Allocatable)
		totalRequested = resource.Sum(totalRequested, *n.NodeInfo().Requested)
	}
	available := resource.SubtractNonNegative(totalAllocatable, totalRequested)
	diff := resource.Subtract(available, resource.FromListToFramework(podRequest))

	// consider only negative (e.g. lacking) quantities
	res := framework.NewResource(v1.ResourceList{})
	res.ScalarResources = make(map[v1.ResourceName]int64)
	if diff.MilliCPU < 0 {
		res.MilliCPU = diff.MilliCPU
	}
	if diff.Memory < 0 {
		res.Memory = diff.Memory
	}
	if diff.EphemeralStorage < 0 {
		res.EphemeralStorage = diff.EphemeralStorage
	}
	if diff.AllowedPodNumber < 0 {
		res.AllowedPodNumber = diff.AllowedPodNumber
	}
	for k, v := range diff.ScalarResources {
		if v < 0 {
			res.ScalarResources[k] = v
		}
	}

	return resource.Abs(*res)
}

func (c *clusterSnapshot) GetNodes() map[string]PartitionableNode {
	return c.getData().nodes
}

func (c *clusterSnapshot) GetNode(name string) (PartitionableNode, bool) {
	node, found := c.GetNodes()[name]
	return node, found
}

func (c *clusterSnapshot) SetNode(node PartitionableNode) {
	c.getData().nodes[node.GetName()] = node
}

func (c *clusterSnapshot) AddPod(nodeName string, pod v1.Pod) error {
	node, found := c.getData().nodes[nodeName]
	if !found {
		return fmt.Errorf("could not find node %s in cluster snapshot", nodeName)
	}
	if err := node.AddPod(pod); err != nil {
		return err
	}
	c.getData().nodes[nodeName] = node
	return nil
}
