package plan

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewMigConfigPlan(t *testing.T) {
	testCases := []struct {
		name              string
		state             MigState
		specAnnotations   map[string]string
		expectedCreateOps []CreateOperation
		expectedDeleteOps []DeleteOperation
	}{
		{
			name:  "Empty state",
			state: map[int]mig.DeviceResourceList{},
			specAnnotations: map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "1g.20gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "4g.20gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 1, "1g.10gb"): "2",
			},
			expectedDeleteOps: []DeleteOperation{},
			expectedCreateOps: []CreateOperation{
				{
					MigProfile: mig.Profile{
						GpuIndex: 0,
						Name:     "1g.20gb",
					},
					Quantity: 1,
				},
				{
					MigProfile: mig.Profile{
						GpuIndex: 0,
						Name:     "4g.20gb",
					},
					Quantity: 1,
				},
				{
					MigProfile: mig.Profile{
						GpuIndex: 1,
						Name:     "1g.10gb",
					},
					Quantity: 2,
				},
			},
		},
		{
			name: "Empty spec annotations",
			state: map[int]mig.DeviceResourceList{
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
			expectedDeleteOps: []DeleteOperation{
				{
					Resources: []mig.DeviceResource{
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
					Quantity: 2,
				},
				{
					Resources: []mig.DeviceResource{
						{
							Device: resource.Device{
								ResourceName: "nvidia.com/mig-2g.20gb",
								DeviceId:     "3",
								Status:       resource.StatusFree,
							},
							GpuIndex: 1,
						},
					},
					Quantity: 1,
				},
			},
			expectedCreateOps: []CreateOperation{},
		},
		{
			name:              "Empty state, empty spec annotations",
			state:             MigState{},
			specAnnotations:   map[string]string{},
			expectedCreateOps: []CreateOperation{},
			expectedDeleteOps: []DeleteOperation{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotations := make(mig.GPUSpecAnnotationList, 0)
			for k, v := range tt.specAnnotations {
				a, err := mig.NewGPUSpecAnnotation(k, v)
				assert.NoError(t, err)
				annotations = append(annotations, a)
			}
			plan := NewMigConfigPlan(tt.state, annotations)
			assert.ElementsMatch(t, tt.expectedDeleteOps, plan.DeleteOperations)
			assert.ElementsMatch(t, tt.expectedCreateOps, plan.CreateOperations)
		})
	}
}
