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
)

// Labels
const (
	// LabelGPUMemory specifies the GPU Memory requirements of Pod, expressed in GigaByte
	LabelGPUMemory = "n8s.nebuly.ai/gpu-memory"
	// LabelCapacityInfo specifies the status of a Pod in regard to the ElasticQuota it belongs to
	LabelCapacityInfo = "n8s.nebuly.ai/capacity"
)

// Error messages
const (
	// InternalErrorMsg todo
	InternalErrorMsg = "internal error"
)

// Common RegEx
const (
	// RegexNvidiaMigDevice is a regex matching the name of the MIG devices exposed by the NVIDIA device plugin
	RegexNvidiaMigDevice       = `nvidia\.com\/mig-\d+g\d+gb`
	RegexNvidiaMigFormatMemory = `\d+gb`
)

// Resource names
const (
	// ResourceGPUMemory is the name of the custom resource used by n8s for specifying GPU memory GigaBytes
	ResourceGPUMemory v1.ResourceName = "nebuly.ai/gpu-memory"
	// ResourceNvidiaGPU is the name of the GPU resource exposed by the NVIDIA device plugin
	ResourceNvidiaGPU v1.ResourceName = "nvidia.com/gpu"
)

const (
	// DefaultNvidiaGPUResourceMemory is the default memory value (in GigaByte) that is associated to
	// nvidia.com/gpu resources. The value represents the GPU memory requirement of a single resource.
	// This value is used when the controller and scheduler configurations do not specify any value for this
	// setting.
	DefaultNvidiaGPUResourceMemory = 16
)
