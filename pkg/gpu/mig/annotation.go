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
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
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
	// AnnotationMigStatusFormat is the format of the annotation used to expose MIG devices of a GPU
	// Example:
	// 		"n8s.nebuly.ai/status-gpu-0-1g.10gb-<status>"
	AnnotationMigStatusFormat = fmt.Sprintf(
		"%s-%%d-%%s-%%s",
		v1alpha1.AnnotationGPUStatusPrefix,
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
	Profile  ProfileName
	Index    int
	Status   resource.Status
	Quantity int
}

func ParseGPUStatusAnnotation(key, value string) (GPUStatusAnnotation, error) {
	if !strings.HasPrefix(key, v1alpha1.AnnotationGPUStatusPrefix) {
		err := fmt.Errorf("GPUStatusAnnotation prefix is %q, got %q", v1alpha1.AnnotationGPUStatusPrefix, key)
		return GPUStatusAnnotation{}, err
	}
	parts := strings.Split(key, "-")
	if len(parts) != 5 {
		return GPUStatusAnnotation{}, fmt.Errorf("invalid GPUStatusAnnotation key %q", key)
	}
	quantity, err := strconv.Atoi(value)
	if err != nil {
		return GPUStatusAnnotation{}, err
	}
	index, err := strconv.Atoi(parts[2])
	if err != nil {
		return GPUStatusAnnotation{}, fmt.Errorf("invalid GPU index: %s", err)
	}
	status, err := resource.ParseStatus(parts[len(parts)-1])
	if err != nil {
		return GPUStatusAnnotation{}, fmt.Errorf("invalid GPU status: %s", err)
	}

	return GPUStatusAnnotation{
		Index:    index,
		Profile:  ProfileName(parts[3]),
		Status:   status,
		Quantity: quantity,
	}, nil
}

func (a GPUStatusAnnotation) GetValue() string {
	return fmt.Sprintf("%d", a.Quantity)
}

func (a GPUStatusAnnotation) GetGPUIndex() int {
	return a.Index
}

func (a GPUStatusAnnotation) String() string {
	return fmt.Sprintf(AnnotationMigStatusFormat, a.Index, a.Profile, a.Status)
}

// IsUsed returns true if the annotation refers to a used device
func (a GPUStatusAnnotation) IsUsed() bool {
	return a.Status == resource.StatusUsed
}

// IsFree returns true if the annotation refers to a free device
func (a GPUStatusAnnotation) IsFree() bool {
	return a.Status == resource.StatusFree
}

func (a GPUStatusAnnotation) GetMigProfileName() ProfileName {
	return a.Profile
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
	return fmt.Sprintf("%d-%s", a.GetGPUIndex(), a.GetMigProfileName())
}

func GetGPUAnnotationsFromNode(node v1.Node) (GPUStatusAnnotationList, GPUSpecAnnotationList) {
	statusAnnotations := make(GPUStatusAnnotationList, 0)
	specAnnotations := make(GPUSpecAnnotationList, 0)
	for k, v := range node.Annotations {
		if specAnnotation, err := NewGPUSpecAnnotationFromNodeAnnotation(k, v); err == nil {
			specAnnotations = append(specAnnotations, specAnnotation)
			continue
		}
		if statusAnnotation, err := ParseGPUStatusAnnotation(k, v); err == nil {
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

func ComputeStatusAnnotations(devices gpu.DeviceList) []GPUStatusAnnotation {
	res := make([]GPUStatusAnnotation, 0)
	for profile, d := range GroupByMigProfile(devices) {
		for status, groupedByStatus := range d.GroupByStatus() {
			res = append(res, GPUStatusAnnotation{
				Index:    profile.GpuIndex,
				Profile:  profile.Name,
				Status:   status,
				Quantity: len(groupedByStatus),
			})
		}
	}
	return res
}
