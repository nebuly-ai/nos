/*
 * Copyright 2022 Nebuly.ai
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package constant

import (
	v1 "k8s.io/api/core/v1"
	"time"
)

type CapacityInfo string

const (
	CapacityInfoOverQuota CapacityInfo = "over-quota"
	CapacityInfoInQuota   CapacityInfo = "in-quota"
)

// Controller names
const (
	ElasticQuotaControllerName          = "eq-controller"
	CompositeElasticQuotaControllerName = "ceq-controller"
	ClusterStateNodeControllerName      = "clusterstate-node-controller"
	ClusterStatePodControllerName       = "clusterstate-pod-controller"
	MigPartitionerControllerName        = "mig-partitioner-controller"
	MpsPartitionerControllerName        = "mps-partitioner-controller"
)

// Error messages
const (
	// InternalErrorMsg is the error message shown in logs for internal errors
	InternalErrorMsg = "internal error"
)

// Common RegEx
const (
	// RegexNvidiaMigResource is a regex matching the name of the MIG devices exposed by the NVIDIA device plugin
	RegexNvidiaMigResource     = `nvidia\.com\/mig-\d+g\.\d+gb`
	RegexNvidiaMigProfile      = `\d+g\.\d+gb`
	RegexNvidiaMigFormatMemory = `\d+gb`
)

// Prefixes
const (
	// NvidiaMigResourcePrefix is the prefix of NVIDIA MIG resources
	NvidiaMigResourcePrefix = "nvidia.com/mig-"
	NvidiaResourcePrefix    = "nvidia.com/"
)

// Resource names
const (
	// ResourceNvidiaGPU is the name of the GPU resource exposed by the NVIDIA device plugin
	ResourceNvidiaGPU v1.ResourceName = "nvidia.com/gpu"
)

// Env variables
const (
	// EnvVarNodeName is the name of the env variable containing the name of the node
	EnvVarNodeName = "NODE_NAME"
)

// Labels
const (
	// LabelNvidiaProduct is the name of the label assigned by the NVIDIA GPU Operator that identifies
	// the model of the NVIDIA GPUs on a certain node
	LabelNvidiaProduct = "nvidia.com/gpu.product"
	// LabelNvidiaCount is the name of the label assigned by the NVIDIA GPU Operator that identifies
	// the number of NVIDIA GPUs on a certain node
	LabelNvidiaCount = "nvidia.com/gpu.count"
	// LabelNvidiaMemory is the name of the label assigned by the NVIDIA GPU Operator that identifies
	// the amount of memory of the GPUs of a node
	LabelNvidiaMemory = "nvidia.com/gpu.memory"
	// LabelNvidiaDevicePluginConfig is the label used by the NVIDIA k8s device plugin for determining the
	// which plugin config to apply choosing from the respective ConfigMap
	LabelNvidiaDevicePluginConfig = "nvidia.com/device-plugin.config"
)

// Defaults
const (
	// DefaultNvidiaGPUResourceMemory is the default memory value (in GigaByte) that is associated to
	// nvidia.com/gpu resources. The value represents the GPU memory requirement of a single resource.
	// This value is used when the controller and scheduler configurations do not specify any value for this
	// setting.
	DefaultNvidiaGPUResourceMemory = 16

	// DefaultPodResourcesTimeout is the default timeout used for the Pod resource lister
	DefaultPodResourcesTimeout = 10 * time.Second
	// DefaultPodResourcesMaxMsgSize is the default max message size used for the Pod resource lister
	DefaultPodResourcesMaxMsgSize = 1024 * 1024 * 16 // 16 Mb

	// DefaultDevicePluginCMName is the default name of the ConfigMap used by the NVIDIA device plugin
	DefaultDevicePluginCMName = "device-plugin-configs"
	// DefaultDevicePluginCMNamespace is the default namespace of the ConfigMap used by the NVIDIA device plugin
	DefaultDevicePluginCMNamespace = "gpu-operator"
)

const (
	PodPhaseKey    = "status.phase"
	PodNodeNameKey = "spec.nodeName"
)
