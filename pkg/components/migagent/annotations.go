package migagent

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	v1 "k8s.io/api/core/v1"
	"reflect"
	"strconv"
	"strings"
)

func getStatusAnnotations(node *v1.Node) map[string]string {
	res := make(map[string]string)
	for k, v := range node.Annotations {
		if strings.HasPrefix(k, v1alpha1.AnnotationGPUStatusPrefix) {
			res[k] = v
		}
	}
	return res
}

func getSpecAnnotations(node *v1.Node) map[string]string {
	res := make(map[string]string)
	for k, v := range node.Annotations {
		if strings.HasPrefix(k, v1alpha1.AnnotationGPUSpecPrefix) {
			res[k] = v
		}
	}
	return res
}

func specMatchesStatusAnnotations(node *v1.Node) bool {
	specAnnotations := getSpecAnnotations(node)
	statusAnnotations := getStatusAnnotations(node)

	specMIGProfilesWithQuantity := make(map[string]int)
	statusMIGProfilesWithQuantity := make(map[string]int)
	for k, v := range specAnnotations {
		quantity, _ := strconv.Atoi(v)
		specMIGProfilesWithQuantity[v1alpha1.GPUSpecAnnotation(k).GetGPUIndexWithMIGProfile()] += quantity
	}
	for k, v := range statusAnnotations {
		quantity, _ := strconv.Atoi(v)
		statusMIGProfilesWithQuantity[v1alpha1.GPUStatusAnnotation(k).GetGPUIndexWithMIGProfile()] += quantity
	}

	return reflect.DeepEqual(specMIGProfilesWithQuantity, statusMIGProfilesWithQuantity)
}

func computeStatusAnnotations(used []mig.Device, free []mig.Device) map[string]string {
	res := make(map[string]string)

	// Compute used MIG devices quantities
	usedMigToQuantity := make(map[string]int)
	for _, u := range used {
		key := u.FullResourceName()
		currentCount, _ := usedMigToQuantity[key]
		currentCount++
		usedMigToQuantity[key] = currentCount
	}
	// Compute free MIG devices quantities
	freeMigToQuantity := make(map[string]int)
	for _, u := range free {
		key := u.FullResourceName()
		currentCount, _ := freeMigToQuantity[key]
		currentCount++
		freeMigToQuantity[key] = currentCount
	}

	// Used annotations
	for _, u := range used {
		quantity, _ := usedMigToQuantity[u.FullResourceName()]
		key := fmt.Sprintf(v1alpha1.AnnotationUsedMIGStatusFormat, u.GpuIndex, u.GetMIGProfileName())
		res[key] = fmt.Sprintf("%d", quantity)
	}
	// Free annotations
	for _, u := range free {
		quantity, _ := freeMigToQuantity[u.FullResourceName()]
		key := fmt.Sprintf(v1alpha1.AnnotationFreeMIGStatusFormat, u.GpuIndex, u.GetMIGProfileName())
		res[key] = fmt.Sprintf("%d", quantity)
	}

	return res
}
