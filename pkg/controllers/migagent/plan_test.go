package migagent

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/migagent/types"
	migtypes "github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestComputePlan(t *testing.T) {
	testCases := []struct {
		name            string
		state           types.MigState
		specAnnotations map[string]string
		expected        migConfigPlan
	}{
		{
			name:  "Empty state",
			state: map[int]migtypes.MigDeviceResourceList{},
			specAnnotations: map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "1g.20gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "4g.20gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 1, "1g.10gb"): "2",
			},
			expected: migConfigPlan{
				{
					migProfile:      "1g.20gb",
					gpuIndex:        0,
					desiredQuantity: 1,
					actualResources: []migtypes.MigDeviceResource{},
				},
				{
					migProfile:      "4g.20gb",
					gpuIndex:        0,
					desiredQuantity: 1,
					actualResources: []migtypes.MigDeviceResource{},
				},
				{
					migProfile:      "1g.10gb",
					gpuIndex:        1,
					desiredQuantity: 2,
					actualResources: []migtypes.MigDeviceResource{},
				},
			},
		},
		{
			name: "Empty spec annotations",
			state: map[int]migtypes.MigDeviceResourceList{
				0: {
					{
						Device: resource.Device{
							ResourceName: "nvidia.com/mig-1g.10gb",
							DeviceId:     "1",
							Status:       resource.StatusUsed,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: "nvidia.com/mig-1g.10gb",
							DeviceId:     "2",
							Status:       resource.StatusUsed,
						},
						GpuIndex: 0,
					},
				},
				1: {
					{
						Device: resource.Device{
							ResourceName: "nvidia.com/mig-2g.20gb",
							DeviceId:     "3",
							Status:       resource.StatusFree,
						},
						GpuIndex: 1,
					},
				},
			},
			specAnnotations: map[string]string{},
			expected: migConfigPlan{
				{
					migProfile:      "1g.10gb",
					gpuIndex:        0,
					desiredQuantity: 0,
					actualResources: []migtypes.MigDeviceResource{
						{
							Device: resource.Device{
								ResourceName: "nvidia.com/mig-1g.10gb",
								DeviceId:     "1",
								Status:       resource.StatusUsed,
							},
							GpuIndex: 0,
						},
						{
							Device: resource.Device{
								ResourceName: "nvidia.com/mig-1g.10gb",
								DeviceId:     "2",
								Status:       resource.StatusUsed,
							},
							GpuIndex: 0,
						},
					},
				},
				{
					migProfile:      "2g.20gb",
					gpuIndex:        1,
					desiredQuantity: 0,
					actualResources: []migtypes.MigDeviceResource{
						{
							Device: resource.Device{
								ResourceName: "nvidia.com/mig-2g.20gb",
								DeviceId:     "3",
								Status:       resource.StatusFree,
							},
							GpuIndex: 1,
						},
					},
				},
			},
		},
		{
			name:            "Empty state, empty spec annotations",
			state:           types.MigState{},
			specAnnotations: map[string]string{},
			expected:        make(migConfigPlan, 0),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotations := make(types.GPUSpecAnnotationList, 0)
			for k, v := range tt.specAnnotations {
				a, err := types.NewGPUSpecAnnotation(k, v)
				assert.NoError(t, err)
				annotations = append(annotations, a)
			}
			plan := computePlan(tt.state, annotations)
			assert.ElementsMatch(t, tt.expected, plan)
		})
	}
}
