package types

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"
	v1 "k8s.io/api/core/v1"
	"regexp"
	"strconv"
	"strings"
)

var (
	numberBeginningLineRegex = regexp.MustCompile("^\\d+")
	migProfileRegex          = regexp.MustCompile(constant.RegexNvidiaMigProfile)
)

type GPUAnnotationList []GPUAnnotation

func (l GPUAnnotationList) ContainsMigProfile(migProfile string) bool {
	for _, a := range l {
		if a.GetMigProfileName() == migProfile {
			return true
		}
	}
	return false
}

type GPUAnnotation interface {
	GetValue() string
	GetGPUIndex() int
	GetMigProfileName() string
	GetGPUIndexWithMigProfile() string
}

type GPUSpecAnnotationList []GPUSpecAnnotation

func (l GPUSpecAnnotationList) GroupByGpuIndex() map[int]GPUSpecAnnotationList {
	result := make(map[int]GPUSpecAnnotationList)
	for _, r := range l {
		key := r.GetGPUIndex()
		if result[key] == nil {
			result[key] = make(GPUSpecAnnotationList, 0)
		}
		result[key] = append(result[key], r)
	}
	return result
}

func (l GPUSpecAnnotationList) GroupByMigProfile() map[types.MigProfile]GPUSpecAnnotationList {
	result := make(map[types.MigProfile]GPUSpecAnnotationList)
	for _, a := range l {
		key := types.MigProfile{
			GpuIndex: a.GetGPUIndex(),
			Name:     a.GetMigProfileName(),
		}
		if result[key] == nil {
			result[key] = make(GPUSpecAnnotationList, 0)
		}
		result[key] = append(result[key], a)
	}
	return result
}

type GPUSpecAnnotation struct {
	Name     string
	Quantity int
}

func NewGPUSpecAnnotation(key, value string) (GPUSpecAnnotation, error) {
	if !strings.HasPrefix(key, v1alpha1.AnnotationGPUSpecPrefix) {
		err := fmt.Errorf("GPUSpecAnnotation prefix is %q, got %q", v1alpha1.AnnotationGPUSpecPrefix, key)
		return GPUSpecAnnotation{}, err
	}
	quantity, err := strconv.Atoi(value)
	if err != nil {
		return GPUSpecAnnotation{}, err
	}
	return GPUSpecAnnotation{Name: key, Quantity: quantity}, nil
}

func (a GPUSpecAnnotation) GetValue() string {
	return fmt.Sprintf("%d", a.Quantity)
}

func (a GPUSpecAnnotation) GetGPUIndex() int {
	trimmed := strings.TrimPrefix(a.Name, v1alpha1.AnnotationGPUSpecPrefix+"-")
	indexStr := numberBeginningLineRegex.FindString(trimmed)
	index, _ := strconv.Atoi(indexStr)
	return index
}

func (a GPUSpecAnnotation) GetMigProfileName() string {
	return migProfileRegex.FindString(a.Name)
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
	return fmt.Sprintf("%d-%s", a.GetGPUIndex(), a.GetMigProfileName())
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

func (a GPUStatusAnnotation) GetValue() string {
	return fmt.Sprintf("%d", a.Quantity)
}

func (a GPUStatusAnnotation) GetGPUIndex() int {
	trimmed := strings.TrimPrefix(a.Name, v1alpha1.AnnotationGPUStatusPrefix+"-")
	indexStr := numberBeginningLineRegex.FindString(trimmed)
	index, _ := strconv.Atoi(indexStr)
	return index
}

func (a GPUStatusAnnotation) GetMigProfileName() string {
	return migProfileRegex.FindString(a.Name)
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
		if specAnnotation, err := NewGPUSpecAnnotation(k, v); err == nil {
			specAnnotations = append(specAnnotations, specAnnotation)
			continue
		}
		if statusAnnotation, err := NewGPUStatusAnnotation(k, v); err == nil {
			statusAnnotations = append(statusAnnotations, statusAnnotation)
			continue
		}
	}
	return statusAnnotations, specAnnotations
}
