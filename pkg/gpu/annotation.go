/*
 * Copyright 2023 nebuly.com.
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

package gpu

import (
	"fmt"
	"github.com/nebuly-ai/nos/pkg/api/nos.nebuly.com/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/resource"
	"github.com/nebuly-ai/nos/pkg/util"
	v1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
)

func ParseSpecAnnotation(key, value string) (SpecAnnotation, error) {
	if !strings.HasPrefix(key, v1alpha1.AnnotationGpuSpecPrefix) {
		err := fmt.Errorf(
			"expected spec annotation prefix is %q, but got %q",
			v1alpha1.AnnotationGpuSpecPrefix,
			key,
		)
		return SpecAnnotation{}, err
	}
	parts := strings.Split(key, "-")
	if len(parts) != 4 {
		return SpecAnnotation{}, fmt.Errorf("invalid spec annotation key %q", key)
	}
	quantity, err := strconv.Atoi(value)
	if err != nil {
		return SpecAnnotation{}, err
	}
	index, err := strconv.Atoi(parts[2])
	if err != nil {
		return SpecAnnotation{}, fmt.Errorf("invalid GPU index: %s", err)
	}
	return SpecAnnotation{
		ProfileName: parts[len(parts)-1],
		Quantity:    quantity,
		Index:       index,
	}, nil
}

func ParseStatusAnnotation(key, value string) (StatusAnnotation, error) {
	if !strings.HasPrefix(key, v1alpha1.AnnotationGpuStatusPrefix) {
		err := fmt.Errorf("expected status prefix is %q, but got %q", v1alpha1.AnnotationGpuStatusPrefix, key)
		return StatusAnnotation{}, err
	}
	parts := strings.Split(key, "-")
	if len(parts) != 5 {
		return StatusAnnotation{}, fmt.Errorf("invalid status annotation key %q", key)
	}
	quantity, err := strconv.Atoi(value)
	if err != nil {
		return StatusAnnotation{}, err
	}
	index, err := strconv.Atoi(parts[2])
	if err != nil {
		return StatusAnnotation{}, fmt.Errorf("invalid GPU index: %s", err)
	}
	status, err := resource.ParseStatus(parts[len(parts)-1])
	if err != nil {
		return StatusAnnotation{}, fmt.Errorf("invalid GPU status: %s", err)
	}

	return StatusAnnotation{
		Index:       index,
		ProfileName: parts[3],
		Status:      status,
		Quantity:    quantity,
	}, nil
}

func ParseNodeAnnotations(node v1.Node) (StatusAnnotationList, SpecAnnotationList) {
	statusAnnotations := make(StatusAnnotationList, 0)
	specAnnotations := make(SpecAnnotationList, 0)
	for k, v := range node.Annotations {
		if specAnnotation, err := ParseSpecAnnotation(k, v); err == nil {
			specAnnotations = append(specAnnotations, specAnnotation)
			continue
		}
		if statusAnnotation, err := ParseStatusAnnotation(k, v); err == nil {
			statusAnnotations = append(statusAnnotations, statusAnnotation)
			continue
		}
	}
	return statusAnnotations, specAnnotations
}

type StatusAnnotation struct {
	ProfileName string
	Index       int
	Status      resource.Status
	Quantity    int
}

// IsUsed returns true if the annotation refers to a used device
func (a StatusAnnotation) IsUsed() bool {
	return a.Status == resource.StatusUsed
}

// IsFree returns true if the annotation refers to a free device
func (a StatusAnnotation) IsFree() bool {
	return a.Status == resource.StatusFree
}

func (a StatusAnnotation) String() string {
	return fmt.Sprintf(v1alpha1.AnnotationGpuStatusFormat, a.Index, a.ProfileName, a.Status)
}

func (a StatusAnnotation) GetValue() string {
	return fmt.Sprintf("%d", a.Quantity)
}

// GetIndexWithProfile returns the GPU index included in the annotation together with the
// respective profile. Example:
//
// Annotation:
//
//	"nos.nebuly.com/status-gpu-0-1g.10gb-used"
//
// Result:
//
//	"0-1g.10gb"
func (a StatusAnnotation) GetIndexWithProfile() string {
	return fmt.Sprintf("%d-%s", a.Index, a.ProfileName)
}

type SpecAnnotation struct {
	ProfileName string
	Index       int
	Quantity    int
}

func (a SpecAnnotation) String() string {
	return fmt.Sprintf(v1alpha1.AnnotationGpuSpecFormat, a.Index, a.ProfileName)
}

func (a SpecAnnotation) GetValue() string {
	return fmt.Sprintf("%d", a.Quantity)
}

// GetIndexWithProfile returns the GPU index included in the annotation together with the
// respective profile. Example:
//
// Annotation:
//
//	"nos.nebuly.com/status-gpu-0-1g.10gb-used"
//
// Result:
//
//	"0-1g.10gb"
func (a SpecAnnotation) GetIndexWithProfile() string {
	return fmt.Sprintf("%d-%s", a.Index, a.ProfileName)
}

type SpecAnnotationList []SpecAnnotation

func (l SpecAnnotationList) GroupByGpuIndex() map[int]SpecAnnotationList {
	result := make(map[int]SpecAnnotationList)
	for _, r := range l {
		key := r.Index
		if result[key] == nil {
			result[key] = make(SpecAnnotationList, 0)
		}
		result[key] = append(result[key], r)
	}
	return result
}

type StatusAnnotationList []StatusAnnotation

func (l StatusAnnotationList) GroupByGpuIndex() map[int]StatusAnnotationList {
	result := make(map[int]StatusAnnotationList)
	for _, r := range l {
		key := r.Index
		if result[key] == nil {
			result[key] = make(StatusAnnotationList, 0)
		}
		result[key] = append(result[key], r)
	}
	return result
}

func (l StatusAnnotationList) Filter(keep func(annotation StatusAnnotation) bool) StatusAnnotationList {
	result := make(StatusAnnotationList, 0)
	for _, a := range l {
		if keep(a) {
			result = append(result, a)
		}
	}
	return result
}

// GetUsed return a new GPUStatusAnnotationList containing the annotations referring to used devices
func (l StatusAnnotationList) GetUsed() StatusAnnotationList {
	return l.Filter(func(a StatusAnnotation) bool {
		return a.IsUsed()
	})
}

// GetFree return a new GPUStatusAnnotationList containing the annotations referring to free devices
func (l StatusAnnotationList) GetFree() StatusAnnotationList {
	return l.Filter(func(a StatusAnnotation) bool {
		return a.IsFree()
	})
}

func (l StatusAnnotationList) Equal(other StatusAnnotationList) bool {
	return util.UnorderedEqual(l, other)
}
