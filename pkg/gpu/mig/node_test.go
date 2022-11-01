package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
						constant.LabelNvidiaProduct: string(GPUModel_A100_SMX4_40GB),
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
						fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 0, Profile1g10gb): "1",
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
						fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 0, Profile1g10gb): "1",
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
						fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 0, Profile1g5gb):  "2",
						fmt.Sprintf(v1alpha1.AnnotationUsedMigStatusFormat, 0, Profile2g10gb): "3",
						fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 1, Profile3g20gb): "2",
					},
					Labels: map[string]string{
						constant.LabelNvidiaProduct: string(GPUModel_A30),
					},
				},
			},
			expectedNode: Node{
				Name: "test-node",
				GPUs: []GPU{
					{
						index:                0,
						model:                GPUModel_A30,
						allowedMigGeometries: gpuModelToAllowedMigGeometries[GPUModel_A30],
						usedMigDevices: map[ProfileName]int{
							Profile2g10gb: 3,
						},
						freeMigDevices: map[ProfileName]int{
							Profile1g5gb: 2,
						},
					},
					{
						index:                1,
						model:                GPUModel_A30,
						allowedMigGeometries: gpuModelToAllowedMigGeometries[GPUModel_A30],
						usedMigDevices:       map[ProfileName]int{},
						freeMigDevices: map[ProfileName]int{
							Profile3g20gb: 2,
						},
					},
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewNode(tt.node)
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
		expected Geometry
	}{
		{
			name:     "Empty node",
			node:     Node{},
			expected: Geometry{},
		},
		{
			name: "Geometry is the sum of all GPUs Geometry",
			node: Node{
				Name: "node-1",
				GPUs: []GPU{
					{
						index:                0,
						model:                GPUModel_A30,
						allowedMigGeometries: gpuModelToAllowedMigGeometries[GPUModel_A30],
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
						model:                GPUModel_A100_SMX4_40GB,
						allowedMigGeometries: gpuModelToAllowedMigGeometries[GPUModel_A100_SMX4_40GB],
						usedMigDevices: map[ProfileName]int{
							Profile1g5gb:  3,
							Profile2g10gb: 1,
						},
						freeMigDevices: map[ProfileName]int{
							Profile1g5gb:  1,
							Profile3g20gb: 2,
						},
					},
				},
			},
			expected: Geometry{
				Profile4g24gb: 2,
				Profile2g10gb: 1,
				Profile1g5gb:  8,
				Profile1g6gb:  1,
				Profile3g20gb: 2,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.node.GetGeometry())
		})
	}
}

func TestNode__UpdateGeometryFor(t *testing.T) {
	type gpuSpec struct {
		model GPUModel
		index int
		used  map[ProfileName]int
		free  map[ProfileName]int
	}

	testCases := []struct {
		name             string
		nodeGPUs         []gpuSpec
		migProfile       ProfileName
		expectedErr      bool
		expectedGeometry Geometry
	}{
		{
			name:             "Node without GPUs",
			nodeGPUs:         make([]gpuSpec, 0),
			migProfile:       Profile1g6gb,
			expectedErr:      true,
			expectedGeometry: make(Geometry),
		},
		{
			name: "Node geometry already provides required profiles, should do nothing",
			nodeGPUs: []gpuSpec{
				{
					model: GPUModel_A30,
					index: 0,
					used:  map[ProfileName]int{},
					free: map[ProfileName]int{
						Profile1g6gb: 4,
					},
				},
				{
					model: GPUModel_A100_SMX4_40GB,
					index: 1,
					used:  map[ProfileName]int{},
					free: map[ProfileName]int{
						Profile1g6gb: 2,
					},
				},
			},
			migProfile:  Profile1g6gb,
			expectedErr: false,
			expectedGeometry: Geometry{
				Profile1g6gb: 6,
			},
		},
		{
			name: "Multiple GPUs, all are full: should return error and geometry should not change",
			nodeGPUs: []gpuSpec{
				{
					model: GPUModel_A30,
					index: 0,
					used: map[ProfileName]int{
						Profile4g24gb: 1,
					},
					free: map[ProfileName]int{},
				},
				{
					model: GPUModel_A100_SMX4_40GB,
					index: 1,
					used: map[ProfileName]int{
						Profile7g40gb: 1,
					},
					free: map[ProfileName]int{},
				},
			},
			migProfile:  Profile1g5gb,
			expectedErr: true,
			expectedGeometry: Geometry{
				Profile4g24gb: 1,
				Profile7g40gb: 1,
			},
		},
		{
			name: "GPU with available capacity: should create a new profile without changing the existing free ones",
			nodeGPUs: []gpuSpec{
				{
					model: GPUModel_A30,
					index: 0,
					used: map[ProfileName]int{
						Profile1g6gb: 1,
					},
					free: map[ProfileName]int{}, // free is empty, but GPU has enough capacity for creating new profiles
				},
			},
			migProfile:  Profile1g6gb,
			expectedErr: false,
			expectedGeometry: Geometry{
				Profile1g6gb:  2,
				Profile2g12gb: 1,
			},
		},
		{
			name: "GPU with free MIG device: should split it into smaller profiles for making up space",
			nodeGPUs: []gpuSpec{
				{
					model: GPUModel_A30,
					index: 0,
					used: map[ProfileName]int{
						Profile4g24gb: 1, // GPU is full
					},
					free: map[ProfileName]int{},
				},
				{
					model: GPUModel_A30,
					index: 1,
					used: map[ProfileName]int{
						Profile1g6gb: 2,
					},
					free: map[ProfileName]int{
						Profile2g12gb: 1,
					},
				},
			},
			migProfile:  Profile1g6gb,
			expectedErr: false,
			expectedGeometry: Geometry{
				Profile4g24gb: 1,
				Profile1g6gb:  4,
			},
		},
		{
			name: "GPU with free small MIG devices: should delete them and create the required one",
			nodeGPUs: []gpuSpec{
				{
					model: GPUModel_A30,
					index: 0,
					used: map[ProfileName]int{
						Profile4g24gb: 1,
					},
					free: map[ProfileName]int{},
				},
				{
					model: GPUModel_A30,
					index: 1,
					used:  map[ProfileName]int{},
					free: map[ProfileName]int{
						Profile1g6gb: 4,
					},
				},
			},
			migProfile:  Profile4g24gb,
			expectedErr: false,
			expectedGeometry: Geometry{
				Profile4g24gb: 2,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Init node
			node := Node{Name: "test"}
			gpus := make([]GPU, 0)
			for _, spec := range tt.nodeGPUs {
				gpu, err := NewGPU(spec.model, spec.index, spec.used, spec.free)
				assert.NoError(t, err)
				gpus = append(gpus, gpu)
			}
			node.GPUs = gpus

			// Run test
			err := node.UpdateGeometryFor(tt.migProfile)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedGeometry, node.GetGeometry())
		})
	}
}
