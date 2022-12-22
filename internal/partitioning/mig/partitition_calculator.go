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
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/core"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
)

var _ core.PartitionCalculator = partitionCalculator{}

type partitionCalculator struct {
}

func (p partitionCalculator) GetPartitioning(node core.PartitionableNode) state.NodePartitioning {
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
			Resources: mig.AsResources(g.GetGeometry()),
		}
		gpuPartitioning = append(gpuPartitioning, gp)
	}
	return state.NodePartitioning{GPUs: gpuPartitioning}
}

func NewPartitionCalculator() core.PartitionCalculator {
	return partitionCalculator{}
}
