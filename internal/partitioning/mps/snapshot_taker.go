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

package mps

import (
	"github.com/nebuly-ai/nos/internal/partitioning/core"
	"github.com/nebuly-ai/nos/internal/partitioning/state"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/slicing"
)

var _ core.SnapshotTaker = snapshotTaker{}

type snapshotTaker struct {
}

func (s snapshotTaker) TakeSnapshot(clusterState *state.ClusterState) (core.Snapshot, error) {
	nodes := make(map[string]core.PartitionableNode)
	for k, v := range clusterState.GetNodes() {
		if v.Node() == nil {
			continue
		}
		if !gpu.IsMpsPartitioningEnabled(*v.Node()) {
			continue
		}
		slicingNode, err := slicing.NewNode(v)
		if err != nil {
			return nil, err
		}
		nodes[k] = &slicingNode
	}
	snapshot := core.NewClusterSnapshot(
		nodes,
		NewPartitionCalculator(),
		NewSliceCalculator(),
		NewSliceFilter(),
	)
	return snapshot, nil
}

func NewSnapshotTaker() core.SnapshotTaker {
	return snapshotTaker{}
}
