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

package gpu

import (
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

type PartitioningKind string

func (p PartitioningKind) String() string {
	return string(p)
}

const (
	PartitioningKindMig         PartitioningKind = "mig"
	PartitioningKindTimeSlicing PartitioningKind = "time-slicing"
	PartitioningKindHybrid      PartitioningKind = "hybrid"
)

// IsMigPartitioningEnabled returns true if the node is enabled for automatic MIG GPU partitioning, false otherwise
func IsMigPartitioningEnabled(node v1.Node) bool {
	partitioningKind, ok := node.Labels[v1alpha1.LabelGpuPartitioning]
	if !ok {
		return false
	}
	return partitioningKind == PartitioningKindMig.String()
}

// IsTimeSlicingPartitioningEnabled returns true if the node is enabled for
// automatic time-slicing GPU partitioning, false otherwise
func IsTimeSlicingPartitioningEnabled(node v1.Node) bool {
	partitioningKind, ok := node.Labels[v1alpha1.LabelGpuPartitioning]
	if !ok {
		return false
	}
	return partitioningKind == PartitioningKindTimeSlicing.String()
}
