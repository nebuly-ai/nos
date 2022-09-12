package v1alpha1

import (
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"
)

func TestComputeRequiredGPUMemoryGB(t *testing.T) {
	const nvidiaDeviceGPUMemoryGB = 8
	tests := []struct {
		name          string
		pod           v1.Pod
		expected      int64
		errorExpected bool
	}{
		{
			name: "Pod does not require GPU resource",
			pod: factory.BuildPod("ns-1", "pd-1").
				WithContainer(
					factory.BuildContainer("c1", "image:laster").
						WithCPUMilliRequest(100).
						WithGPUMemoryRequest(2000).
						Get(),
				).
				WithInitContainer(
					factory.BuildContainer("c2", "image:laster").
						WithCPUMilliRequest(50).
						Get(),
				).
				Get(),
			expected:      0,
			errorExpected: false,
		},
		{
			name: "Pod requires NVIDIA GPU resource",
			pod: factory.BuildPod("ns-1", "pd-1").
				WithContainer(
					factory.BuildContainer("c1", "image:laster").
						WithCPUMilliRequest(100).
						WithResourceRequest(constant.ResourceNvidiaGPU, *resource.NewQuantity(2, resource.DecimalSI)).
						Get(),
				).
				WithContainer(
					factory.BuildContainer("c1", "image:laster").
						WithCPUMilliRequest(100).
						WithResourceRequest(constant.ResourceNvidiaGPU, *resource.NewQuantity(1, resource.DecimalSI)).
						Get(),
				).
				WithInitContainer(
					factory.BuildContainer("c2", "image:laster").
						WithResourceRequest(constant.ResourceNvidiaGPU, *resource.NewQuantity(2, resource.DecimalSI)).
						Get(),
				).
				Get(),
			expected:      nvidiaDeviceGPUMemoryGB * 3,
			errorExpected: false,
		},
		{
			name: "Pod requires NVIDIA GPU resource MIG and MIG-like resources, only NVIDIA GPU + MIG are considered",
			pod: factory.BuildPod("ns-1", "pd-1").
				WithContainer(
					factory.BuildContainer("c1", "image:laster").
						WithCPUMilliRequest(100).
						WithResourceRequest(constant.ResourceNvidiaGPU, *resource.NewQuantity(1, resource.DecimalSI)).
						WithResourceRequest("foo/1g32gb", *resource.NewQuantity(2, resource.DecimalSI)).
						Get(),
				).
				WithContainer(
					factory.BuildContainer("c1", "image:laster").
						WithCPUMilliRequest(100).
						WithResourceRequest("nvidia.com/mig-2g16gb", *resource.NewQuantity(2, resource.DecimalSI)).
						Get(),
				).
				WithInitContainer(
					factory.BuildContainer("c2", "image:laster").
						WithResourceRequest("foo2/1g2gb", *resource.NewQuantity(2, resource.DecimalSI)).
						WithResourceRequest("nvidia.com/mig-1g8gb", *resource.NewQuantity(2, resource.DecimalSI)).
						Get(),
				).
				Get(),
			expected:      nvidiaDeviceGPUMemoryGB + 16*2 + 8*2,
			errorExpected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := computeRequiredGPUMemoryGB(tt.pod, nvidiaDeviceGPUMemoryGB)
			if tt.errorExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			}
		})
	}
}
