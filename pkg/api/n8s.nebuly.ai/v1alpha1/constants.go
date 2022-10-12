package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
)

// Resources
const (
	// ResourceGPUMemory is the name of the custom resource used by n8s for specifying GPU memory GigaBytes
	ResourceGPUMemory v1.ResourceName = "n8s.nebuly.ai/gpu-memory"
)

// Labels
const (
	// LabelCapacityInfo specifies the status of a Pod in regard to the ElasticQuota it belongs to
	LabelCapacityInfo = "n8s.nebuly.ai/capacity"
)
