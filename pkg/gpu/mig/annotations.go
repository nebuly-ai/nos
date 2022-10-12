package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"
	"reflect"
)

func SpecMatchesStatus(specAnnotations []types.GPUSpecAnnotation, statusAnnotations []types.GPUStatusAnnotation) bool {
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

func SpecMatchesResources(specAnnotations []types.GPUSpecAnnotation, resources []types.MigDeviceResource) bool {
	getKey := func(migProfile string, gpuIndex int) string {
		return fmt.Sprintf("%d-%s", gpuIndex, migProfile)
	}

	resourceGpuIndexWithMigProfileQuantities := make(map[string]int)
	for _, r := range resources {
		migProfile, _ := ExtractMigProfile(r.ResourceName)
		key := getKey(migProfile, r.GpuIndex)
		resourceGpuIndexWithMigProfileQuantities[key]++
	}

	specGpuIndexWithMigProfileQuantities := make(map[string]int)
	for _, a := range specAnnotations {
		key := getKey(a.GetMigProfile(), a.GetGPUIndex())
		specGpuIndexWithMigProfileQuantities[key] += a.Quantity
	}

	return reflect.DeepEqual(resourceGpuIndexWithMigProfileQuantities, specGpuIndexWithMigProfileQuantities)
}

func ComputeStatusAnnotations(used []types.MigDeviceResource, free []types.MigDeviceResource) []types.GPUStatusAnnotation {
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
		key := fmt.Sprintf(v1alpha1.AnnotationUsedMigStatusFormat, u.GpuIndex, u.GetMigProfile())
		annotationToQuantity[key] = quantity
	}
	// Free annotations
	for _, u := range free {
		quantity, _ := freeMigToQuantity[u.FullResourceName()]
		key := fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, u.GpuIndex, u.GetMigProfile())
		annotationToQuantity[key] = quantity
	}

	res := make([]types.GPUStatusAnnotation, len(annotationToQuantity))
	for k, v := range annotationToQuantity {
		if a, err := types.NewGPUStatusAnnotation(k, fmt.Sprintf("%d", v)); err == nil {
			res = append(res, a)
		}
	}
	return res
}
