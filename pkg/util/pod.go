package util

import (
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	quota "k8s.io/apiserver/pkg/quota/v1"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	kubefeatures "k8s.io/kubernetes/pkg/features"
)

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
//
// Copyright 2020 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
func ComputePodResourceRequest(pod v1.Pod) v1.ResourceList {
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
	gpuMemory := ComputeRequiredGPUMemoryGB(res, 16) // TODO: use memory of smallest GPU currently present instead of fixed value
	res[constant.ResourceGPUMemory] = *resource.NewQuantity(gpuMemory, resource.DecimalSI)

	return res
}

// IsPodOverQuota foo
func IsPodOverQuota(pod v1.Pod) bool {
	if val, ok := pod.Labels[constant.LabelCapacityInfo]; ok {
		return val == string(constant.CapacityInfoOverQuota)
	}
	return false
}
