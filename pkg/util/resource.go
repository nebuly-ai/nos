package util

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1helper "k8s.io/kubernetes/pkg/apis/core/v1/helper"
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

func ComputeRequiredGPUMemoryGB(resourceList v1.ResourceList, nvidiaGPUDeviceMemoryGB int64) int64 {
	var totalRequiredGB int64

	for resourceName, quantity := range resourceList {
		if resourceName == constant.ResourceNvidiaGPU {
			totalRequiredGB += nvidiaGPUDeviceMemoryGB * quantity.Value()
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
