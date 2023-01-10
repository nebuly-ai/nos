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
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/slicing"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestSliceFilter__ExtractSlices(t *testing.T) {
	testCases := []struct {
		name      string
		resources map[v1.ResourceName]int64
		expected  map[gpu.Slice]int
	}{
		{
			name:      "Empty resources",
			resources: map[v1.ResourceName]int64{},
			expected:  map[gpu.Slice]int{},
		},
		{
			name: "Should include only time-slicing profiles",
			resources: map[v1.ResourceName]int64{
				constant.ResourceNvidiaGPU:                   1,
				v1.ResourceCPU:                               2,
				slicing.ProfileName("10gb").AsResourceName(): 1,
				slicing.ProfileName("30gb").AsResourceName(): 2,
			},
			expected: map[gpu.Slice]int{
				slicing.ProfileName("10gb"): 1,
				slicing.ProfileName("30gb"): 2,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			slices := mps.NewSliceFilter().ExtractSlices(tt.resources)
			assert.Equal(t, tt.expected, slices)
		})
	}
}
