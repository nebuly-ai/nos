package types

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
)

type GPUSpecAnnotation struct {
	Name     string
	Quantity int
}

func NewGPUSpecAnnotation(key, value string) (GPUSpecAnnotation, error) {
	if !strings.HasPrefix(key, v1alpha1.AnnotationGPUSpecPrefix) {
		err := fmt.Errorf("GPUSpecAnnotation prefix is %q, got %q", v1alpha1.AnnotationGPUMigSpecFormat, key)
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
	result := strings.TrimPrefix(a.Name, v1alpha1.AnnotationGPUSpecPrefix)
	return strings.TrimPrefix(result, "-")
}

type GPUStatusAnnotation struct {
	Name     string
	Quantity int
}

func NewGPUStatusAnnotation(key, value string) (GPUStatusAnnotation, error) {
	if !strings.HasPrefix(key, v1alpha1.AnnotationGPUStatusPrefix) {
		err := fmt.Errorf("GPUStatusAnnotation prefix is %q, got %q", v1alpha1.AnnotationGPUStatusPrefix, key)
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
	result := strings.TrimPrefix(a.Name, v1alpha1.AnnotationGPUStatusPrefix)
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
