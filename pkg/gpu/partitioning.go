/*
 * Copyright 2023 Nebuly.ai.
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
	"fmt"
	"github.com/nebuly-ai/nos/pkg/api/nos.nebuly.ai/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"sort"
	"strings"
)

type Slice interface {
	SmallerThan(other Slice) bool
	String() string
}

// Geometry corresponds to the partitioning Geometry of a GPU,
// namely the slices of the GPU with the respective quantity.
type Geometry map[Slice]int

func (g Geometry) Id() string {
	return g.String()
}

func (g Geometry) String() string {
	// Sort profiles
	var orderedProfiles = make([]Slice, 0, len(g))
	for profile := range g {
		orderedProfiles = append(orderedProfiles, profile)
	}
	sort.SliceStable(orderedProfiles, func(i, j int) bool {
		return orderedProfiles[i].String() < orderedProfiles[j].String()
	})
	// Build string
	var builder strings.Builder
	for _, profile := range orderedProfiles {
		builder.WriteString(fmt.Sprintf("%s:%d, ", profile, g[profile]))
	}
	return builder.String()
}

func (g Geometry) MarshalJSON() ([]byte, error) {
	var asStr = make(map[string]int, len(g))
	for k, v := range g {
		asStr[k.String()] = v
	}
	return json.Marshal(asStr)
}

type PartitioningKind string

func (p PartitioningKind) String() string {
	return string(p)
}

const (
	PartitioningKindMig    PartitioningKind = "mig"
	PartitioningKindMps    PartitioningKind = "mps"
	PartitioningKindHybrid PartitioningKind = "hybrid"
)

// IsMigPartitioningEnabled returns true if the node is enabled for automatic MIG GPU partitioning, false otherwise
func IsMigPartitioningEnabled(node v1.Node) bool {
	partitioningKind, ok := node.Labels[v1alpha1.LabelGpuPartitioning]
	if !ok {
		return false
	}
	return partitioningKind == PartitioningKindMig.String()
}

// IsMpsPartitioningEnabled returns true if the node is enabled for
// automatic MPS GPU partitioning, false otherwise
func IsMpsPartitioningEnabled(node v1.Node) bool {
	partitioningKind, ok := node.Labels[v1alpha1.LabelGpuPartitioning]
	if !ok {
		return false
	}
	return partitioningKind == PartitioningKindMps.String()
}

func GetPartitioningKind(node v1.Node) (PartitioningKind, bool) {
	partitioningKindStr, ok := node.Labels[v1alpha1.LabelGpuPartitioning]
	if !ok {
		return "", false
	}
	partitioningKind, valid := asPartitioningKind(partitioningKindStr)
	if !valid {
		return "", false
	}
	return partitioningKind, true
}

func asPartitioningKind(kind string) (PartitioningKind, bool) {
	switch kind {
	case PartitioningKindMig.String():
		return PartitioningKindMig, true
	case PartitioningKindMps.String():
		return PartitioningKindMps, true
	case PartitioningKindHybrid.String():
		return PartitioningKindHybrid, true
	default:
		return "", false
	}
}

type SliceCalculator interface {
	GetRequestedSlices(pod v1.Pod) map[Slice]int
}

type SliceFilter interface {
	ExtractSlices(resources map[v1.ResourceName]int64) map[Slice]int
}
