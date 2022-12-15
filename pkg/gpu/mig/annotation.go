/*
 * Copyright 2022 Nebuly.ai
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

package mig

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
	"regexp"
	"strconv"
	"strings"
)

var (
	numberBeginningLineRegex = regexp.MustCompile(`\d+`)
)

var (
	AnnotationFreeMigStatusFormat = fmt.Sprintf(
		"%s-%%d-%%s-%s",
		v1alpha1.AnnotationGPUStatusPrefix,
		v1alpha1.AnnotationGPUStatusFreeSuffix,
	)
	AnnotationUsedMigStatusFormat = fmt.Sprintf(
		"%s-%%d-%%s-%s",
		v1alpha1.AnnotationGPUStatusPrefix,
		v1alpha1.AnnotationGPUStatusUsedSuffix,
	)
	AnnotationGPUMigSpecFormat = fmt.Sprintf("%s-%%d-%%s", v1alpha1.AnnotationGPUSpecPrefix)
)

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

func (l GPUSpecAnnotationList) GroupByMigProfile() map[Profile]GPUSpecAnnotationList {
	result := make(map[Profile]GPUSpecAnnotationList)
	for _, a := range l {
		key := Profile{
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

func NewGPUSpecAnnotationFromNodeAnnotation(key, value string) (GPUSpecAnnotation, error) {
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

func NewGpuSpecAnnotation(gpuIndex int, profile ProfileName, quantity int) GPUSpecAnnotation {
	return GPUSpecAnnotation{
		Name:     fmt.Sprintf(AnnotationGPUMigSpecFormat, gpuIndex, profile),
		Quantity: quantity,
	}
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

func (a GPUSpecAnnotation) GetMigProfileName() ProfileName {
	return ProfileName(migProfileRegex.FindString(a.Name))
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

type GPUStatusAnnotationList []GPUStatusAnnotation

func (l GPUStatusAnnotationList) GroupByMigProfile() map[Profile]GPUStatusAnnotationList {
	result := make(map[Profile]GPUStatusAnnotationList)
	for _, a := range l {
		key := Profile{
			GpuIndex: a.GetGPUIndex(),
			Name:     a.GetMigProfileName(),
		}
		if result[key] == nil {
			result[key] = make(GPUStatusAnnotationList, 0)
		}
		result[key] = append(result[key], a)
	}
	return result
}

func (l GPUStatusAnnotationList) GroupByGpuIndex() map[int]GPUStatusAnnotationList {
	result := make(map[int]GPUStatusAnnotationList)
	for _, r := range l {
		key := r.GetGPUIndex()
		if result[key] == nil {
			result[key] = make(GPUStatusAnnotationList, 0)
		}
		result[key] = append(result[key], r)
	}
	return result
}

func (l GPUStatusAnnotationList) Filter(filteringFunc func(annotation GPUStatusAnnotation) bool) GPUStatusAnnotationList {
	result := make(GPUStatusAnnotationList, 0)
	for _, a := range l {
		if filteringFunc(a) {
			result = append(result, a)
		}
	}
	return result
}

// GetUsed return a new GPUStatusAnnotationList containing the annotations referring to used devices
func (l GPUStatusAnnotationList) GetUsed() GPUStatusAnnotationList {
	return l.Filter(func(a GPUStatusAnnotation) bool {
		return a.IsUsed()
	})
}

// GetFree return a new GPUStatusAnnotationList containing the annotations referring to free devices
func (l GPUStatusAnnotationList) GetFree() GPUStatusAnnotationList {
	return l.Filter(func(a GPUStatusAnnotation) bool {
		return a.IsFree()
	})
}

func (l GPUStatusAnnotationList) Equal(other *GPUStatusAnnotationList) bool {
	return util.UnorderedEqual(l, *other)
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

// IsUsed returns true if the annotation refers to a used device
func (a GPUStatusAnnotation) IsUsed() bool {
	return strings.HasSuffix(a.Name, v1alpha1.AnnotationGPUStatusUsedSuffix)
}

// IsFree returns true if the annotation refers to a free device
func (a GPUStatusAnnotation) IsFree() bool {
	return strings.HasSuffix(a.Name, v1alpha1.AnnotationGPUStatusFreeSuffix)
}

func (a GPUStatusAnnotation) GetMigProfileName() ProfileName {
	return ProfileName(migProfileRegex.FindString(a.Name))
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

func GetGPUAnnotationsFromNode(node v1.Node) (GPUStatusAnnotationList, GPUSpecAnnotationList) {
	statusAnnotations := make(GPUStatusAnnotationList, 0)
	specAnnotations := make(GPUSpecAnnotationList, 0)
	for k, v := range node.Annotations {
		if specAnnotation, err := NewGPUSpecAnnotationFromNodeAnnotation(k, v); err == nil {
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

func SpecMatchesStatus(specAnnotations []GPUSpecAnnotation, statusAnnotations []GPUStatusAnnotation) bool {
	specMigProfilesWithQuantity := make(map[string]int)
	statusMigProfilesWithQuantity := make(map[string]int)
	for _, a := range specAnnotations {
		specMigProfilesWithQuantity[a.GetGPUIndexWithMigProfile()] += a.Quantity
	}
	for _, a := range statusAnnotations {
		statusMigProfilesWithQuantity[a.GetGPUIndexWithMigProfile()] += a.Quantity
	}

	return cmp.Equal(specMigProfilesWithQuantity, statusMigProfilesWithQuantity)
}

func ComputeStatusAnnotations(used []gpu.Device, free []gpu.Device) []GPUStatusAnnotation {
	annotationToQuantity := make(map[string]int)

	// Compute used MIG devices quantities
	usedMigToQuantity := make(map[string]int)
	for _, u := range used {
		key := u.FullResourceName()
		usedMigToQuantity[key]++
	}
	// Compute free MIG devices quantities
	freeMigToQuantity := make(map[string]int)
	for _, u := range free {
		key := u.FullResourceName()
		freeMigToQuantity[key]++
	}

	// Used annotations
	for _, u := range used {
		quantity := usedMigToQuantity[u.FullResourceName()]
		key := fmt.Sprintf(AnnotationUsedMigStatusFormat, u.GpuIndex, GetMigProfileName(u))
		annotationToQuantity[key] = quantity
	}
	// Free annotations
	for _, u := range free {
		quantity := freeMigToQuantity[u.FullResourceName()]
		key := fmt.Sprintf(AnnotationFreeMigStatusFormat, u.GpuIndex, GetMigProfileName(u))
		annotationToQuantity[key] = quantity
	}

	res := make([]GPUStatusAnnotation, 0)
	for k, v := range annotationToQuantity {
		if a, err := NewGPUStatusAnnotation(k, fmt.Sprintf("%d", v)); err == nil {
			res = append(res, a)
		}
	}
	return res
}
