package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"k8s.io/api/core/v1"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	resourceRegexp        = regexp.MustCompile(constant.RegexNvidiaMigResource)
	migDeviceMemoryRegexp = regexp.MustCompile(constant.RegexNvidiaMigFormatMemory)
	numberRegexp          = regexp.MustCompile("\\d+")
)

func IsNvidiaMigDevice(resourceName v1.ResourceName) bool {
	return resourceRegexp.MatchString(string(resourceName))
}

// ExtractMigProfile extracts the name of the MIG profile from the provided resource name, and returns an error
// if the resource name is not a valid NVIDIA MIG resource.
//
// Example:
//
//	nvidia.com/mig-1g.10gb => 1g.10gb
func ExtractMigProfile(migFormatResourceName v1.ResourceName) (string, error) {
	if isMigResource := resourceRegexp.MatchString(string(migFormatResourceName)); !isMigResource {
		return "", fmt.Errorf("invalid input string, required format is %s", resourceRegexp.String())
	}
	return strings.TrimPrefix(string(migFormatResourceName), "nvidia.com/mig-"), nil
}

func ExtractMemoryGBFromMigFormat(migFormatResourceName v1.ResourceName) (int64, error) {
	var err error
	var res int64

	if isMigResource := resourceRegexp.MatchString(string(migFormatResourceName)); !isMigResource {
		return res, fmt.Errorf("invalid input string, required format is %s", resourceRegexp.String())
	}

	matches := migDeviceMemoryRegexp.FindAllString(string(migFormatResourceName), -1)
	if len(matches) != 1 {
		return res, fmt.Errorf("invalid input string, expected 1 regexp match but found %d", len(matches))
	}
	if res, err = strconv.ParseInt(numberRegexp.FindString(matches[0]), 10, 64); err != nil {
		return res, err
	}

	return res, nil
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

	return reflect.DeepEqual(specMigProfilesWithQuantity, statusMigProfilesWithQuantity)
}

func ComputeStatusAnnotations(used []DeviceResource, free []DeviceResource) []GPUStatusAnnotation {
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
		quantity, _ := usedMigToQuantity[u.FullResourceName()]
		key := fmt.Sprintf(v1alpha1.AnnotationUsedMigStatusFormat, u.GpuIndex, u.GetMigProfileName())
		annotationToQuantity[key] = quantity
	}
	// Free annotations
	for _, u := range free {
		quantity, _ := freeMigToQuantity[u.FullResourceName()]
		key := fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, u.GpuIndex, u.GetMigProfileName())
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
