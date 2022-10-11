package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	"strings"
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

// Annotations
const (
	AnnotationGPUSpecPrefix = "n8s.nebuly.ai/spec-gpu"
	AnnotationGPUSpecFormat = "n8s.nebuly.ai/spec-gpu-%d-%s"

	AnnotationGPUStatusPrefix     = "n8s.nebuly.ai/status-gpu"
	AnnotationUsedMIGStatusFormat = "n8s.nebuly.ai/status-gpu-%d-%s-used"
	AnnotationFreeMIGStatusFormat = "n8s.nebuly.ai/status-gpu-%d-%s-free"
)

type GPUSpecAnnotation string

func (a GPUSpecAnnotation) GetGPUIndexWithMIGProfile() string {
	result := strings.TrimPrefix(string(a), AnnotationGPUSpecPrefix)
	return strings.TrimPrefix(result, "-")
}

type GPUStatusAnnotation string

func (a GPUStatusAnnotation) GetGPUIndexWithMIGProfile() string {
	result := strings.TrimPrefix(string(a), AnnotationGPUStatusPrefix)
	result = strings.TrimSuffix(result, "-used")
	result = strings.TrimSuffix(result, "-free")
	result = strings.TrimPrefix(result, "-")
	return result
}
