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

package plan_test

import (
	"github.com/nebuly-ai/nos/internal/controllers/migagent/plan"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/mig"
	"github.com/nebuly-ai/nos/pkg/resource"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeleteOperation__Equal(t *testing.T) {
	testCases := []struct {
		name     string
		deleteOp plan.DeleteOperation
		other    plan.DeleteOperation
		expected bool
	}{
		{
			name:     "Empty op",
			deleteOp: plan.DeleteOperation{},
			other:    plan.DeleteOperation{},
			expected: true,
		},
		{
			name: "Op are equals, different order",
			deleteOp: plan.DeleteOperation{
				Resources: gpu.DeviceList{
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
							ResourceName: mig.Profile2g20gb.AsResourceName(),
							DeviceId:     "1",
							Status:       resource.StatusFree,
						},
						GpuIndex: 0,
					},
				},
			},
			other: plan.DeleteOperation{
				Resources: gpu.DeviceList{
					{
						Device: resource.Device{
							ResourceName: mig.Profile2g20gb.AsResourceName(),
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
			expected: true,
		},
		{
			name: "Op are *not* equals",
			deleteOp: plan.DeleteOperation{
				Resources: gpu.DeviceList{
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
			other: plan.DeleteOperation{
				Resources: gpu.DeviceList{
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
			expected: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.deleteOp.Equal(tt.other)
			swappedRes := tt.other.Equal(tt.deleteOp)
			assert.Equal(t, res, swappedRes, "equal function is not symmetric")
			assert.Equal(t, tt.expected, res)
		})
	}
}
