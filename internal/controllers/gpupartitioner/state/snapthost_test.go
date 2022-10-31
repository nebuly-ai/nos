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
		name     string
		snapshot state.ClusterSnapshot
		pod      v1.Pod
		expected framework.Resource
	}{
		{
			name:     "Empty snapshot",
			snapshot: state.ClusterSnapshot{},
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
			snapshot: state.ClusterSnapshot{
				Nodes: map[string]framework.NodeInfo{
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
			assert.Equal(t, tt.expected, tt.snapshot.GetLackingResources(tt.pod))
		})
	}
}
