package resource

import (
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	quota "k8s.io/apiserver/pkg/quota/v1"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	v1helper "k8s.io/kubernetes/pkg/apis/core/v1/helper"
	kubefeatures "k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type Calculator interface {
	ComputePodRequest(pod v1.Pod) v1.ResourceList
}

// FromFrameworkToList converts the input scheduler framework.Resource to a core v1.ResourceList
func FromFrameworkToList(r framework.Resource) v1.ResourceList {
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

// FromListToFramework converts the input core v1.ResourceList to a scheduler framework.Resource
func FromListToFramework(r v1.ResourceList) framework.Resource {
	return *framework.NewResource(r)
}

// Sum returns a new resource corresponding to the result of Max(0, r1 - r2).
// The returned resource contains the union of the scalar resources of r1 and r2.
func Sum(r1 framework.Resource, r2 framework.Resource) framework.Resource {
	var res = framework.Resource{}
	res.Memory = r1.Memory + r2.Memory
	res.MilliCPU = r1.MilliCPU + r2.MilliCPU
	res.AllowedPodNumber = r1.AllowedPodNumber + r2.AllowedPodNumber
	res.EphemeralStorage = r1.EphemeralStorage + r2.EphemeralStorage

	for _, r := range util.GetKeys(r1.ScalarResources, r2.ScalarResources) {
		sum := r1.ScalarResources[r] + r2.ScalarResources[r]
		res.SetScalar(r, sum)
	}

	return res
}

// SubtractNonNegative returns a new resource corresponding to the result of Max(0, r1 - r2).
// The returned resource contains the union of the scalar resources of r1 and r2.
func SubtractNonNegative(r1 framework.Resource, r2 framework.Resource) framework.Resource {
	res := Subtract(r1, r2)

	res.Memory = util.Max(0, res.Memory)
	res.MilliCPU = util.Max(0, res.MilliCPU)
	res.AllowedPodNumber = util.Max(0, res.AllowedPodNumber)
	res.EphemeralStorage = util.Max(0, res.EphemeralStorage)
	for r, v := range res.ScalarResources {
		res.SetScalar(r, util.Max(0, v))
	}

	return res
}

// Subtract returns a new resource corresponding to the result of r1 - r2.
// The returned resource contains the union of the scalar resources of r1 and r2.
func Subtract(r1 framework.Resource, r2 framework.Resource) framework.Resource {
	var res = framework.Resource{}
	res.Memory = r1.Memory - r2.Memory
	res.MilliCPU = r1.MilliCPU - r2.MilliCPU
	res.AllowedPodNumber = r1.AllowedPodNumber - r2.AllowedPodNumber
	res.EphemeralStorage = r1.EphemeralStorage - r2.EphemeralStorage
	for _, r := range util.GetKeys(r1.ScalarResources, r2.ScalarResources) {
		sub := r1.ScalarResources[r] - r2.ScalarResources[r]
		res.SetScalar(r, sub)
	}
	return res
}

func ComputePodRequest(pod v1.Pod) v1.ResourceList {
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
	return quota.Max(containersRes, initRes)
}
