package v1alpha1

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
)

// Annotations
const (
	AnnotationGPUSpecPrefix = "n8s.nebuly.ai/spec-gpu"
	AnnotationGPUSpecFormat = "n8s.nebuly.ai/spec-gpu-%d-%s"

	AnnotationGPUStatusPrefix     = "n8s.nebuly.ai/status-gpu"
	AnnotationUsedMigStatusFormat = "n8s.nebuly.ai/status-gpu-%d-%s-used"
	AnnotationFreeMigStatusFormat = "n8s.nebuly.ai/status-gpu-%d-%s-free"
)

type GPUSpecAnnotation struct {
	Name     string
	Quantity int
}

func NewGPUSpecAnnotation(key, value string) (GPUSpecAnnotation, error) {
	if !strings.HasPrefix(key, AnnotationGPUSpecPrefix) {
		err := fmt.Errorf("GPUSpecAnnotation prefix is %q, got %q", AnnotationGPUSpecFormat, key)
		return GPUSpecAnnotation{}, err
	}
	quantity, err := strconv.Atoi(value)
	if err != nil {
		return GPUSpecAnnotation{}, err
	}
	return GPUSpecAnnotation{Name: key, Quantity: quantity}, nil
}

func (a GPUSpecAnnotation) Value() string {
	return fmt.Sprintf("%d", a.Quantity)
}

// GetGPUIndexWithMigProfile returns the GPU index included in the annotation together with the
// respective MIG profile. Example:
//
// Annotation
//
//	"n8s.nebuly.ai/spec-gpu-0-1g.10gb"
//
// Result
//
//	"0-1g.10gb"
func (a GPUSpecAnnotation) GetGPUIndexWithMigProfile() string {
	result := strings.TrimPrefix(a.Name, AnnotationGPUSpecPrefix)
	return strings.TrimPrefix(result, "-")
}

type GPUStatusAnnotation struct {
	Name     string
	Quantity int
}

func NewGPUStatusAnnotation(key, value string) (GPUStatusAnnotation, error) {
	if !strings.HasPrefix(key, AnnotationGPUStatusPrefix) {
		err := fmt.Errorf("GPUStatusAnnotation prefix is %q, got %q", AnnotationGPUStatusPrefix, key)
		return GPUStatusAnnotation{}, err
	}
	quantity, err := strconv.Atoi(value)
	if err != nil {
		return GPUStatusAnnotation{}, err
	}
	return GPUStatusAnnotation{Name: key, Quantity: quantity}, nil
}

func (a GPUStatusAnnotation) Value() string {
	return fmt.Sprintf("%d", a.Quantity)
}

// GetGPUIndexWithMigProfile returns the GPU index included in the annotation together with the
// respective MIG profile. Example:
//
// Annotation
//
//	"n8s.nebuly.ai/status-gpu-0-1g.10gb-used"
//
// Result
//
//	"0-1g.10gb"
func (a GPUStatusAnnotation) GetGPUIndexWithMigProfile() string {
	result := strings.TrimPrefix(a.Name, AnnotationGPUStatusPrefix)
	result = strings.TrimSuffix(result, "-used")
	result = strings.TrimSuffix(result, "-free")
	result = strings.TrimPrefix(result, "-")
	return result
}

func GetGPUAnnotationsFromNode(node v1.Node) ([]GPUStatusAnnotation, []GPUSpecAnnotation) {
	statusAnnotations := make([]GPUStatusAnnotation, 0)
	specAnnotations := make([]GPUSpecAnnotation, 0)
	for k, v := range node.Annotations {
		if specAnnotation, err := NewGPUSpecAnnotation(k, v); err != nil {
			specAnnotations = append(specAnnotations, specAnnotation)
			continue
		}
		if statusAnnotation, err := NewGPUStatusAnnotation(k, v); err != nil {
			statusAnnotations = append(statusAnnotations, statusAnnotation)
			continue
		}
	}
	return statusAnnotations, specAnnotations
}
