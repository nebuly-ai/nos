package gpu

import (
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"
)

func TestResourceCalculator_ComputeRequiredGPUMemoryGB(t *testing.T) {
	const nvidiaDeviceGPUMemoryGB = 8
	tests := []struct {
		name         string
		resourceList v1.ResourceList
		expected     int64
	}{
		{
			name: "Resource list does not contain GPU resources",
			resourceList: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(2000, resource.BinarySI),
			},
			expected: 0,
		},
		{
			name: "Resource list contains NVIDIA GPU resource",
			resourceList: v1.ResourceList{
				v1.ResourceCPU:             *resource.NewMilliQuantity(1000, resource.DecimalSI),
				v1.ResourceMemory:          *resource.NewQuantity(2000, resource.BinarySI),
				constant.ResourceNvidiaGPU: *resource.NewQuantity(2, resource.DecimalSI),
			},
			expected: nvidiaDeviceGPUMemoryGB * 2,
		},
		{
			name: "Resource list contains NVIDIA GPU resource, MIG and MIG-like resources. Only NVIDIA GPU + MIG are considered",
			resourceList: v1.ResourceList{
				constant.ResourceNvidiaGPU:                *resource.NewQuantity(2, resource.DecimalSI),
				v1.ResourceName("foo/1g32gb"):             *resource.NewQuantity(2, resource.DecimalSI),
				v1.ResourceName("nvidia.com/mig-2g.32gb"): *resource.NewQuantity(3, resource.DecimalSI),
			},
			expected: nvidiaDeviceGPUMemoryGB*2 + 32*3,
		},
	}

	resourceCalculator := ResourceCalculator{
		NvidiaGPUDeviceMemoryGB: nvidiaDeviceGPUMemoryGB,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := resourceCalculator.ComputeRequiredGPUMemoryGB(tt.resourceList)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
