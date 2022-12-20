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
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/core"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

var _ core.SliceFilter = sliceFilterAdapter(nil)
var _ core.Partitioner = partitionerAdapter(nil)
var _ core.SliceCalculator = sliceCalculatorAdapter(nil)
var _ core.SnapshotTaker = snapshotTakerAdapter(nil)

type sliceFilterAdapter func(resources map[v1.ResourceName]int64) map[gpu.Slice]int

func (f sliceFilterAdapter) ExtractSlices(resources map[v1.ResourceName]int64) map[gpu.Slice]int {
	return f(resources)
}

func NewSliceFilter() core.SliceFilter {
	var sliceFilter sliceFilterAdapter = func(resources map[v1.ResourceName]int64) map[gpu.Slice]int {
		var res = make(map[gpu.Slice]int)
		for r, q := range resources {
			if mig.IsNvidiaMigDevice(r) {
				profileName, _ := mig.ExtractProfileName(r)
				res[profileName] += int(q)
			}
		}
		return res
	}
	return sliceFilter
}

type partitionerAdapter func(node core.PartitionableNode) state.NodePartitioning

func (f partitionerAdapter) GetPartitioning(node core.PartitionableNode) state.NodePartitioning {
	return f(node)
}

func NewPartitioner() core.Partitioner {
	var partitioner partitionerAdapter = func(node core.PartitionableNode) state.NodePartitioning {
		migNode, ok := node.(*mig.Node)
		if !ok {
			return state.NodePartitioning{
				GPUs: make([]state.GPUPartitioning, 0),
			}
		}
		gpuPartitioning := make([]state.GPUPartitioning, 0)
		for _, g := range migNode.GPUs {
			gp := state.GPUPartitioning{
				GPUIndex:  g.GetIndex(),
				Resources: g.GetGeometry().AsResources(),
			}
			gpuPartitioning = append(gpuPartitioning, gp)
		}
		return state.NodePartitioning{GPUs: gpuPartitioning}
	}
	return partitioner
}

type sliceCalculatorAdapter func(pod v1.Pod) map[gpu.Slice]int

func (f sliceCalculatorAdapter) GetRequestedSlices(pod v1.Pod) map[gpu.Slice]int {
	return f(pod)
}

func NewSliceCalculator() core.SliceCalculator {
	var sliceCalculator sliceCalculatorAdapter = func(pod v1.Pod) map[gpu.Slice]int {
		requestedMigProfiles := mig.GetRequestedProfiles(pod)
		res := make(map[gpu.Slice]int, len(requestedMigProfiles))
		for p, q := range requestedMigProfiles {
			res[p] = q
		}
		return res
	}
	return sliceCalculator
}

func NewPlanner(scheduler framework.Framework) core.Planner {
	sliceCalculator := NewSliceCalculator()
	partitioner := NewPartitioner()
	return core.NewPlanner(
		partitioner,
		sliceCalculator,
		scheduler,
	)
}

type snapshotTakerAdapter func(clusterState *state.ClusterState) (core.Snapshot, error)

func (s snapshotTakerAdapter) TakeSnapshot(clusterState *state.ClusterState) (core.Snapshot, error) {
	return s(clusterState)
}

func NewSnapshotTaker() core.SnapshotTaker {
	var snapshotTaker snapshotTakerAdapter = func(clusterState *state.ClusterState) (core.Snapshot, error) {
		// Extract nodes with MIG partitioning enabled
		nodes := make(map[string]core.PartitionableNode)
		for k, v := range clusterState.GetNodes() {
			if v.Node() == nil {
				continue
			}
			if !gpu.IsMigPartitioningEnabled(*v.Node()) {
				continue
			}
			migNode, err := mig.NewNode(v)
			if err != nil {
				return nil, err
			}
			nodes[k] = &migNode
		}
		partitioner := NewPartitioner()
		sliceCalculator := NewSliceCalculator()
		sliceFilter := NewSliceFilter()
		snapshot := core.NewClusterSnapshot(
			nodes,
			partitioner,
			sliceCalculator,
			sliceFilter,
		)
		return snapshot, nil
	}
	return snapshotTaker
}
