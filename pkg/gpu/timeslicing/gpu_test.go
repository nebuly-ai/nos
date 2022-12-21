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

package timeslicing_test

import (
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/timeslicing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewGPU(t *testing.T) {
	testCases := []struct {
		name         string
		model        gpu.Model
		index        int
		memoryGB     int
		usedProfiles map[timeslicing.ProfileName]int
		freeProfiles map[timeslicing.ProfileName]int
		expected     timeslicing.GPU
		expectedErr  bool
	}{
		{
			name:     "Sum of profiles memory exceeds GPU memory",
			model:    gpu.GPUModel_A100_PCIe_80GB,
			index:    0,
			memoryGB: 40,
			usedProfiles: map[timeslicing.ProfileName]int{
				"nvidia.com/gpu-10gb": 5,
			},
			freeProfiles: map[timeslicing.ProfileName]int{
				"nvidia.com/gpu-20gb": 1,
			},
			expected:    timeslicing.GPU{},
			expectedErr: true,
		},
		{
			name:     "Sum of profiles memory equal to GPU memory",
			model:    gpu.GPUModel_A100_PCIe_80GB,
			index:    0,
			memoryGB: 30,
			usedProfiles: map[timeslicing.ProfileName]int{
				"nvidia.com/gpu-10gb": 2,
			},
			freeProfiles: map[timeslicing.ProfileName]int{
				"nvidia.com/gpu-10gb": 1,
			},
			expected: timeslicing.NewGpuOrPanic(
				gpu.GPUModel_A100_PCIe_80GB,
				0,
				30,
				map[timeslicing.ProfileName]int{
					"nvidia.com/gpu-10gb": 2,
				},
				map[timeslicing.ProfileName]int{
					"nvidia.com/gpu-10gb": 1,
				},
			),
			expectedErr: false,
		},
		{
			name:     "Used profile with memory size smaller than min",
			model:    gpu.GPUModel_A100_PCIe_80GB,
			index:    0,
			memoryGB: 30,
			usedProfiles: map[timeslicing.ProfileName]int{
				"nvidia.com/gpu-0gb": 2,
			},
			freeProfiles: map[timeslicing.ProfileName]int{
				"nvidia.com/gpu-10gb": 2,
			},
			expected:    timeslicing.GPU{},
			expectedErr: true,
		},
		{
			name:     "Free profile with memory size smaller than min",
			model:    gpu.GPUModel_A100_PCIe_80GB,
			index:    0,
			memoryGB: 30,
			usedProfiles: map[timeslicing.ProfileName]int{
				"nvidia.com/gpu-10gb": 2,
			},
			freeProfiles: map[timeslicing.ProfileName]int{
				"nvidia.com/gpu-0gb": 2,
			},
			expected:    timeslicing.GPU{},
			expectedErr: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			g, err := timeslicing.NewGPU(
				tt.model,
				tt.index,
				tt.memoryGB,
				tt.usedProfiles,
				tt.freeProfiles,
			)
			if tt.expectedErr {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.expected, g)
		})
	}
}
