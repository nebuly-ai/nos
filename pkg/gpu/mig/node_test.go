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

package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"strconv"
	"testing"
)

func TestNewNode(t *testing.T) {
	testCases := []struct {
		name          string
		node          v1.Node
		expectedNode  Node
		expectedError bool
	}{
		{
			name: "Node without GPU annotations",
			node: v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Labels: map[string]string{
						constant.LabelNvidiaProduct: string(gpu.GPUModel_A100_SXM4_40GB),
					},
				},
			},
			expectedNode: Node{
				Name: "test-node",
				GPUs: make([]GPU, 0),
			},
			expectedError: false,
		},
		{
			name: "Node without GPU model label",
			node: v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Annotations: map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, Profile1g10gb, resource.StatusFree): "1",
					},
				},
			},
			expectedNode: Node{
				Name: "test-node",
				GPUs: make([]GPU, 0),
			},
			expectedError: false,
		},
		{
			name: "Node with unknown GPU model",
			node: v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Annotations: map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, Profile1g10gb, resource.StatusFree): "1",
					},
					Labels: map[string]string{
						constant.LabelNvidiaProduct: "unknown-gpu-model",
					},
				},
			},
			expectedError: true,
		},
		{
			name: "Node with multiple GPUs with used and free MIG device annotations",
			node: v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Annotations: map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, Profile1g5gb, resource.StatusFree):  "2",
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 0, Profile2g20gb, resource.StatusUsed): "3",
						fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, 1, Profile3g20gb, resource.StatusFree): "2",
					},
					Labels: map[string]string{
						constant.LabelNvidiaProduct: string(gpu.GPUModel_A30),
					},
				},
			},
			expectedNode: Node{
				Name: "test-node",
				GPUs: []GPU{
					{
						index:                0,
						model:                gpu.GPUModel_A30,
						allowedMigGeometries: GetKnownGeometries()[gpu.GPUModel_A30],
						usedMigDevices: map[ProfileName]int{
							Profile2g20gb: 3,
						},
						freeMigDevices: map[ProfileName]int{
							Profile1g5gb: 2,
						},
					},
					{
						index:                1,
						model:                gpu.GPUModel_A30,
						allowedMigGeometries: GetKnownGeometries()[gpu.GPUModel_A30],
						usedMigDevices:       map[ProfileName]int{},
						freeMigDevices: map[ProfileName]int{
							Profile3g20gb: 2,
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Node with MIG-enabled GPUs, but without any MIG profile created",
			node: v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Labels: map[string]string{
						constant.LabelNvidiaProduct: string(gpu.GPUModel_A30),
						constant.LabelNvidiaCount:   strconv.Itoa(2),
					},
				},
			},
			expectedNode: Node{
				Name: "test-node",
				GPUs: []GPU{
					{
						index:                0,
						model:                gpu.GPUModel_A30,
						allowedMigGeometries: GetKnownGeometries()[gpu.GPUModel_A30],
						usedMigDevices:       map[ProfileName]int{},
						freeMigDevices:       map[ProfileName]int{},
					},
					{
						index:                1,
						model:                gpu.GPUModel_A30,
						allowedMigGeometries: GetKnownGeometries()[gpu.GPUModel_A30],
						usedMigDevices:       map[ProfileName]int{},
						freeMigDevices:       map[ProfileName]int{},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			nodeInfo := framework.NewNodeInfo()
			nodeInfo.SetNode(&tt.node)
			node, err := NewNode(*nodeInfo)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.expectedNode.Name, node.Name)
				assert.ElementsMatch(t, tt.expectedNode.GPUs, node.GPUs)
				assert.NoError(t, err)
			}
		})
	}
}

func TestNode__GetGeometry(t *testing.T) {
	testCases := []struct {
		name     string
		node     Node
		expected map[gpu.Slice]int
	}{
		{
			name:     "Empty node",
			node:     Node{},
			expected: make(map[gpu.Slice]int),
		},
		{
			name: "Geometry is the sum of all GPUs Geometry",
			node: Node{
				Name: "node-1",
				GPUs: []GPU{
					{
						index:                0,
						model:                gpu.GPUModel_A30,
						allowedMigGeometries: GetKnownGeometries()[gpu.GPUModel_A30],
						usedMigDevices: map[ProfileName]int{
							Profile4g24gb: 2,
							Profile1g5gb:  3,
						},
						freeMigDevices: map[ProfileName]int{
							Profile1g5gb: 1,
							Profile1g6gb: 1,
						},
					},
					{
						index:                1,
						model:                gpu.GPUModel_A100_SXM4_40GB,
						allowedMigGeometries: GetKnownGeometries()[gpu.GPUModel_A100_SXM4_40GB],
						usedMigDevices: map[ProfileName]int{
							Profile1g5gb:  3,
							Profile2g20gb: 1,
						},
						freeMigDevices: map[ProfileName]int{
							Profile1g5gb:  1,
							Profile3g20gb: 2,
						},
					},
				},
			},
			expected: map[gpu.Slice]int{
				Profile4g24gb: 2,
				Profile2g20gb: 1,
				Profile1g5gb:  8,
				Profile1g6gb:  1,
				Profile3g20gb: 2,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.node.Geometry())
		})
	}
}

func TestNode__UpdateGeometryFor(t *testing.T) {
	type gpuSpec struct {
		model gpu.Model
		index int
		used  map[ProfileName]int
		free  map[ProfileName]int
	}

	testCases := []struct {
		name             string
		nodeGPUs         []gpuSpec
		migProfiles      map[gpu.Slice]int
		expectedUpdated  bool
		expectedGeometry map[gpu.Slice]int
	}{
		{
			name:     "Node without GPUs",
			nodeGPUs: make([]gpuSpec, 0),
			migProfiles: map[gpu.Slice]int{
				Profile1g6gb: 1,
			},
			expectedUpdated:  false,
			expectedGeometry: make(map[gpu.Slice]int),
		},
		{
			name: "Node geometry already provides required profiles, should do nothing",
			nodeGPUs: []gpuSpec{
				{
					model: gpu.GPUModel_A30,
					index: 0,
					used:  map[ProfileName]int{},
					free: map[ProfileName]int{
						Profile1g6gb: 4,
					},
				},
				{
					model: gpu.GPUModel_A100_SXM4_40GB,
					index: 1,
					used:  map[ProfileName]int{},
					free: map[ProfileName]int{
						Profile1g6gb: 2,
					},
				},
			},
			migProfiles: map[gpu.Slice]int{
				Profile1g6gb: 1,
			},
			expectedUpdated: false,
			expectedGeometry: map[gpu.Slice]int{
				Profile1g6gb: 6,
			},
		},
		{
			name: "Multiple GPUs, all are full: should not change anything",
			nodeGPUs: []gpuSpec{
				{
					model: gpu.GPUModel_A30,
					index: 0,
					used: map[ProfileName]int{
						Profile4g24gb: 1,
					},
					free: map[ProfileName]int{},
				},
				{
					model: gpu.GPUModel_A100_SXM4_40GB,
					index: 1,
					used: map[ProfileName]int{
						Profile7g40gb: 1,
					},
					free: map[ProfileName]int{},
				},
			},
			migProfiles: map[gpu.Slice]int{
				Profile1g5gb:  4,
				Profile2g10gb: 1,
			},
			expectedUpdated: false,
			expectedGeometry: map[gpu.Slice]int{
				Profile4g24gb: 1,
				Profile7g40gb: 1,
			},
		},
		{
			name: "GPU with available capacity: should create a new profile without changing the existing free ones",
			nodeGPUs: []gpuSpec{
				{
					model: gpu.GPUModel_A30,
					index: 0,
					used: map[ProfileName]int{
						Profile1g6gb: 1,
					},
					free: map[ProfileName]int{}, // free is empty, but GPU has enough capacity for creating new profiles
				},
			},
			migProfiles: map[gpu.Slice]int{
				Profile1g6gb: 2,
			},
			expectedUpdated: true,
			expectedGeometry: map[gpu.Slice]int{
				Profile1g6gb: 4,
			},
		},
		{
			name: "GPU with free MIG device: should split it into smaller profiles for making up space",
			nodeGPUs: []gpuSpec{
				{
					model: gpu.GPUModel_A30,
					index: 0,
					used: map[ProfileName]int{
						Profile4g24gb: 1, // GPU is full
					},
					free: map[ProfileName]int{},
				},
				{
					model: gpu.GPUModel_A30,
					index: 1,
					used: map[ProfileName]int{
						Profile1g6gb: 2,
					},
					free: map[ProfileName]int{
						Profile2g12gb: 1,
					},
				},
			},
			migProfiles: map[gpu.Slice]int{
				Profile1g6gb: 1,
			},
			expectedUpdated: true,
			expectedGeometry: map[gpu.Slice]int{
				Profile4g24gb: 1,
				Profile1g6gb:  4,
			},
		},
		{
			name: "GPU with free small MIG devices: should delete them and create the required one",
			nodeGPUs: []gpuSpec{
				{
					model: gpu.GPUModel_A30,
					index: 0,
					used: map[ProfileName]int{
						Profile4g24gb: 1,
					},
					free: map[ProfileName]int{},
				},
				{
					model: gpu.GPUModel_A30,
					index: 1,
					used:  map[ProfileName]int{},
					free: map[ProfileName]int{
						Profile1g6gb: 4,
					},
				},
			},
			migProfiles: map[gpu.Slice]int{
				Profile4g24gb: 1,
			},
			expectedUpdated: true,
			expectedGeometry: map[gpu.Slice]int{
				Profile4g24gb: 2,
			},
		},
		{
			name: "Multiple GPUs, if the first one can accommodate the required profiles, all the others should remain untouched",
			nodeGPUs: []gpuSpec{
				{
					model: gpu.GPUModel_A30,
					index: 0,
					used:  map[ProfileName]int{},
					free:  map[ProfileName]int{},
				},
				{
					model: gpu.GPUModel_A30,
					index: 1,
					used:  map[ProfileName]int{},
					free:  map[ProfileName]int{},
				},
			},
			migProfiles: map[gpu.Slice]int{
				Profile1g6gb: 3,
			},
			expectedGeometry: map[gpu.Slice]int{
				Profile1g6gb: 4,
			},
			expectedUpdated: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Init node
			node := Node{Name: "test"}
			gpus := make([]GPU, 0)
			for _, spec := range tt.nodeGPUs {
				g, err := NewGPU(spec.model, spec.index, spec.used, spec.free)
				assert.NoError(t, err)
				gpus = append(gpus, g)
			}
			node.GPUs = gpus
			node.nodeInfo = *framework.NewNodeInfo()

			// Run test
			updated, err := node.UpdateGeometryFor(tt.migProfiles)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedUpdated, updated)
			assert.Equal(t, tt.expectedGeometry, node.Geometry())
		})
	}
}

func TestNode__HasFreeMigCapacity(t *testing.T) {
	testCases := []struct {
		name     string
		nodeGPUs []GPU
		expected bool
	}{
		{
			name:     "Node without GPUs",
			nodeGPUs: make([]GPU, 0),
			expected: false,
		},
		{
			name: "Node with GPU without any free or used device",
			nodeGPUs: []GPU{
				NewGpuOrPanic(gpu.GPUModel_A30, 0, make(map[ProfileName]int), make(map[ProfileName]int)),
			},
			expected: true,
		},
		{
			name: "Node with GPU with free MIG devices",
			nodeGPUs: []GPU{
				NewGpuOrPanic(
					gpu.GPUModel_A30,
					0,
					map[ProfileName]int{Profile1g6gb: 1},
					map[ProfileName]int{Profile1g6gb: 1},
				),
			},
			expected: true,
		},
		{
			name: "Node with just a single GPU with just used MIG device, but which MIG allowed geometries allow to create more MIG devices",
			nodeGPUs: []GPU{
				NewGpuOrPanic(
					gpu.GPUModel_A30,
					0,
					map[ProfileName]int{
						Profile1g6gb: 1,
					},
					make(map[ProfileName]int),
				),
			},
			expected: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			n := Node{Name: "test", GPUs: tt.nodeGPUs}
			res := n.HasFreeCapacity()
			assert.Equal(t, tt.expected, res)
		})
	}
}
