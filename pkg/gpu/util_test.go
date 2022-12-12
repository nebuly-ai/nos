package gpu_test

import (
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestGetMemoryGB(t *testing.T) {
	testCases := []struct {
		name        string
		node        v1.Node
		expectedVal int
		expectedOk  bool
	}{
		{
			name:        "no label",
			node:        factory.BuildNode("node-1").Get(),
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name: "label with invalid value",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaMemory: "invalid",
			}).Get(),
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name: "memory value gets rounded up to the nearest GB",
			node: factory.BuildNode("node-1").WithLabels(map[string]string{
				constant.LabelNvidiaMemory: "1100",
			}).Get(),
			expectedVal: 2,
			expectedOk:  true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			gb, ok := gpu.GetMemoryGB(tt.node)
			assert.Equal(t, tt.expectedVal, gb)
			assert.Equal(t, tt.expectedOk, ok)
		})
	}
}
