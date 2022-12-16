package gpu

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
)

func ParseSpecAnnotation[K ~string](key, value string, _ K) (SpecAnnotation[K], error) {
	if !strings.HasPrefix(key, v1alpha1.AnnotationGpuSpecPrefix) {
		err := fmt.Errorf(
			"expected spec annotation prefix is %q, but got %q",
			v1alpha1.AnnotationGpuSpecPrefix,
			key,
		)
		return SpecAnnotation[K]{}, err
	}
	parts := strings.Split(key, "-")
	if len(parts) != 4 {
		return SpecAnnotation[K]{}, fmt.Errorf("invalid spec annotation key %q", key)
	}
	quantity, err := strconv.Atoi(value)
	if err != nil {
		return SpecAnnotation[K]{}, err
	}
	index, err := strconv.Atoi(parts[2])
	if err != nil {
		return SpecAnnotation[K]{}, fmt.Errorf("invalid GPU index: %s", err)
	}
	return SpecAnnotation[K]{
		ProfileName: K(parts[len(parts)-1]),
		Quantity:    quantity,
		Index:       index,
	}, nil
}

func ParseStatusAnnotation[K ~string](key, value string, _ K) (StatusAnnotation[K], error) {
	if !strings.HasPrefix(key, v1alpha1.AnnotationGpuStatusPrefix) {
		err := fmt.Errorf("expected status prefix is %q, but got %q", v1alpha1.AnnotationGpuStatusPrefix, key)
		return StatusAnnotation[K]{}, err
	}
	parts := strings.Split(key, "-")
	if len(parts) != 5 {
		return StatusAnnotation[K]{}, fmt.Errorf("invalid status annotation key %q", key)
	}
	quantity, err := strconv.Atoi(value)
	if err != nil {
		return StatusAnnotation[K]{}, err
	}
	index, err := strconv.Atoi(parts[2])
	if err != nil {
		return StatusAnnotation[K]{}, fmt.Errorf("invalid GPU index: %s", err)
	}
	status, err := resource.ParseStatus(parts[len(parts)-1])
	if err != nil {
		return StatusAnnotation[K]{}, fmt.Errorf("invalid GPU status: %s", err)
	}

	return StatusAnnotation[K]{
		Index:       index,
		ProfileName: K(parts[3]),
		Status:      status,
		Quantity:    quantity,
	}, nil
}

func ParseNodeAnnotations[K ~string](node v1.Node, t K) (StatusAnnotationList[K], SpecAnnotationList[K]) {
	statusAnnotations := make(StatusAnnotationList[K], 0)
	specAnnotations := make(SpecAnnotationList[K], 0)
	for k, v := range node.Annotations {
		if specAnnotation, err := ParseSpecAnnotation(k, v, t); err == nil {
			specAnnotations = append(specAnnotations, specAnnotation)
			continue
		}
		if statusAnnotation, err := ParseStatusAnnotation(k, v, t); err == nil {
			statusAnnotations = append(statusAnnotations, statusAnnotation)
			continue
		}
	}
	return statusAnnotations, specAnnotations
}

type StatusAnnotation[K ~string] struct {
	ProfileName K
	Index       int
	Status      resource.Status
	Quantity    int
}

// IsUsed returns true if the annotation refers to a used device
func (a StatusAnnotation[K]) IsUsed() bool {
	return a.Status == resource.StatusUsed
}

// IsFree returns true if the annotation refers to a free device
func (a StatusAnnotation[K]) IsFree() bool {
	return a.Status == resource.StatusFree
}

func (a StatusAnnotation[K]) String() string {
	return fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, a.Index, a.ProfileName, a.Status)
}

func (a StatusAnnotation[K]) GetValue() string {
	return fmt.Sprintf("%d", a.Quantity)
}

// GetIndexWithProfile returns the GPU index included in the annotation together with the
// respective profile. Example:
//
// Annotation:
//
//	"n8s.nebuly.ai/status-gpu-0-1g.10gb-used"
//
// Result:
//
//	"0-1g.10gb"
func (a StatusAnnotation[K]) GetIndexWithProfile() string {
	return fmt.Sprintf("%d-%s", a.Index, a.ProfileName)
}

type SpecAnnotation[K ~string] struct {
	ProfileName K
	Index       int
	Quantity    int
}

func (a SpecAnnotation[K]) String() string {
	return fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, a.Index, a.ProfileName)
}

func (a SpecAnnotation[K]) GetValue() string {
	return fmt.Sprintf("%d", a.Quantity)
}

// GetIndexWithProfile returns the GPU index included in the annotation together with the
// respective profile. Example:
//
// Annotation:
//
//	"n8s.nebuly.ai/status-gpu-0-1g.10gb-used"
//
// Result:
//
//	"0-1g.10gb"
func (a SpecAnnotation[K]) GetIndexWithProfile() string {
	return fmt.Sprintf("%d-%s", a.Index, a.ProfileName)
}

type SpecAnnotationList[K ~string] []SpecAnnotation[K]

func (l SpecAnnotationList[K]) GroupByGpuIndex() map[int]SpecAnnotationList[K] {
	result := make(map[int]SpecAnnotationList[K])
	for _, r := range l {
		key := r.Index
		if result[key] == nil {
			result[key] = make(SpecAnnotationList[K], 0)
		}
		result[key] = append(result[key], r)
	}
	return result
}

type StatusAnnotationList[K ~string] []StatusAnnotation[K]

func (l StatusAnnotationList[K]) GroupByGpuIndex() map[int]StatusAnnotationList[K] {
	result := make(map[int]StatusAnnotationList[K])
	for _, r := range l {
		key := r.Index
		if result[key] == nil {
			result[key] = make(StatusAnnotationList[K], 0)
		}
		result[key] = append(result[key], r)
	}
	return result
}

func (l StatusAnnotationList[K]) Filter(keep func(annotation StatusAnnotation[K]) bool) StatusAnnotationList[K] {
	result := make(StatusAnnotationList[K], 0)
	for _, a := range l {
		if keep(a) {
			result = append(result, a)
		}
	}
	return result
}

// GetUsed return a new GPUStatusAnnotationList containing the annotations referring to used devices
func (l StatusAnnotationList[K]) GetUsed() StatusAnnotationList[K] {
	return l.Filter(func(a StatusAnnotation[K]) bool {
		return a.IsUsed()
	})
}

// GetFree return a new GPUStatusAnnotationList containing the annotations referring to free devices
func (l StatusAnnotationList[K]) GetFree() StatusAnnotationList[K] {
	return l.Filter(func(a StatusAnnotation[K]) bool {
		return a.IsFree()
	})
}

func (l StatusAnnotationList[K]) Equal(other *StatusAnnotationList[K]) bool {
	return util.UnorderedEqual(l, *other)
}
