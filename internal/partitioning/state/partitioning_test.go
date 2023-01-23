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

package state_test

import (
	"github.com/nebuly-ai/nos/internal/partitioning/state"
	"github.com/nebuly-ai/nos/pkg/gpu/mig"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestPartitioningState__Equal(t *testing.T) {
	testCases := []struct {
		name              string
		partitioningState state.PartitioningState
		other             state.PartitioningState
		expected          bool
	}{
		{
			name:              "Empty partitioning states",
			partitioningState: state.PartitioningState{},
			other:             state.PartitioningState{},
			expected:          true,
		},
		{
			name: "Different GPUs number",
			partitioningState: state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName(): 1,
							},
						},
					},
				},
			},
			other: state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName(): 1,
							},
						},
						{
							GPUIndex: 1,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName(): 1,
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Different nodes number",
			partitioningState: state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName(): 1,
							},
						},
					},
				},
				"node-2": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName(): 1,
							},
						},
					},
				},
			},
			other: state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName(): 1,
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Same partitioning but with different quantities",
			partitioningState: state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName(): 1,
							},
						},
					},
				},
			},
			other: state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName(): 2,
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Same partitioning but with different GPU indexes",
			partitioningState: state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName(): 1,
							},
						},
					},
				},
			},
			other: state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 1,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName(): 1,
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Same partitioning, different orders",
			partitioningState: state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName():  2,
								mig.Profile4g24gb.AsResourceName(): 4,
							},
						},
						{
							GPUIndex: 1,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName():  1,
								mig.Profile2g12gb.AsResourceName(): 1,
							},
						},
					},
				},
				"node-2": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile2g10gb.AsResourceName(): 2,
							},
						},
					},
				},
			},
			other: state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 1,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName():  1,
								mig.Profile2g12gb.AsResourceName(): 1,
							},
						},
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile1g5gb.AsResourceName():  2,
								mig.Profile4g24gb.AsResourceName(): 4,
							},
						},
					},
				},
				"node-2": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								mig.Profile2g10gb.AsResourceName(): 2,
							},
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.partitioningState.Equal(tt.other)
			assert.Equal(t, tt.expected, res)
		})
	}
}
