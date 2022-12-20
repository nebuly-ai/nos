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
	mig_partitioner "github.com/nebuly-ai/nebulnetes/internal/partitioning/mig"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"testing"
)

func TestSnapshot__GetLackingSlices(t *testing.T) {
	testCases := []struct {
		name          string
		snapshotNodes map[string]framework.NodeInfo
		pod           v1.Pod
		expected      map[gpu.Slice]int
	}{
		{
			name:          "Empty snapshot",
			snapshotNodes: make(map[string]framework.NodeInfo),
			pod: factory.BuildPod("ns-1", "pd-1").
				WithContainer(
					factory.BuildContainer("c1", "test").
						WithResourceRequest(v1.ResourceCPU, *resource.NewMilliQuantity(200, resource.DecimalSI)).
						WithResourceRequest(mig.Profile1g10gb.AsResourceName(), *resource.NewQuantity(2, resource.DecimalSI)).
						Get(),
				).
				Get(),
			expected: map[gpu.Slice]int{
				mig.Profile1g10gb: 2,
			},
		},
		{
			name: "NOT-empty snapshot",
			snapshotNodes: map[string]framework.NodeInfo{
				"node-1": {
					Requested: &framework.Resource{
						MilliCPU:         200,
						Memory:           200,
						EphemeralStorage: 0,
						AllowedPodNumber: 0,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU: 3,
						},
					},
					Allocatable: &framework.Resource{
						MilliCPU:         2000,
						Memory:           200,
						EphemeralStorage: 0,
						AllowedPodNumber: 0,
						ScalarResources: map[v1.ResourceName]int64{
							constant.ResourceNvidiaGPU:        3,
							mig.Profile1g5gb.AsResourceName(): 1, // not requested by Pod, should be excluded from result
						},
					},
				},
				"node-2": {
					Requested: &framework.Resource{
						MilliCPU:         100,
						Memory:           0,
						EphemeralStorage: 0,
						AllowedPodNumber: 0,
						ScalarResources:  nil,
					},
					Allocatable: &framework.Resource{
						MilliCPU:         2000,
						Memory:           200,
						EphemeralStorage: 0,
						AllowedPodNumber: 0,
						ScalarResources:  nil,
					},
				},
			},
			pod: factory.BuildPod("ns-1", "pd-1").
				WithContainer(
					factory.BuildContainer("c1", "test").
						WithResourceRequest(v1.ResourceCPU, *resource.NewMilliQuantity(4000, resource.DecimalSI)).
						WithResourceRequest(v1.ResourceMemory, *resource.NewQuantity(200, resource.DecimalSI)).
						WithResourceRequest(v1.ResourceEphemeralStorage, *resource.NewQuantity(1, resource.DecimalSI)).
						WithResourceRequest(v1.ResourcePods, *resource.NewQuantity(1, resource.DecimalSI)).
						WithResourceRequest(constant.ResourceNvidiaGPU, *resource.NewQuantity(2, resource.DecimalSI)).
						WithResourceRequest(mig.Profile1g10gb.AsResourceName(), *resource.NewQuantity(2, resource.DecimalSI)).
						Get(),
				).
				WithContainer(
					factory.BuildContainer("c1", "test").
						WithResourceRequest(mig.Profile7g40gb.AsResourceName(), *resource.NewQuantity(1, resource.DecimalSI)).
						Get(),
				).
				Get(),
			expected: map[gpu.Slice]int{
				mig.Profile1g10gb: 2,
				mig.Profile7g40gb: 1,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			s := state.NewClusterState(tt.snapshotNodes)
			snapshot, err := mig_partitioner.NewSnapshotTaker().TakeSnapshot(&s)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, snapshot.GetLackingSlices(tt.pod))
		})
	}
}
