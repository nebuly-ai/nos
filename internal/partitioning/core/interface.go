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
	"context"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type Planner interface {
	Plan(ctx context.Context, snapshot Snapshot, pendingPods []v1.Pod) (PartitioningPlan, error)
}

type Actuator interface {
	Apply(ctx context.Context, snapshot Snapshot, plan PartitioningPlan) (bool, error)
}

type SliceCalculator interface {
	GetRequestedSlices(pod v1.Pod) map[gpu.Slice]int
}

type SliceFilter interface {
	ExtractSlices(resources map[v1.ResourceName]int64) map[gpu.Slice]int
}

type PartitionableNode interface {
	UpdateGeometryFor(slices map[gpu.Slice]int) (bool, error)
	GetName() string
	Geometry() map[gpu.Slice]int
	NodeInfo() framework.NodeInfo
	Clone() interface{}
	AddPod(pod v1.Pod) error
	HasFreeCapacity() bool
}

type Partitioner interface {
	GetPartitioning(node PartitionableNode) state.NodePartitioning
}

type Snapshot interface {
	GetPartitioningState() state.PartitioningState
	GetCandidateNodes() []PartitionableNode
	GetLackingSlices(pod v1.Pod) map[gpu.Slice]int
	SetNode(n PartitionableNode)
	Fork() error
	Commit()
	Revert()
	GetNode(name string) (PartitionableNode, bool)
	GetNodes() map[string]PartitionableNode
	AddPod(node string, pod v1.Pod) error
}

type SnapshotTaker interface {
	TakeSnapshot(clusterState *state.ClusterState) (Snapshot, error)
}

type Sorter interface {
	Sort(pods []v1.Pod) []v1.Pod
}
