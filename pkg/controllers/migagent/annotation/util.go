package annotation

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	migtypes "github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"reflect"
)

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

func ComputeStatusAnnotations(used []migtypes.DeviceResource, free []migtypes.DeviceResource) []GPUStatusAnnotation {
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
