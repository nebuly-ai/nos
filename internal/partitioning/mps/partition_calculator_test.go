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

package mps_test

import (
	"fmt"
	"github.com/nebuly-ai/nos/internal/partitioning/core"
	"github.com/nebuly-ai/nos/internal/partitioning/mps"
	"github.com/nebuly-ai/nos/internal/partitioning/state"
	"github.com/nebuly-ai/nos/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/slicing"
	"github.com/nebuly-ai/nos/pkg/resource"
	"github.com/nebuly-ai/nos/pkg/test/factory"
	"github.com/nebuly-ai/nos/pkg/test/mocks"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"testing"
)

func newSliceNodeOrPanic(node v1.Node) *slicing.Node {
	nodeInfo := framework.NewNodeInfo()
	nodeInfo.SetNode(&node)
	sliceNode, err := slicing.NewNode(*nodeInfo)
	if err != nil {
		panic(err)
	}
	return &sliceNode
}

func TestPartitionCalculator__GetPartitioning(t *testing.T) {
	testCases := []struct {
		name     string
		node     core.PartitionableNode
		expected state.NodePartitioning
	}{
		{
			name:     "Node is not MPS node, should return empty partitioning",
			node:     &mocks.PartitionableNode{},
			expected: state.NodePartitioning{GPUs: make([]state.GPUPartitioning, 0)},
		},
		{
			name: "MPS node without any slice",
			node: newSliceNodeOrPanic(
				factory.BuildNode("node-1").WithLabels(map[string]string{
					constant.LabelNvidiaMemory:  "2000",
					constant.LabelNvidiaProduct: string(gpu.GPUModel_A30),
					constant.LabelNvidiaCount:   "2",
				}).Get(),
			),
			expected: state.NodePartitioning{
				GPUs: []state.GPUPartitioning{
					{
						GPUIndex:  0,
						Resources: map[v1.ResourceName]int{},
					},
					{
						GPUIndex:  1,
						Resources: map[v1.ResourceName]int{},
					},
				},
			},
		},
		{
			name: "MPS node with slice annotations",
			node: newSliceNodeOrPanic(
				factory.BuildNode("node-1").WithLabels(map[string]string{
					constant.LabelNvidiaMemory:  "20000",
					constant.LabelNvidiaProduct: string(gpu.GPUModel_A30),
					constant.LabelNvidiaCount:   "2",
				}).WithAnnotations(map[string]string{
					fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 1, "20gb", resource.StatusUsed): "1",
					fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, "10gb", resource.StatusFree): "2",
				}).Get(),
			),
			expected: state.NodePartitioning{
				GPUs: []state.GPUPartitioning{
					{
						GPUIndex: 0,
						Resources: map[v1.ResourceName]int{
							slicing.ProfileName("10gb").AsResourceName(): 2,
						},
					},
					{
						GPUIndex: 1,
						Resources: map[v1.ResourceName]int{
							slicing.ProfileName("20gb").AsResourceName(): 1,
						},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			partitioning := mps.NewPartitionCalculator().GetPartitioning(tt.node)
			assert.True(t, tt.expected.Equal(partitioning))
		})
	}
}
