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
		expectedCreateOps CreateOperationList
		expectedDeleteOps DeleteOperationList
	}{
		{
			name:  "Empty state",
			state: map[int]mig.DeviceResourceList{},
			specAnnotations: map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "1g.20gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "4g.20gb"): "1",
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 1, "1g.10gb"): "2",
			},
			expectedDeleteOps: DeleteOperationList{},
			expectedCreateOps: CreateOperationList{
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
			expectedDeleteOps: DeleteOperationList{
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
			expectedCreateOps: CreateOperationList{},
		},
		{
			name:              "Empty state, empty spec annotations",
			state:             MigState{},
			specAnnotations:   map[string]string{},
			expectedCreateOps: CreateOperationList{},
			expectedDeleteOps: DeleteOperationList{},
		},
		{
			name: "Free devices should not be re-created if there aren't create op on the GPU",
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
			expectedCreateOps: CreateOperationList{},
			expectedDeleteOps: DeleteOperationList{
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
			name: "Creating new profiles on a GPU should delete and re-create all the existing **free** MIG profiles of the same type on that GPU",
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
				1: {
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
			expectedCreateOps: CreateOperationList{
				{
					MigProfile: mig.Profile{
						GpuIndex: 0,
						Name:     mig.Profile1g10gb,
					},
					Quantity: 1, // op that creates the new device to create
				},
				{
					MigProfile: mig.Profile{
						GpuIndex: 0,
						Name:     mig.Profile1g10gb,
					},
					Quantity: 2, // op for re-creating the existing free-device
				},
			},
			expectedDeleteOps: DeleteOperationList{
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
			name: "All free devices should be re-created if there's any create op on the GPU",
			state: MigState{
				// 0:
				// 	1g.10gb-free -> 2
				//  1g.10gb-used -> 1
				//
				// 1:
				//  1g.10gb-free -> 1
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
				1: {
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
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "1g.10gb"): "3", // unchanged
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "2g.20gb"): "1", // new device
				fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 1, "1g.10gb"): "1", // unchanged
			},
			expectedCreateOps: CreateOperationList{
				{
					MigProfile: mig.Profile{
						GpuIndex: 0,
						Name:     mig.Profile1g10gb,
					},
					Quantity: 2, // op that re-creates the existing devices
				},
				{
					MigProfile: mig.Profile{
						GpuIndex: 0,
						Name:     mig.Profile2g20gb,
					},
					Quantity: 1, // that creates the new device
				},
			},
			expectedDeleteOps: DeleteOperationList{
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
