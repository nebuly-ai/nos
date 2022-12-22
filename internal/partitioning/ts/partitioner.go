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

package ts

import (
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/core"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/timeslicing"
)

var _ core.PartitionCalculator = partitioner{}

type partitioner struct {
}

func (p partitioner) GetPartitioning(node core.PartitionableNode) state.NodePartitioning {
	tsNode, ok := node.(*timeslicing.Node)
	if !ok {
		return state.NodePartitioning{
			GPUs: make([]state.GPUPartitioning, 0),
		}
	}
	gpuPartitioning := make([]state.GPUPartitioning, 0)
	for _, g := range tsNode.GPUs {
		gp := state.GPUPartitioning{
			GPUIndex:  g.Index,
			Resources: timeslicing.AsResources(g.GetGeometry()),
		}
		gpuPartitioning = append(gpuPartitioning, gp)
	}
	return state.NodePartitioning{GPUs: gpuPartitioning}
}

func NewPartitioner() core.PartitionCalculator {
	return partitioner{}
}
