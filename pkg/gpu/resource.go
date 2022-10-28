package gpu

import (
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	v1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
)

type ResourceCalculator struct {
	NvidiaGPUDeviceMemoryGB int64
}

// ComputePodRequest returns a v1.ResourceList that covers the largest
// width in each resource dimension. Because init-containers run sequentially, we collect
// the max in each dimension iteratively. In contrast, we sum the resource vectors for
// regular containers since they run simultaneously.
//
// If Pod Overhead is specified and the feature gate is set, the resources defined for Overhead
// are added to the calculated Resource request sum
//
// Example:
//
// Pod:
//
//	InitContainers
//	  IC1:
//	    CPU: 2
//	    Memory: 1G
//	  IC2:
//	    CPU: 2
//	    Memory: 3G
//	Containers
//	  C1:
//	    CPU: 2
//	    Memory: 1G
//	  C2:
//	    CPU: 1
//	    Memory: 1G
//
// Result: CPU: 3, Memory: 3G
func (r ResourceCalculator) ComputePodRequest(pod v1.Pod) v1.ResourceList {
	res := resource.ComputePodRequest(pod)

	// add required GPU memory resource
	gpuMemory := r.ComputeRequiredGPUMemoryGB(res)
	res[v1alpha1.ResourceGPUMemory] = *k8sresource.NewQuantity(gpuMemory, k8sresource.DecimalSI)

	return res
}

func (r ResourceCalculator) ComputeRequiredGPUMemoryGB(resourceList v1.ResourceList) int64 {
	var totalRequiredGB int64

	for resourceName, quantity := range resourceList {
		if resourceName == constant.ResourceNvidiaGPU {
			totalRequiredGB += r.NvidiaGPUDeviceMemoryGB * quantity.Value()
			continue
		}
		if mig.IsNvidiaMigDevice(resourceName) {
			migMemory, _ := mig.ExtractMemoryGBFromMigFormat(resourceName)
			totalRequiredGB += migMemory * quantity.Value()
			continue
		}
	}

	return totalRequiredGB
}
