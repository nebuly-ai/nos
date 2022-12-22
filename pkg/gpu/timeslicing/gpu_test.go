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
				"10gb": 5,
			},
			freeProfiles: map[timeslicing.ProfileName]int{
				"20gb": 1,
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
				"10gb": 2,
			},
			freeProfiles: map[timeslicing.ProfileName]int{
				"10gb": 1,
			},
			expected: timeslicing.NewGpuOrPanic(
				gpu.GPUModel_A100_PCIe_80GB,
				0,
				30,
				map[timeslicing.ProfileName]int{
					"10gb": 2,
				},
				map[timeslicing.ProfileName]int{
					"10gb": 1,
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
				"0gb": 2,
			},
			freeProfiles: map[timeslicing.ProfileName]int{
				"10gb": 2,
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
				"10gb": 2,
			},
			freeProfiles: map[timeslicing.ProfileName]int{
				"0gb": 2,
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

func TestGPU_UpdateGeometryFor(t *testing.T) {
	testCases := []struct {
		name             string
		gpu              timeslicing.GPU
		requiredSlices   map[gpu.Slice]int
		expectedGeometry gpu.Geometry
		expectedUpdate   bool
	}{
		{
			name: "No slices required, should not update geometry",
			gpu: timeslicing.NewGpuOrPanic(
				gpu.GPUModel_A100_PCIe_80GB,
				0,
				40,
				map[timeslicing.ProfileName]int{
					"10gb": 2,
				},
				map[timeslicing.ProfileName]int{
					"20gb": 1,
				},
			),
			requiredSlices: map[gpu.Slice]int{},
			expectedUpdate: false,
			expectedGeometry: map[gpu.Slice]int{
				timeslicing.ProfileName("10gb"): 2,
				timeslicing.ProfileName("20gb"): 1,
			},
		},
		{
			name: "GPU already provides required slices, should not update geometry",
			gpu: timeslicing.NewGpuOrPanic(
				gpu.GPUModel_A100_PCIe_80GB,
				0,
				40,
				map[timeslicing.ProfileName]int{},
				map[timeslicing.ProfileName]int{
					"20gb": 2,
				},
			),
			requiredSlices: map[gpu.Slice]int{
				timeslicing.ProfileName("20gb"): 2,
			},
			expectedUpdate: false,
			expectedGeometry: map[gpu.Slice]int{
				timeslicing.ProfileName("20gb"): 2,
			},
		},
		{
			name: "GPU is full, should not update geometry",
			gpu: timeslicing.NewGpuOrPanic(
				gpu.GPUModel_A100_PCIe_80GB,
				0,
				40,
				map[timeslicing.ProfileName]int{
					"20gb": 2,
				},
				map[timeslicing.ProfileName]int{},
			),
			requiredSlices: map[gpu.Slice]int{
				timeslicing.ProfileName("10gb"): 1,
				timeslicing.ProfileName("20gb"): 1,
			},
			expectedUpdate: false,
			expectedGeometry: map[gpu.Slice]int{
				timeslicing.ProfileName("20gb"): 2,
			},
		},
		{
			name: "GPU has spare capacity, should create new slices without deleting existing ones",
			gpu: timeslicing.NewGpuOrPanic(
				gpu.GPUModel_A100_PCIe_80GB,
				0,
				60,
				map[timeslicing.ProfileName]int{
					"10gb": 1,
				},
				map[timeslicing.ProfileName]int{},
			),
			requiredSlices: map[gpu.Slice]int{
				timeslicing.ProfileName("10gb"): 1,
				timeslicing.ProfileName("20gb"): 2,
			},
			expectedUpdate: true,
			expectedGeometry: map[gpu.Slice]int{
				timeslicing.ProfileName("10gb"): 2,
				timeslicing.ProfileName("20gb"): 2,
			},
		},
		{
			name: "Created slices should never exceed the max GPU memory",
			gpu: timeslicing.NewFullGPU(
				gpu.GPUModel_A100_PCIe_80GB,
				0,
				40,
			),
			requiredSlices: map[gpu.Slice]int{
				timeslicing.ProfileName("10gb"): 5,
			},
			expectedUpdate: true,
			expectedGeometry: map[gpu.Slice]int{
				timeslicing.ProfileName("10gb"): 4,
			},
		},
		{
			name: "GPU has spare capacity, smaller profiles should be created first",
			gpu: timeslicing.NewFullGPU(
				gpu.GPUModel_A100_PCIe_80GB,
				0,
				40,
			),
			requiredSlices: map[gpu.Slice]int{
				timeslicing.ProfileName("20gb"): 2,
				timeslicing.ProfileName("10gb"): 2,
				timeslicing.ProfileName("5gb"):  2,
			},
			expectedUpdate: true,
			expectedGeometry: map[gpu.Slice]int{
				timeslicing.ProfileName("5gb"):  2,
				timeslicing.ProfileName("10gb"): 2,
			},
		},
		{
			name: "GPU with free devices, should delete them to make up space for requested slices",
			gpu: timeslicing.NewGpuOrPanic(
				gpu.GPUModel_A100_PCIe_80GB,
				0,
				40,
				map[timeslicing.ProfileName]int{
					"20gb": 1,
				},
				map[timeslicing.ProfileName]int{
					"10gb": 2,
				},
			),
			requiredSlices: map[gpu.Slice]int{
				timeslicing.ProfileName("20gb"): 1,
			},
			expectedUpdate: true,
			expectedGeometry: map[gpu.Slice]int{
				timeslicing.ProfileName("20gb"): 2,
			},
		},
		{
			name: "GPU free devices shouldn't be deleted if GPU has spare capacity for creating requested slices",
			gpu: timeslicing.NewGpuOrPanic(
				gpu.GPUModel_A100_PCIe_80GB,
				0,
				40,
				map[timeslicing.ProfileName]int{
					"10gb": 2,
				},
				map[timeslicing.ProfileName]int{},
			),
			requiredSlices: map[gpu.Slice]int{
				timeslicing.ProfileName("20gb"): 1,
			},
			expectedUpdate: true,
			expectedGeometry: map[gpu.Slice]int{
				timeslicing.ProfileName("10gb"): 2,
				timeslicing.ProfileName("20gb"): 1,
			},
		},
		{
			name: "GPU with free devices, should delete different slice sizes to make up space",
			gpu: timeslicing.NewGpuOrPanic(
				gpu.GPUModel_A100_PCIe_80GB,
				0,
				45,
				map[timeslicing.ProfileName]int{
					"20gb": 1,
				},
				map[timeslicing.ProfileName]int{
					"10gb": 1,
					"15gb": 1,
				},
			),
			requiredSlices: map[gpu.Slice]int{
				timeslicing.ProfileName("20gb"): 1,
			},
			expectedUpdate: true,
			expectedGeometry: map[gpu.Slice]int{
				timeslicing.ProfileName("20gb"): 2,
			},
		},
		{
			name: "GPU with free devices, should remain unchanged if required slices cannot be created",
			gpu: timeslicing.NewGpuOrPanic(
				gpu.GPUModel_A100_PCIe_80GB,
				0,
				45,
				map[timeslicing.ProfileName]int{
					"20gb": 1,
				},
				map[timeslicing.ProfileName]int{
					"10gb": 1,
					"15gb": 1,
				},
			),
			requiredSlices: map[gpu.Slice]int{
				timeslicing.ProfileName("30gb"): 1,
				timeslicing.ProfileName("31gb"): 2,
				timeslicing.ProfileName("32gb"): 2,
			},
			expectedUpdate: false,
			expectedGeometry: map[gpu.Slice]int{
				timeslicing.ProfileName("20gb"): 1,
				timeslicing.ProfileName("10gb"): 1,
				timeslicing.ProfileName("15gb"): 1,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.gpu
			updated := g.UpdateGeometryFor(tt.requiredSlices)
			assert.Equal(t, tt.expectedUpdate, updated)
			assert.Equal(t, tt.expectedGeometry, g.GetGeometry())
		})
	}
}

func TestGPU__Clone(t *testing.T) {
	testCases := []struct {
		name string
		gpu  timeslicing.GPU
	}{
		{
			name: "Full GPU",
			gpu:  timeslicing.NewFullGPU(gpu.GPUModel_A100_SXM4_40GB, 0, 10),
		},
		{
			name: "Partitioned GPU",
			gpu: timeslicing.NewGpuOrPanic(
				gpu.GPUModel_A100_PCIe_80GB,
				0,
				100,
				map[timeslicing.ProfileName]int{
					"20gb": 1,
					"10gb": 2,
				},
				map[timeslicing.ProfileName]int{
					"10gb": 1,
					"15gb": 1,
				},
			),
		},
		{
			name: "Used/Free profiles are nil",
			gpu: timeslicing.GPU{
				Model:        gpu.GPUModel_A100_PCIe_80GB,
				Index:        0,
				MemoryGB:     100,
				UsedProfiles: nil,
				FreeProfiles: nil,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			clone := tt.gpu.Clone()
			assert.Equalf(t, tt.gpu, clone, "Cloned GPU should be equal to original")
		})
	}
}
