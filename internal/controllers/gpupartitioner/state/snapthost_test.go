package state_test

import (
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"testing"
)

func TestSnapshot__GetLackingResources(t *testing.T) {
	testCases := []struct {
		name          string
		snapshotNodes map[string]framework.NodeInfo
		pod           v1.Pod
		expected      framework.Resource
	}{
		{
			name:          "Empty snapshot",
			snapshotNodes: make(map[string]framework.NodeInfo),
			pod: factory.BuildPod("ns-1", "pd-1").
				WithContainer(
					factory.BuildContainer("c1", "test").
						WithResourceRequest(v1.ResourceCPU, *resource.NewMilliQuantity(200, resource.DecimalSI)).
						WithResourceRequest(constant.ResourceNvidiaGPU, *resource.NewQuantity(2, resource.DecimalSI)).
						Get(),
				).
				Get(),
			expected: framework.Resource{
				MilliCPU: 200,
				ScalarResources: map[v1.ResourceName]int64{
					constant.ResourceNvidiaGPU: 2,
				},
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
							constant.ResourceNvidiaGPU: 3,
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
						WithResourceRequest(constant.ResourceNvidiaGPU, *resource.NewQuantity(2, resource.DecimalSI)).
						Get(),
				).
				Get(),
			expected: framework.Resource{
				MilliCPU: 300,
				ScalarResources: map[v1.ResourceName]int64{
					constant.ResourceNvidiaGPU: 2,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := state.NewClusterSnapshot(tt.snapshotNodes)
			assert.Equal(t, tt.expected, snapshot.GetLackingResources(tt.pod))
		})
	}
}

func TestSnapshot__Forking(t *testing.T) {
	t.Run("Forking multiple times shall return error", func(t *testing.T) {
		snapshot := state.NewClusterSnapshot(map[string]framework.NodeInfo{})
		assert.NoError(t, snapshot.Fork())
		assert.Error(t, snapshot.Fork())
	})

	t.Run("Test Revert changes", func(t *testing.T) {
		snapshot := state.NewClusterSnapshot(map[string]framework.NodeInfo{
			"node-1": *framework.NewNodeInfo(),
		})
		originalNodes := make(map[string]framework.NodeInfo)
		for k, v := range snapshot.GetNodes() {
			originalNodes[k] = *v.Clone()
		}
		assert.NoError(t, snapshot.Fork())
		assert.NoError(t, snapshot.AddPod("node-1", factory.BuildPod("ns-1", "pod-1").Get()))
		// Snapshot modified, should differ from original one
		assert.NotEqual(t, originalNodes, snapshot.GetNodes())
		// Revert changes
		snapshot.Revert()
		// Changes reverted, snapshot should be equal as the original one before the changes
		assert.Equal(t, originalNodes, snapshot.GetNodes())
	})

	t.Run("Test Commit changes", func(t *testing.T) {
		snapshot := state.NewClusterSnapshot(map[string]framework.NodeInfo{
			"node-1": *framework.NewNodeInfo(),
		})
		originalNodes := make(map[string]*framework.NodeInfo)
		for k, v := range snapshot.GetNodes() {
			originalNodes[k] = v.Clone()
		}
		assert.NoError(t, snapshot.Fork())
		assert.NoError(t, snapshot.AddPod("node-1", factory.BuildPod("ns-1", "pod-1").Get()))
		// Snapshot modified, should differ from original one
		assert.NotEqual(t, originalNodes, snapshot.GetNodes())
		// Commit changes
		snapshot.Commit()
		assert.NotEqual(t, originalNodes, snapshot.GetNodes())
		// After committing it should be possible to fork the snapshot again
		assert.NoError(t, snapshot.Fork())
	})
}
