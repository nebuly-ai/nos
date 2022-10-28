package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"testing"
)

func TestNewNode(t *testing.T) {
	testCases := []struct {
		name          string
		node          *v1.Node
		expectedNode  Node
		expectedError bool
	}{
		{
			name: "Node without GPU annotations",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Labels: map[string]string{
						constant.LabelNvidiaProduct: string(GPUModel_A100_SMX4_40GB),
					},
				},
			},
			expectedNode: Node{
				Name: "test-node",
				gpus: make([]GPU, 0),
			},
			expectedError: false,
		},
		{
			name: "Node without GPU model label",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Annotations: map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 0, profile1g10gb): "1",
					},
				},
			},
			expectedNode: Node{
				Name: "test-node",
				gpus: make([]GPU, 0),
			},
			expectedError: false,
		},
		{
			name: "Node with unknown GPU model",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Annotations: map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 0, profile1g10gb): "1",
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
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Annotations: map[string]string{
						fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 0, profile1g5gb):  "2",
						fmt.Sprintf(v1alpha1.AnnotationUsedMigStatusFormat, 0, profile2g10gb): "3",
						fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 1, profile3g20gb): "2",
					},
					Labels: map[string]string{
						constant.LabelNvidiaProduct: string(GPUModel_A30),
					},
				},
			},
			expectedNode: Node{
				Name: "test-node",
				gpus: []GPU{
					{
						index:                0,
						model:                GPUModel_A30,
						allowedMigGeometries: gpuModelToAllowedMigGeometries[GPUModel_A30],
						usedMigDevices: map[ProfileName]int{
							profile2g10gb: 3,
						},
						freeMigDevices: map[ProfileName]int{
							profile1g5gb: 2,
						},
					},
					{
						index:                1,
						model:                GPUModel_A30,
						allowedMigGeometries: gpuModelToAllowedMigGeometries[GPUModel_A30],
						usedMigDevices:       map[ProfileName]int{},
						freeMigDevices: map[ProfileName]int{
							profile3g20gb: 2,
						},
					},
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			nodeInfo := framework.NewNodeInfo()
			nodeInfo.SetNode(tt.node)
			node, err := NewNode(*nodeInfo)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.expectedNode, node)
				assert.NoError(t, err)
			}
		})
	}

	t.Run("Node is nil", func(t *testing.T) {
		ni := framework.NodeInfo{}
		_, err := NewNode(ni)
		assert.Error(t, err)
	})
}
