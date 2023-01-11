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

package core_test

import (
	"context"
	"errors"
	"github.com/nebuly-ai/nos/internal/partitioning/core"
	"github.com/nebuly-ai/nos/internal/partitioning/state"
	"github.com/nebuly-ai/nos/pkg/test/factory"
	"github.com/nebuly-ai/nos/pkg/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestActuator__Apply(t *testing.T) {
	testCases := []struct {
		name                      string
		snapshotPartitioningState state.PartitioningState
		plan                      core.PartitioningPlan

		mockedClientNode        v1.Node
		mockedPartitionerReturn error

		expectedRes bool
		expectedErr bool
	}{
		{
			name: "Empty plan, should do nothing",
			snapshotPartitioningState: map[string]state.NodePartitioning{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								"nvidia.com/gpu-10gb": 1,
							},
						},
					},
				},
			},
			plan: core.NewPartitioningPlan(state.PartitioningState{}),

			mockedClientNode:        v1.Node{},
			mockedPartitionerReturn: nil,

			expectedRes: false,
			expectedErr: false,
		},
		{
			name: "Snapshot equal to plan, should do nothing",
			snapshotPartitioningState: map[string]state.NodePartitioning{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								"nvidia.com/gpu-10gb": 1,
							},
						},
					},
				},
			},
			plan: core.NewPartitioningPlan(state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								"nvidia.com/gpu-10gb": 1,
							},
						},
					},
				},
			}),

			mockedClientNode:        v1.Node{},
			mockedPartitionerReturn: nil,

			expectedRes: false,
			expectedErr: false,
		},
		{
			name: "Node not found, should do nothing and return error",
			snapshotPartitioningState: map[string]state.NodePartitioning{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								"nvidia.com/gpu-10gb": 1,
							},
						},
					},
				},
			},
			plan: core.NewPartitioningPlan(state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								"nvidia.com/gpu-10gb": 2,
							},
						},
					},
				},
			}),

			mockedClientNode:        v1.Node{},
			mockedPartitionerReturn: nil,

			expectedRes: false,
			expectedErr: true,
		},
		{
			name: "Partitioner returns error when applying plan, should do nothing and return error",
			snapshotPartitioningState: map[string]state.NodePartitioning{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								"nvidia.com/gpu-10gb": 1,
							},
						},
					},
				},
			},
			plan: core.NewPartitioningPlan(state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								"nvidia.com/gpu-10gb": 2,
							},
						},
					},
				},
			}),

			mockedClientNode:        factory.BuildNode("node-1").Get(),
			mockedPartitionerReturn: errors.New("error"),

			expectedRes: false,
			expectedErr: true,
		},
		{
			name: "Plan is applied",
			snapshotPartitioningState: map[string]state.NodePartitioning{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								"nvidia.com/gpu-10gb": 1,
							},
						},
					},
				},
			},
			plan: core.NewPartitioningPlan(state.PartitioningState{
				"node-1": {
					GPUs: []state.GPUPartitioning{
						{
							GPUIndex: 0,
							Resources: map[v1.ResourceName]int{
								"nvidia.com/gpu-10gb": 2,
							},
						},
					},
				},
			}),

			mockedClientNode:        factory.BuildNode("node-1").Get(),
			mockedPartitionerReturn: nil,

			expectedRes: true,
			expectedErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockPartitioner := mocks.NewPartitioner(t)
			mockPartitioner.On(
				"ApplyPartitioning",
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).Return(tt.mockedPartitionerReturn).Maybe()
			mockClient := fake.NewClientBuilder().WithObjects(&tt.mockedClientNode).Build()
			actuator := core.NewActuator(mockClient, mockPartitioner)

			mockSnapshot := mocks.NewSnapshot(t)
			mockSnapshot.On("GetPartitioningState").Return(tt.snapshotPartitioningState).Maybe()

			res, err := actuator.Apply(context.Background(), mockSnapshot, tt.plan)
			assert.Equal(t, tt.expectedRes, res)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
