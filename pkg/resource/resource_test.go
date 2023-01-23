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

package resource

import (
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"testing"
)

const (
	customResourceName v1.ResourceName = "nebuly.com/custom-resource"
)

func TestSum(t *testing.T) {
	tests := []struct {
		name     string
		r1       framework.Resource
		r2       framework.Resource
		expected framework.Resource
	}{
		{
			name:     "empty resources",
			r1:       framework.Resource{},
			r2:       framework.Resource{},
			expected: framework.Resource{},
		},
		{
			name: "one resource is empty",
			r1: framework.Resource{
				MilliCPU:         10,
				Memory:           20,
				EphemeralStorage: 10,
				AllowedPodNumber: 5,
				ScalarResources: map[v1.ResourceName]int64{
					constant.ResourceNvidiaGPU: 1,
				},
			},
			r2: framework.Resource{},
			expected: framework.Resource{
				MilliCPU:         10,
				Memory:           20,
				EphemeralStorage: 10,
				AllowedPodNumber: 5,
				ScalarResources: map[v1.ResourceName]int64{
					constant.ResourceNvidiaGPU: 1,
				},
			},
		},
		{
			name: "resources with different scalars",
			r1: framework.Resource{
				MilliCPU:         10,
				Memory:           20,
				EphemeralStorage: 10,
				AllowedPodNumber: 5,
				ScalarResources: map[v1.ResourceName]int64{
					customResourceName: 1,
				},
			},
			r2: framework.Resource{
				MilliCPU:         20,
				Memory:           20,
				EphemeralStorage: 15,
				AllowedPodNumber: 1,
				ScalarResources: map[v1.ResourceName]int64{
					customResourceName:         1,
					constant.ResourceNvidiaGPU: 3,
				},
			},
			expected: framework.Resource{
				MilliCPU:         30,
				Memory:           40,
				EphemeralStorage: 25,
				AllowedPodNumber: 6,
				ScalarResources: map[v1.ResourceName]int64{
					customResourceName:         2,
					constant.ResourceNvidiaGPU: 3,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Sum(tt.r1, tt.r2)
			assert.Equal(t, tt.expected, res)
		})
	}
}

func TestSubtract(t *testing.T) {
	tests := []struct {
		name     string
		r1       framework.Resource
		r2       framework.Resource
		expected framework.Resource
	}{
		{
			name:     "empty resources",
			r1:       framework.Resource{},
			r2:       framework.Resource{},
			expected: framework.Resource{},
		},
		{
			name: "r1 is empty",
			r1:   framework.Resource{},
			r2: framework.Resource{
				MilliCPU:         10,
				Memory:           20,
				EphemeralStorage: 10,
				AllowedPodNumber: 5,
				ScalarResources: map[v1.ResourceName]int64{
					customResourceName: 1,
				},
			},
			expected: framework.Resource{
				MilliCPU:         -10,
				Memory:           -20,
				EphemeralStorage: -10,
				AllowedPodNumber: -5,
				ScalarResources: map[v1.ResourceName]int64{
					customResourceName: -1,
				},
			},
		},
		{
			name: "resources with different scalars, result values can be negative",
			r1: framework.Resource{
				MilliCPU:         100,
				Memory:           10,
				EphemeralStorage: 10,
				AllowedPodNumber: 6,
				ScalarResources: map[v1.ResourceName]int64{
					customResourceName: 3,
				},
			},
			r2: framework.Resource{
				MilliCPU:         10,
				Memory:           20,
				EphemeralStorage: 10,
				AllowedPodNumber: 5,
				ScalarResources: map[v1.ResourceName]int64{
					customResourceName:         1,
					constant.ResourceNvidiaGPU: 5,
				},
			},
			expected: framework.Resource{
				MilliCPU:         90,
				Memory:           -10,
				EphemeralStorage: 0,
				AllowedPodNumber: 1,
				ScalarResources: map[v1.ResourceName]int64{
					customResourceName:         2,
					constant.ResourceNvidiaGPU: -5,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Subtract(tt.r1, tt.r2)
			assert.Equal(t, tt.expected, res)
		})
	}
}

func TestSubtractNonNegative(t *testing.T) {
	tests := []struct {
		name     string
		r1       framework.Resource
		r2       framework.Resource
		expected framework.Resource
	}{
		{
			name:     "empty resources",
			r1:       framework.Resource{},
			r2:       framework.Resource{},
			expected: framework.Resource{},
		},
		{
			name: "r1 is empty",
			r1:   framework.Resource{},
			r2: framework.Resource{
				MilliCPU:         10,
				Memory:           20,
				EphemeralStorage: 10,
				AllowedPodNumber: 5,
				ScalarResources: map[v1.ResourceName]int64{
					customResourceName: 1,
				},
			},
			expected: framework.Resource{
				MilliCPU:         0,
				Memory:           0,
				EphemeralStorage: 0,
				AllowedPodNumber: 0,
				ScalarResources: map[v1.ResourceName]int64{
					customResourceName: 0,
				},
			},
		},
		{
			name: "resources with different scalars, result values must be >= 0",
			r1: framework.Resource{
				MilliCPU:         100,
				Memory:           10,
				EphemeralStorage: 10,
				AllowedPodNumber: 6,
				ScalarResources: map[v1.ResourceName]int64{
					customResourceName: 3,
				},
			},
			r2: framework.Resource{
				MilliCPU:         10,
				Memory:           20,
				EphemeralStorage: 10,
				AllowedPodNumber: 5,
				ScalarResources: map[v1.ResourceName]int64{
					customResourceName:         1,
					constant.ResourceNvidiaGPU: 5,
				},
			},
			expected: framework.Resource{
				MilliCPU:         90,
				Memory:           0,
				EphemeralStorage: 0,
				AllowedPodNumber: 1,
				ScalarResources: map[v1.ResourceName]int64{
					customResourceName:         2,
					constant.ResourceNvidiaGPU: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := SubtractNonNegative(tt.r1, tt.r2)
			assert.Equal(t, tt.expected, res)
		})
	}
}
