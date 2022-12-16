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
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
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
	timeSlicingAnnotations := timeslicing.ComputeStatusAnnotations(devices)
	stringAnnotations := make(map[string]string)
	for _, a := range timeSlicingAnnotations {
		stringAnnotations[a.String()] = a.GetValue()
	}

	// From annotations to devices
	node := v1.Node{}
	node.Annotations = stringAnnotations
	parsedStatusAnnotations, _ := timeslicing.ParseNodeAnnotations(node)

	// Check that the devices are the same
	assert.ElementsMatch(t, timeSlicingAnnotations, parsedStatusAnnotations)
}
