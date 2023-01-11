/*
 * Copyright 2023 Nebuly.ai.
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

package mig_test

import (
	mig_partitioner "github.com/nebuly-ai/nos/internal/partitioning/mig"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/mig"
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
			name:      "Empty list",
			resources: map[v1.ResourceName]int64{},
			expected:  map[gpu.Slice]int{},
		},
		{
			name: "Should include only MIG profiles",
			resources: map[v1.ResourceName]int64{
				constant.ResourceNvidiaGPU:         1,
				v1.ResourceCPU:                     2,
				mig.Profile1g5gb.AsResourceName():  2,
				mig.Profile7g40gb.AsResourceName(): 1,
			},
			expected: map[gpu.Slice]int{
				mig.Profile1g5gb:  2,
				mig.Profile7g40gb: 1,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			filtered := mig_partitioner.NewSliceFilter().ExtractSlices(tt.resources)
			assert.Equal(t, tt.expected, filtered)
		})
	}
}
