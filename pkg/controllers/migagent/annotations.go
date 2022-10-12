package migagent

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"
	"reflect"
)

func specMatchesStatus(specAnnotations []v1alpha1.GPUSpecAnnotation, statusAnnotations []v1alpha1.GPUStatusAnnotation) bool {
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

func specMatchesResources(specAnnotations []v1alpha1.GPUSpecAnnotation, resources []types.MigDeviceResource) bool {
	resourceNameWithQuantity := make(map[string]int)
	for _, r := range resources {
		resourceNameWithQuantity[string(r.ResourceName)]++
	}
	return false
}

func computeStatusAnnotations(used []types.MigDeviceResource, free []types.MigDeviceResource) []v1alpha1.GPUStatusAnnotation {
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

	res := make([]v1alpha1.GPUStatusAnnotation, len(annotationToQuantity))
	for k, v := range annotationToQuantity {
		if a, err := v1alpha1.NewGPUStatusAnnotation(k, fmt.Sprintf("%d", v)); err == nil {
			res = append(res, a)
		}
	}
	return res
}
