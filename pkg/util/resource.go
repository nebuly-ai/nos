package util

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	quota "k8s.io/apiserver/pkg/quota/v1"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	v1helper "k8s.io/kubernetes/pkg/apis/core/v1/helper"
	kubefeatures "k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"regexp"
	"strconv"
)

var migDeviceRegexp = regexp.MustCompile(constant.RegexNvidiaMigDevice)
var migDeviceMemoryRegexp = regexp.MustCompile(constant.RegexNvidiaMigFormatMemory)
var numberRegexp = regexp.MustCompile("\\d+")
var nonScalarResources = []v1.ResourceName{
	v1.ResourceCPU,
	v1.ResourceMemory,
	v1.ResourcePods,
	v1.ResourceEphemeralStorage,
}

// FromFrameworkResourceToResourceList
func FromFrameworkResourceToResourceList(r framework.Resource) v1.ResourceList {
	result := v1.ResourceList{
		v1.ResourceCPU:              *resource.NewMilliQuantity(r.MilliCPU, resource.DecimalSI),
		v1.ResourceMemory:           *resource.NewQuantity(r.Memory, resource.BinarySI),
		v1.ResourcePods:             *resource.NewQuantity(int64(r.AllowedPodNumber), resource.BinarySI),
		v1.ResourceEphemeralStorage: *resource.NewQuantity(r.EphemeralStorage, resource.BinarySI),
	}
	for rName, rQuant := range r.ScalarResources {
		if v1helper.IsHugePageResourceName(rName) {
			result[rName] = *resource.NewQuantity(rQuant, resource.BinarySI)
		} else {
			result[rName] = *resource.NewQuantity(rQuant, resource.DecimalSI)
		}
	}
	return result
}

func FromResourceListToFrameworkResource(r v1.ResourceList) framework.Resource {
	res := framework.Resource{
		MilliCPU:         r.Cpu().MilliValue(),
		Memory:           r.Memory().Value(),
		EphemeralStorage: r.StorageEphemeral().Value(),
		AllowedPodNumber: int(r.Pods().Value()),
		ScalarResources:  make(map[v1.ResourceName]int64),
	}
	for resourceName, quantity := range r {
		if IsScalarResource(resourceName) {
			res.ScalarResources[resourceName] = quantity.Value()
		}
	}
	return res
}

func IsScalarResource(name v1.ResourceName) bool {
	for _, r := range nonScalarResources {
		if r == name {
			return false
		}
	}
	return true
}

type ResourceCalculator struct {
	NvidiaGPUDeviceMemoryGB int64
}

// ComputePodResourceRequest returns a v1.ResourceList that covers the largest
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
func (r ResourceCalculator) ComputePodResourceRequest(pod v1.Pod) v1.ResourceList {
	containersRes := v1.ResourceList{}
	for _, container := range pod.Spec.Containers {
		containersRes = quota.Add(containersRes, container.Resources.Requests)
	}
	initRes := v1.ResourceList{}

	// take max_resource for init_containers
	for _, container := range pod.Spec.InitContainers {
		initRes = quota.Max(initRes, container.Resources.Requests)
	}

	// If Overhead is being utilized, add to the total requests for the pod
	if pod.Spec.Overhead != nil && utilfeature.DefaultFeatureGate.Enabled(kubefeatures.PodOverhead) {
		quota.Add(containersRes, pod.Spec.Overhead)
	}

	// take max_resource for init_containers and containers
	res := quota.Max(containersRes, initRes)

	// add required GPU memory resource
	gpuMemory := r.ComputeRequiredGPUMemoryGB(res)
	res[constant.ResourceGPUMemory] = *resource.NewQuantity(gpuMemory, resource.DecimalSI)

	return res
}

func (r ResourceCalculator) ComputeRequiredGPUMemoryGB(resourceList v1.ResourceList) int64 {
	var totalRequiredGB int64

	for resourceName, quantity := range resourceList {
		if resourceName == constant.ResourceNvidiaGPU {
			totalRequiredGB += r.NvidiaGPUDeviceMemoryGB * quantity.Value()
			continue
		}
		if IsNvidiaMigDevice(resourceName) {
			migMemory, _ := ExtractMemoryGBFromMigFormat(resourceName)
			totalRequiredGB += migMemory * quantity.Value()
			continue
		}
	}

	return totalRequiredGB
}

func IsNvidiaMigDevice(resourceName v1.ResourceName) bool {
	return migDeviceRegexp.MatchString(string(resourceName))
}

func ExtractMemoryGBFromMigFormat(migFormatResourceName v1.ResourceName) (int64, error) {
	var err error
	var res int64

	matches := migDeviceMemoryRegexp.FindAllString(string(migFormatResourceName), -1)
	if len(matches) != 1 {
		return res, fmt.Errorf("invalid input string, expected 1 regexp match but found %d", len(matches))
	}
	if res, err = strconv.ParseInt(numberRegexp.FindString(matches[0]), 10, 64); err != nil {
		return res, err
	}

	return res, nil
}
