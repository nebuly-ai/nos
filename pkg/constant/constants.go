package constant

const (
	ElasticQuotaControllerName = "elasticquota-controller"
)

type CapacityInfo string

const (
	CapacityInfoOverQuota CapacityInfo = "over-quota"
	CapacityInfoInQuota   CapacityInfo = "in-quota"
)

const (
	// LabelGPUMemory specifies the GPU Memory requirements of Pod, expressed in GigaByte
	LabelGPUMemory = "n8s.nebuly.ai/gpu-memory"
	// LabelCapacityInfo specifies the status of a Pod in regard to the ElasticQuota it belongs to
	LabelCapacityInfo = "n8s.nebuly.ai/capacity"
)
