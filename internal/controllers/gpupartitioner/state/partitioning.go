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

package state

import (
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
)

type GPUPartitioning struct {
	GPUIndex  int
	Resources map[v1.ResourceName]int
}

type NodePartitioning struct {
	GPUs []GPUPartitioning
}

func (n NodePartitioning) Equal(other NodePartitioning) bool {
	if len(n.GPUs) != len(other.GPUs) {
		return false
	}
	return util.UnorderedEqual(n.GPUs, other.GPUs)
}

type PartitioningState map[string]NodePartitioning

func (p PartitioningState) IsEmpty() bool {
	return len(p) == 0
}

func (p PartitioningState) Equal(other PartitioningState) bool {
	if len(p) != len(other) {
		return false
	}
	for node, nodePartitioning := range p {
		if !nodePartitioning.Equal(other[node]) {
			return false
		}
	}
	return true
}
