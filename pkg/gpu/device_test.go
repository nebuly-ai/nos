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

package gpu_test

import (
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/resource"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestDeviceList__AsStatusAnnotation(t *testing.T) {
	testCases := []struct {
		name     string
		list     gpu.DeviceList
		expected gpu.StatusAnnotationList
	}{
		{
			name:     "empty",
			list:     gpu.DeviceList{},
			expected: gpu.StatusAnnotationList{},
		},
		{
			name: "multiple devices, same GPU index, same resource ID, same status, different profiles",
			list: gpu.DeviceList{
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu-1",
						DeviceId:     "1",
						Status:       resource.StatusFree,
					},
					GpuIndex: 0,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu-2",
						DeviceId:     "1",
						Status:       resource.StatusFree,
					},
					GpuIndex: 0,
				},
			},
			expected: gpu.StatusAnnotationList{
				{
					ProfileName: "nvidia.com/gpu-1",
					Index:       0,
					Status:      resource.StatusFree,
					Quantity:    1,
				},
				{
					ProfileName: "nvidia.com/gpu-2",
					Index:       0,
					Status:      resource.StatusFree,
					Quantity:    1,
				},
			},
		},
		{
			name: "multiple devices, statuses, indexes and IDs",
			list: gpu.DeviceList{
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu-1",
						DeviceId:     "1",
						Status:       resource.StatusFree,
					},
					GpuIndex: 0,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu-2",
						DeviceId:     "1",
						Status:       resource.StatusUsed,
					},
					GpuIndex: 0,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu-2",
						DeviceId:     "2",
						Status:       resource.StatusUsed,
					},
					GpuIndex: 1,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/gpu-2",
						DeviceId:     "3",
						Status:       resource.StatusUsed,
					},
					GpuIndex: 1,
				},
			},
			expected: gpu.StatusAnnotationList{
				{
					ProfileName: "nvidia.com/gpu-1",
					Index:       0,
					Status:      resource.StatusFree,
					Quantity:    1,
				},
				{
					ProfileName: "nvidia.com/gpu-2",
					Index:       0,
					Status:      resource.StatusUsed,
					Quantity:    1,
				},
				{
					ProfileName: "nvidia.com/gpu-2",
					Index:       1,
					Status:      resource.StatusUsed,
					Quantity:    2,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			getProfileName := func(r v1.ResourceName) (string, error) {
				return string(r), nil
			}
			assert.ElementsMatch(t, tt.expected, tt.list.AsStatusAnnotation(getProfileName))
		})
	}
}
