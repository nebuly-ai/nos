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

package slicing_test

import (
	"fmt"
	"github.com/nebuly-ai/nos/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/slicing"
	"github.com/nebuly-ai/nos/pkg/resource"
	"github.com/nebuly-ai/nos/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"testing"
)

func TestNewNode(t *testing.T) {
	testCases := []struct {
		name        string
		node        v1.Node
		expected    slicing.Node
		errExpected bool
	}{
		{
			name: "node without GPU count label",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaProduct: "foo",
				constant.LabelNvidiaMemory:  "10",
			}).Get(),
			errExpected: true,
		},
		{
			name: "node without GPU memory label",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaProduct: "foo",
				constant.LabelNvidiaCount:   "1",
			}).Get(),
			errExpected: true,
		},
		{
			name: "node without GPU model label",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaCount:  "1",
				constant.LabelNvidiaMemory: "1",
			}).Get(),
			errExpected: true,
		},
		{
			name: "no status labels, should return all GPUs",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaProduct: "foo",
				constant.LabelNvidiaCount:   "3", // Number of returned GPUs
				constant.LabelNvidiaMemory:  "2000",
			}).Get(),
			expected: slicing.Node{
				Name: "node-1",
				GPUs: []slicing.GPU{
					slicing.NewFullGPU(
						"foo",
						0,
						2,
					),
					slicing.NewFullGPU(
						"foo",
						1,
						2,
					),
					slicing.NewFullGPU(
						"foo",
						2,
						2,
					),
				},
			},
		},
		{
			name: "free and used labels",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaProduct: "foo",
				constant.LabelNvidiaCount:   "3",
				constant.LabelNvidiaMemory:  "40000",
			}).WithAnnotations(map[string]string{
				fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, "10gb", resource.StatusFree): "2",
				fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 1, "20gb", resource.StatusUsed): "1",
			}).Get(),
			expected: slicing.Node{
				Name: "node-1",
				GPUs: []slicing.GPU{
					slicing.NewGpuOrPanic(
						"foo",
						0,
						40,
						map[slicing.ProfileName]int{},
						map[slicing.ProfileName]int{"10gb": 2},
					),
					slicing.NewGpuOrPanic(
						"foo",
						1,
						40,
						map[slicing.ProfileName]int{"20gb": 1},
						map[slicing.ProfileName]int{},
					),
					slicing.NewGpuOrPanic(
						"foo",
						2,
						40,
						map[slicing.ProfileName]int{},
						map[slicing.ProfileName]int{},
					),
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			nodeInfo := framework.NewNodeInfo()
			nodeInfo.SetNode(&tt.node)
			node, err := slicing.NewNode(*nodeInfo)
			if tt.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Name, node.Name)
				assert.ElementsMatch(t, tt.expected.GPUs, node.GPUs)
			}
		})
	}
}

func TestNode__GetGeometry(t *testing.T) {
	testCases := []struct {
		name     string
		node     slicing.Node
		expected map[gpu.Slice]int
	}{
		{
			name:     "Empty node",
			node:     slicing.Node{},
			expected: make(map[gpu.Slice]int),
		},
		{
			name: "Geometry is the sum of all GPUs Geometry",
			node: slicing.Node{
				Name: "node-1",
				GPUs: []slicing.GPU{
					slicing.NewGpuOrPanic(
						gpu.GPUModel_A100_PCIe_80GB,
						0,
						80,
						map[slicing.ProfileName]int{"10gb": 2},
						map[slicing.ProfileName]int{"20gb": 1},
					),
					slicing.NewGpuOrPanic(
						gpu.GPUModel_A30,
						0,
						30,
						map[slicing.ProfileName]int{"4gb": 1},
						map[slicing.ProfileName]int{"20gb": 1},
					),
				},
			},
			expected: map[gpu.Slice]int{
				slicing.ProfileName("10gb"): 2,
				slicing.ProfileName("4gb"):  1,
				slicing.ProfileName("20gb"): 2,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.node.Geometry())
		})
	}
}

func TestNode__HasFreeCapacity(t *testing.T) {
	testCases := []struct {
		name     string
		nodeGPUs []slicing.GPU
		expected bool
	}{
		{
			name:     "Node without GPUs",
			nodeGPUs: make([]slicing.GPU, 0),
			expected: false,
		},
		{
			name: "Node with GPU without any free or used device",
			nodeGPUs: []slicing.GPU{
				slicing.NewFullGPU(
					gpu.GPUModel_A30,
					0,
					10,
				),
			},
			expected: true,
		},
		{
			name: "Node with GPU with free slices",
			nodeGPUs: []slicing.GPU{
				slicing.NewGpuOrPanic(
					gpu.GPUModel_A30,
					0,
					10,
					map[slicing.ProfileName]int{"5gb": 1},
					map[slicing.ProfileName]int{"5gb": 1},
				),
			},
			expected: true,
		},
		{
			name: "Node with just a single GPU with just used slices, but there is space to create more slices",
			nodeGPUs: []slicing.GPU{
				slicing.NewGpuOrPanic(
					gpu.GPUModel_A30,
					0,
					80,
					map[slicing.ProfileName]int{
						"5gb": 1,
					},
					make(map[slicing.ProfileName]int),
				),
			},
			expected: true,
		},
		{
			name: "Node with just a single GPU with just used slices, and there isn't space to create more slices",
			nodeGPUs: []slicing.GPU{
				slicing.NewGpuOrPanic(
					gpu.GPUModel_A30,
					0,
					20,
					map[slicing.ProfileName]int{
						"20gb": 1,
					},
					make(map[slicing.ProfileName]int),
				),
			},
			expected: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			n := slicing.Node{Name: "test", GPUs: tt.nodeGPUs}
			res := n.HasFreeCapacity()
			assert.Equal(t, tt.expected, res)
		})
	}
}

func TestNode_AddPod(t *testing.T) {
	testCases := []struct {
		name                       string
		node                       v1.Node
		pod                        v1.Pod
		expectedRequestedResources framework.Resource
		expectedUsedSlices         map[gpu.Slice]int
		expectedFreeSlices         map[gpu.Slice]int
		expectedErr                bool
	}{
		{
			name: "Adding a pod should update node info and used GPU slices",
			node: factory.BuildNode("node-1").
				WithLabels(map[string]string{
					constant.LabelNvidiaProduct: "foo",
					constant.LabelNvidiaCount:   "3",
					constant.LabelNvidiaMemory:  "40000",
				}).
				WithAnnotations(map[string]string{
					fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 1, "10gb", resource.StatusFree): "3",
				}).Get(),
			pod: factory.BuildPod("ns-1", "pd-1").WithContainer(
				factory.BuildContainer("c-1", "foo").
					WithCPUMilliRequest(1000).
					WithScalarResourceRequest(slicing.ProfileName("10gb").AsResourceName(), 1).
					Get(),
			).Get(),
			expectedRequestedResources: framework.Resource{
				MilliCPU: 1000,
				ScalarResources: map[v1.ResourceName]int64{
					slicing.ProfileName("10gb").AsResourceName(): 1,
				},
			},
			expectedUsedSlices: map[gpu.Slice]int{
				slicing.ProfileName("10gb"): 1,
			},
			expectedFreeSlices: map[gpu.Slice]int{
				slicing.ProfileName("10gb"): 2,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			nodeInfo := framework.NewNodeInfo()
			nodeInfo.SetNode(&tt.node)
			n, err := slicing.NewNode(*nodeInfo)
			if err != nil {
				panic(err)
			}

			err = n.AddPod(tt.pod)
			if tt.expectedErr {
				assert.Error(t, err)
				return
			}

			var freeSlices = make(map[gpu.Slice]int)
			var usedSlices = make(map[gpu.Slice]int)
			for _, g := range n.GPUs {
				for p, q := range g.UsedProfiles {
					usedSlices[p] += q
				}
				for p, q := range g.FreeProfiles {
					freeSlices[p] += q
				}
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRequestedResources, *n.NodeInfo().Requested)
			assert.Equal(t, tt.expectedUsedSlices, usedSlices)
			assert.Equal(t, tt.expectedFreeSlices, freeSlices)
		})
	}
}

func TestNode__UpdateGeometryFor(t *testing.T) {
	testCases := []struct {
		name   string
		node   v1.Node
		slices map[gpu.Slice]int

		expectedErr    bool
		expectedUpdate bool
	}{
		{
			name: "node without GPUs",
			node: factory.BuildNode("node-1").
				WithLabels(map[string]string{
					constant.LabelNvidiaProduct: "foo",
					constant.LabelNvidiaCount:   "0",
					constant.LabelNvidiaMemory:  "40000",
				}).
				Get(),
			slices: map[gpu.Slice]int{
				slicing.ProfileName("10gb"): 1,
			},
			expectedErr:    false,
			expectedUpdate: false,
		},
		{
			name: "no slices",
			node: factory.BuildNode("node-1").
				WithLabels(map[string]string{
					constant.LabelNvidiaProduct: "foo",
					constant.LabelNvidiaCount:   "2",
					constant.LabelNvidiaMemory:  "40000",
				}).
				Get(),
			slices:         map[gpu.Slice]int{},
			expectedErr:    false,
			expectedUpdate: false,
		},
		{
			name: "node with free capacity, should update",
			node: factory.BuildNode("node-1").
				WithLabels(map[string]string{
					constant.LabelNvidiaProduct: "foo",
					constant.LabelNvidiaCount:   "3",
					constant.LabelNvidiaMemory:  "40000",
				}).
				Get(),
			slices: map[gpu.Slice]int{
				slicing.ProfileName("10gb"): 1,
			},
			expectedErr:    false,
			expectedUpdate: true,
		},
		{
			name: "node without any free slice or capacity, should not update",
			node: factory.BuildNode("node-1").
				WithLabels(map[string]string{
					constant.LabelNvidiaProduct: "foo",
					constant.LabelNvidiaCount:   "1",
					constant.LabelNvidiaMemory:  "10000",
				}).
				WithAnnotations(map[string]string{
					fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, "10gb", resource.StatusUsed): "1",
				}).Get(),
			slices: map[gpu.Slice]int{
				slicing.ProfileName("10gb"): 1,
			},
			expectedErr:    false,
			expectedUpdate: false,
		},
		{
			name: "node already provides required slices, should not update",
			node: factory.BuildNode("node-1").
				WithLabels(map[string]string{
					constant.LabelNvidiaProduct: "foo",
					constant.LabelNvidiaCount:   "1",
					constant.LabelNvidiaMemory:  "40000",
				}).
				WithAnnotations(map[string]string{
					fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, "10gb", resource.StatusFree): "2",
					fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, "20gb", resource.StatusFree): "1",
				}).Get(),
			slices: map[gpu.Slice]int{
				slicing.ProfileName("10gb"): 1,
				slicing.ProfileName("20gb"): 1,
			},
			expectedErr:    false,
			expectedUpdate: false,
		},
		{
			name: "node with free profiles that can be deleted, should update",
			node: factory.BuildNode("node-1").
				WithLabels(map[string]string{
					constant.LabelNvidiaProduct: "foo",
					constant.LabelNvidiaCount:   "1",
					constant.LabelNvidiaMemory:  "40000",
				}).
				WithAnnotations(map[string]string{
					fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, "10gb", resource.StatusFree): "2",
				}).Get(),
			slices: map[gpu.Slice]int{
				slicing.ProfileName("20gb"): 1,
			},
			expectedErr:    false,
			expectedUpdate: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			nodeInfo := framework.NewNodeInfo()
			nodeInfo.SetNode(&tt.node)
			nodeInfo.Allocatable.ScalarResources = map[v1.ResourceName]int64{"nebuly.ai/foo": 1}
			n, err := slicing.NewNode(*nodeInfo)
			if err != nil {
				panic(err)
			}

			updated, err := n.UpdateGeometryFor(tt.slices)
			assert.Equal(t, tt.expectedUpdate, updated)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNode__Clone(t *testing.T) {
	testCases := []struct {
		name string
		node v1.Node
	}{
		{
			name: "Empty node, no GPUs",
			node: factory.BuildNode("node").WithLabels(map[string]string{
				constant.LabelNvidiaProduct: "foo",
				constant.LabelNvidiaCount:   "0",
				constant.LabelNvidiaMemory:  "40000",
			}).Get(),
		},
		{
			name: "Node with GPUs",
			node: factory.BuildNode("node").
				WithLabels(map[string]string{
					constant.LabelNvidiaProduct: "foo",
					constant.LabelNvidiaCount:   "2",
					constant.LabelNvidiaMemory:  "40000",
				}).WithAnnotations(
				map[string]string{
					fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, "10gb", resource.StatusFree): "2",
					fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, "20gb", resource.StatusFree): "1",
				}).Get(),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			nodeInfo := framework.NewNodeInfo()
			nodeInfo.SetNode(&tt.node)
			nodeInfo.Allocatable.ScalarResources = map[v1.ResourceName]int64{"nebuly.ai/foo": 1}
			n, err := slicing.NewNode(*nodeInfo)
			if err != nil {
				panic(err)
			}
			clone := n.Clone()
			assert.Equal(t, &n, clone)
		})
	}
}
