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

package slicing_test

import (
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/slicing"
	"github.com/nebuly-ai/nos/pkg/resource"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestAnnotationConversions(t *testing.T) {
	devices := gpu.DeviceList{
		{
			Device: resource.Device{
				ResourceName: "nvidia.com/gpu-10gb",
				DeviceId:     "id-1",
				Status:       resource.StatusUsed,
			},
			GpuIndex: 0,
		},
		{
			Device: resource.Device{
				ResourceName: "nvidia.com/gpu-10gb",
				DeviceId:     "id-1",
				Status:       resource.StatusUsed,
			},
			GpuIndex: 0,
		},
		{
			Device: resource.Device{
				ResourceName: "nvidia.com/gpu",
				DeviceId:     "id-3",
				Status:       resource.StatusUsed,
			},
			GpuIndex: 3,
		},
		{
			Device: resource.Device{
				ResourceName: "nvidia.com/gpu-20gb",
				DeviceId:     "id-1",
				Status:       resource.StatusFree,
			},
			GpuIndex: 0,
		},
		{
			Device: resource.Device{
				ResourceName: "nvidia.com/gpu-10gb",
				DeviceId:     "id-2",
				Status:       resource.StatusFree,
			},
			GpuIndex: 1,
		},
	}

	// From devices to annotations
	slicingAnnotations := devices.AsStatusAnnotation(slicing.ExtractProfileNameStr)
	stringAnnotations := make(map[string]string)
	for _, a := range slicingAnnotations {
		stringAnnotations[a.String()] = a.GetValue()
	}

	// From annotations to devices
	node := v1.Node{}
	node.Annotations = stringAnnotations
	parsedStatusAnnotations, _ := gpu.ParseNodeAnnotations(node)

	// Check that the devices are the same
	assert.ElementsMatch(t, slicingAnnotations, parsedStatusAnnotations)
}

func TestExtractGpuId(t *testing.T) {
	testCases := []struct {
		name     string
		id       string
		expected string
	}{
		{
			name:     "empty",
			id:       "",
			expected: "",
		},
		{
			name:     "non-replica ID, should return the same ID",
			id:       "id-1",
			expected: "id-1",
		},
		{
			name:     "replica ID, should return the ID without the replica suffix",
			id:       "id-1::1",
			expected: "id-1",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			id := slicing.ExtractGpuId(tt.id)
			assert.Equal(t, tt.expected, id)
		})
	}
}
