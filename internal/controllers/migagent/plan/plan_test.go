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
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "1",
							Status:       resource.StatusUsed,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
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
								ResourceName: mig.Profile1g10gb.AsResourceName(),
								DeviceId:     "1",
								Status:       resource.StatusUsed,
							},
							GpuIndex: 0,
						},
						{
							Device: resource.Device{
								ResourceName: mig.Profile1g10gb.AsResourceName(),
								DeviceId:     "2",
								Status:       resource.StatusUsed,
							},
							GpuIndex: 0,
						},
					},
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
		{
			name: "Delete operations should use free devices when available",
			state: MigState{
				0: {
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "2",
							Status:       resource.StatusUsed,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "3",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
				},
			},
			specAnnotations: map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "1g.10gb"): "1",
			},
			expectedCreateOps: []CreateOperation{},
			expectedDeleteOps: []DeleteOperation{
				{
					Resources: mig.DeviceResourceList{
						{
							Device: resource.Device{
								ResourceName: mig.Profile1g10gb.AsResourceName(),
								DeviceId:     "1",
								Status:       resource.StatusFree,
							},
							GpuIndex: 0,
						},
						{
							Device: resource.Device{
								ResourceName: mig.Profile1g10gb.AsResourceName(),
								DeviceId:     "3",
								Status:       resource.StatusFree,
							},
							GpuIndex: 0,
						},
					},
				},
			},
		},
		{
			name: "Creating new profiles on a GPU should delete all the existing **free** MIG profiles of the same type on that GPU",
			state: MigState{
				0: {
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "2",
							Status:       resource.StatusUsed,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "3",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
					{
						Device: resource.Device{
							ResourceName: mig.Profile1g10gb.AsResourceName(),
							DeviceId:     "1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 1,
					},
				},
			},
			specAnnotations: map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "1g.10gb"): "4",
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 1, "1g.10gb"): "1",
			},
			expectedCreateOps: []CreateOperation{
				{
					MigProfile: mig.Profile{
						GpuIndex: 0,
						Name:     mig.Profile1g10gb,
					},
					Quantity: 1,
				},
			},
			expectedDeleteOps: []DeleteOperation{
				{
					Resources: mig.DeviceResourceList{
						{
							Device: resource.Device{
								ResourceName: mig.Profile1g10gb.AsResourceName(),
								DeviceId:     "1",
								Status:       resource.StatusFree,
							},
							GpuIndex: 0,
						},
						{
							Device: resource.Device{
								ResourceName: mig.Profile1g10gb.AsResourceName(),
								DeviceId:     "3",
								Status:       resource.StatusFree,
							},
							GpuIndex: 0,
						},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			annotations := make(mig.GPUSpecAnnotationList, 0)
			for k, v := range tt.specAnnotations {
				a, err := mig.NewGPUSpecAnnotationFromNodeAnnotation(k, v)
				assert.NoError(t, err)
				annotations = append(annotations, a)
			}
			plan := NewMigConfigPlan(tt.state, annotations)
			assert.ElementsMatch(t, tt.expectedDeleteOps, plan.DeleteOperations)
			assert.ElementsMatch(t, tt.expectedCreateOps, plan.CreateOperations)
		})
	}
}

func TestMigConfigPlan_IsEmpty(t *testing.T) {
	testCases := []struct {
		name     string
		plan     MigConfigPlan
		expected bool
	}{
		{
			name: "Empty plan",
			plan: MigConfigPlan{
				DeleteOperations: make(DeleteOperationList, 0),
				CreateOperations: make(CreateOperationList, 0),
			},
			expected: true,
		},
		{
			name: "Plan with one CreateOperation is not empty",
			plan: MigConfigPlan{
				DeleteOperations: make(DeleteOperationList, 0),
				CreateOperations: CreateOperationList{
					CreateOperation{
						MigProfile: mig.Profile{},
						Quantity:   0,
					},
				},
			},
			expected: false,
		},
		{
			name: "Plan with one DeleteOperation is not empty",
			plan: MigConfigPlan{
				DeleteOperations: DeleteOperationList{{Resources: make(mig.DeviceResourceList, 0)}},
				CreateOperations: make(CreateOperationList, 0),
			},
			expected: false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.plan.IsEmpty()
			assert.Equal(t, tt.expected, res)
		})
	}
}

func TestMigConfigPlan__Equals(t *testing.T) {
	testCases := []struct {
		name     string
		plan     MigConfigPlan
		other    *MigConfigPlan
		expected bool
	}{
		{
			name: "other is nil",
			plan: MigConfigPlan{
				DeleteOperations: DeleteOperationList{{Resources: make(mig.DeviceResourceList, 0)}},
				CreateOperations: make(CreateOperationList, 0),
			},
			other:    nil,
			expected: false,
		},
		{
			name: "other is equal",
			plan: MigConfigPlan{
				DeleteOperations: DeleteOperationList{
					{
						Resources: mig.DeviceResourceList{
							{
								Device: resource.Device{
									ResourceName: mig.Profile1g10gb.AsResourceName(),
									DeviceId:     "1",
									Status:       resource.StatusFree,
								},
								GpuIndex: 0,
							},
							{
								Device: resource.Device{
									ResourceName: mig.Profile1g10gb.AsResourceName(),
									DeviceId:     "3",
									Status:       resource.StatusFree,
								},
								GpuIndex: 0,
							},
						},
					},
				},
				CreateOperations: CreateOperationList{
					{
						MigProfile: mig.Profile{
							GpuIndex: 1,
							Name:     mig.Profile1g10gb,
						},
						Quantity: 1,
					},
					{
						MigProfile: mig.Profile{
							GpuIndex: 1,
							Name:     mig.Profile2g12gb,
						},
						Quantity: 1,
					},
				},
			},
			other: &MigConfigPlan{
				DeleteOperations: DeleteOperationList{
					{
						Resources: mig.DeviceResourceList{
							{
								Device: resource.Device{
									ResourceName: mig.Profile1g10gb.AsResourceName(),
									DeviceId:     "3",
									Status:       resource.StatusFree,
								},
								GpuIndex: 0,
							},
							{
								Device: resource.Device{
									ResourceName: mig.Profile1g10gb.AsResourceName(),
									DeviceId:     "1",
									Status:       resource.StatusFree,
								},
								GpuIndex: 0,
							},
						},
					},
				},
				CreateOperations: CreateOperationList{
					{
						MigProfile: mig.Profile{
							GpuIndex: 1,
							Name:     mig.Profile2g12gb,
						},
						Quantity: 1,
					},
					{
						MigProfile: mig.Profile{
							GpuIndex: 1,
							Name:     mig.Profile1g10gb,
						},
						Quantity: 1,
					},
				},
			},
			expected: true,
		},
		{
			name: "other is *not* equal",
			plan: MigConfigPlan{
				DeleteOperations: DeleteOperationList{
					{
						Resources: mig.DeviceResourceList{
							{
								Device: resource.Device{
									ResourceName: mig.Profile1g10gb.AsResourceName(),
									DeviceId:     "1",
									Status:       resource.StatusFree,
								},
								GpuIndex: 0,
							},
							{
								Device: resource.Device{
									ResourceName: mig.Profile1g10gb.AsResourceName(),
									DeviceId:     "3",
									Status:       resource.StatusFree,
								},
								GpuIndex: 0,
							},
						},
					},
				},
				CreateOperations: CreateOperationList{
					{
						MigProfile: mig.Profile{
							GpuIndex: 1,
							Name:     mig.Profile1g10gb,
						},
						Quantity: 0,
					},
				},
			},
			other: &MigConfigPlan{
				DeleteOperations: DeleteOperationList{
					{
						Resources: mig.DeviceResourceList{
							{
								Device: resource.Device{
									ResourceName: mig.Profile1g10gb.AsResourceName(),
									DeviceId:     "1",
									Status:       resource.StatusFree,
								},
								GpuIndex: 0,
							},
							{
								Device: resource.Device{
									ResourceName: mig.Profile1g10gb.AsResourceName(),
									DeviceId:     "1",
									Status:       resource.StatusFree,
								},
								GpuIndex: 0,
							},
						},
					},
				},
				CreateOperations: CreateOperationList{
					{
						MigProfile: mig.Profile{
							GpuIndex: 1,
							Name:     mig.Profile1g10gb,
						},
						Quantity: 0,
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.plan.Equal(tt.other)
			swappedRes := tt.other.Equal(&tt.plan)
			assert.Equal(t, res, swappedRes, "Equal function is not symmetric")
			assert.Equal(t, tt.expected, res)
		})
	}
}
