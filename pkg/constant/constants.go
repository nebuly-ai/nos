package constant

import v1 "k8s.io/api/core/v1"

type CapacityInfo string

const (
	CapacityInfoOverQuota CapacityInfo = "over-quota"
	CapacityInfoInQuota   CapacityInfo = "in-quota"
)

// Controller names
const (
	ElasticQuotaControllerName          = "elasticquota-controller"
	CompositeElasticQuotaControllerName = "composite-elasticquota-controller"
	ClusterStateNodeControllerName      = "clusterstate-node-controller"
	ClusterStatePodControllerName       = "clusterstate-pod-controller"
	MigPartitionerControllerName        = "mig-partitioner-controller"
)

// Error messages
const (
	// InternalErrorMsg todo
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
)

// Resource names
const (
	// ResourceNvidiaGPU is the name of the GPU resource exposed by the NVIDIA device plugin
	ResourceNvidiaGPU v1.ResourceName = "nvidia.com/gpu"
)

// Labels
const (
	// LabelNvidiaProduct is the name of the label assigned by the NVIDIA GPU Operator that identities
	// the model of the NVIDIA GPUs on a certain node
	LabelNvidiaProduct = "nvidia.com/gpu.product"
)

const (
	// DefaultNvidiaGPUResourceMemory is the default memory value (in GigaByte) that is associated to
	// nvidia.com/gpu resources. The value represents the GPU memory requirement of a single resource.
	// This value is used when the controller and scheduler configurations do not specify any value for this
	// setting.
	DefaultNvidiaGPUResourceMemory = 16
)

const (
	PodPhaseKey    = "status.phase"
	PodNodeNameKey = "spec.nodeName"
)
