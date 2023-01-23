/*
 * Copyright 2023 nebuly.com.
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
	"github.com/nebuly-ai/nos/internal/partitioning/state"
	"github.com/nebuly-ai/nos/pkg/api/nos.nebuly.com/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/mig"
	"github.com/nebuly-ai/nos/pkg/test/factory"
	"github.com/nebuly-ai/nos/pkg/test/mocks"
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
			snapshot, err := mig_partitioner.NewSnapshotTaker().TakeSnapshot(s)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, snapshot.GetLackingSlices(tt.pod))
		})
	}
}

func newMigSnapshot(t *testing.T, nodes []v1.Node) core.Snapshot {
	migNodes := make(map[string]core.PartitionableNode, len(nodes))
	for _, n := range nodes {
		nodeInfo := framework.NewNodeInfo()
		nodeInfo.SetNode(&n)
		migNode, err := mig.NewNode(*nodeInfo)
		if err != nil {
			panic(err)
		}
		migNodes[n.Name] = &migNode
	}
	return core.NewClusterSnapshot(
		migNodes,
		mocks.NewPartitionCalculator(t),
		mocks.NewSliceCalculator(t),
		mocks.NewSliceFilter(t),
	)
}

func TestSnapshot__Forking(t *testing.T) {
	t.Run("Forking multiple times shall return error", func(t *testing.T) {
		snapshot := newMigSnapshot(t, []v1.Node{})
		assert.NoError(t, snapshot.Fork())
		assert.Error(t, snapshot.Fork())
	})

	t.Run("Test Revert changes", func(t *testing.T) {
		node := factory.BuildNode("node-1").WithLabels(map[string]string{
			v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
			constant.LabelNvidiaProduct:   string(gpu.GPUModel_A100_PCIe_80GB),
			constant.LabelNvidiaCount:     "1",
		}).Get()
		snapshot := newMigSnapshot(t, []v1.Node{node})
		originalNodes := make(map[string]core.PartitionableNode)
		for k, v := range snapshot.GetNodes() {
			originalNodes[k] = v.Clone().(core.PartitionableNode)
		}
		pod := factory.BuildPod("ns-1", "pod-1").WithContainer(
			factory.BuildContainer("c1", "i1").WithCPUMilliRequest(1000).Get(),
		).Get()
		assert.NoError(t, snapshot.Fork())
		assert.NoError(t, snapshot.AddPod("node-1", pod))
		// Snapshot modified, should differ from original one
		for _, n := range originalNodes {
			snapshotNode, ok := snapshot.GetNode(n.GetName())
			assert.True(t, ok)
			snapshotRequested := snapshotNode.NodeInfo().Requested
			originalRequested := n.NodeInfo().Requested
			assert.NotEqual(t, originalRequested, snapshotRequested)
		}
		// Revert changes
		snapshot.Revert()
		// Changes reverted, snapshot should be equal as the original one before the changes
		for _, n := range originalNodes {
			snapshotNode, ok := snapshot.GetNode(n.GetName())
			assert.True(t, ok)
			snapshotRequested := snapshotNode.NodeInfo().Requested
			originalRequested := n.NodeInfo().Requested
			assert.Equal(t, originalRequested, snapshotRequested)
		}
	})

	t.Run("Test Commit changes", func(t *testing.T) {
		node := factory.BuildNode("node-1").WithLabels(map[string]string{
			v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
			constant.LabelNvidiaProduct:   string(gpu.GPUModel_A100_PCIe_80GB),
			constant.LabelNvidiaCount:     "1",
		}).Get()
		snapshot := newMigSnapshot(t, []v1.Node{node})
		originalNodes := make(map[string]core.PartitionableNode)
		for k, v := range snapshot.GetNodes() {
			originalNodes[k] = v.Clone().(core.PartitionableNode)
		}
		pod := factory.BuildPod("ns-1", "pod-1").WithContainer(
			factory.BuildContainer("c1", "i1").WithCPUMilliRequest(1000).Get(),
		).Get()
		assert.NoError(t, snapshot.Fork())
		assert.NoError(t, snapshot.AddPod("node-1", pod))
		// Snapshot modified, should differ from original one
		for _, n := range originalNodes {
			snapshotNode, ok := snapshot.GetNode(n.GetName())
			assert.True(t, ok)
			snapshotRequested := snapshotNode.NodeInfo().Requested
			originalRequested := n.NodeInfo().Requested
			assert.NotEqual(t, originalRequested, snapshotRequested)
		}
		// Commit changes
		snapshot.Commit()
		for _, n := range originalNodes {
			snapshotNode, ok := snapshot.GetNode(n.GetName())
			assert.True(t, ok)
			snapshotRequested := snapshotNode.NodeInfo().Requested
			originalRequested := n.NodeInfo().Requested
			assert.NotEqual(t, originalRequested, snapshotRequested)
		}
		// After committing it should be possible to fork the snapshot again
		assert.NoError(t, snapshot.Fork())
	})
}
