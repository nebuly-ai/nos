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

package mps_test

import (
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/mps"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/slicing"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestSliceCalculator(t *testing.T) {
	testCases := []struct {
		name     string
		pod      v1.Pod
		expected map[gpu.Slice]int
	}{
		{
			name:     "Empty pod",
			pod:      v1.Pod{},
			expected: map[gpu.Slice]int{},
		},
		{
			name: "Should include only time-slicing profiles",
			pod: factory.BuildPod("ns-1", "pd-1").WithContainer(
				factory.BuildContainer("c-1", "im-1").
					WithScalarResourceRequest(constant.ResourceNvidiaGPU, 1).
					WithScalarResourceRequest(v1.ResourceCPU, 2).
					WithScalarResourceRequest(mig.Profile1g5gb.AsResourceName(), 2).
					WithScalarResourceRequest(slicing.ProfileName("nvidia.com/gpu-10gb").AsResourceName(), 1).
					Get(),
			).Get(),
			expected: map[gpu.Slice]int{
				slicing.ProfileName("nvidia.com/gpu-10gb"): 1,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			slices := mps.NewSliceCalculator().GetRequestedSlices(tt.pod)
			assert.Equal(t, tt.expected, slices)
		})
	}
}
