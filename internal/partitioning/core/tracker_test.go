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

package core_test

import (
	"github.com/nebuly-ai/nos/internal/partitioning/core"
	mig_partitioner "github.com/nebuly-ai/nos/internal/partitioning/mig"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/mig"
	"github.com/nebuly-ai/nos/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestSliceTracker__Remove(t *testing.T) {
	testCases := []struct {
		name                    string
		nodes                   []v1.Node
		pods                    []v1.Pod
		podToRemove             v1.Pod
		expectedRequestedSlices map[gpu.Slice]int
		expectedLackingSlices   map[gpu.Slice]int
	}{
		{
			name:                    "Empty snapshot, empty tracker",
			nodes:                   []v1.Node{},
			pods:                    []v1.Pod{},
			podToRemove:             v1.Pod{},
			expectedRequestedSlices: map[gpu.Slice]int{},
			expectedLackingSlices:   map[gpu.Slice]int{},
		},
		{
			name:                    "Pod not tracked",
			nodes:                   []v1.Node{},
			pods:                    []v1.Pod{},
			podToRemove:             factory.BuildPod("ns-1", "pd-1").Get(),
			expectedRequestedSlices: map[gpu.Slice]int{},
			expectedLackingSlices:   map[gpu.Slice]int{},
		},
		{
			name:  "Quantities <= 0 should be removed",
			nodes: []v1.Node{},
			pods: []v1.Pod{
				factory.BuildPod("ns-1", "pd-1").WithContainer(
					factory.BuildContainer("c1", "test").
						WithScalarResourceRequest(mig.Profile1g10gb.AsResourceName(), 1).
						WithScalarResourceRequest(mig.Profile7g40gb.AsResourceName(), 2).
						Get(),
				).Get(),
				factory.BuildPod("ns-1", "pd-2").WithContainer(
					factory.BuildContainer("c1", "test").
						WithScalarResourceRequest(mig.Profile1g10gb.AsResourceName(), 1).
						Get(),
				).Get(),
			},
			podToRemove: factory.BuildPod("ns-1", "pd-1").WithContainer(
				factory.BuildContainer("c1", "test").
					WithScalarResourceRequest(mig.Profile1g10gb.AsResourceName(), 1).
					WithScalarResourceRequest(mig.Profile7g40gb.AsResourceName(), 2).
					Get(),
			).Get(),
			expectedRequestedSlices: map[gpu.Slice]int{
				mig.Profile1g10gb: 1,
			},
			expectedLackingSlices: map[gpu.Slice]int{
				mig.Profile1g10gb: 1,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := newSnapshotFromNodes(tt.nodes, mig_partitioner.NewSnapshotTaker())
			tracker := core.NewSliceTracker(
				snapshot,
				mig_partitioner.NewSliceCalculator(),
				tt.pods,
			)
			tracker.Remove(tt.podToRemove)
			assert.Equal(t, tt.expectedRequestedSlices, tracker.GetRequestedSlices())
			assert.Equal(t, tt.expectedLackingSlices, tracker.GetLackingSlices())
		})
	}
}
