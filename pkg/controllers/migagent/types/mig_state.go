package types

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"reflect"
)

// MigState represents the current state in terms of MIG resources of each GPU (which index is stored as key
// in the map)
type MigState map[int]types.MigDeviceResourceList

func NewMigState(resources []types.MigDeviceResource) MigState {
	res := make(MigState)
	for _, r := range resources {
		if res[r.GpuIndex] == nil {
			res[r.GpuIndex] = make([]types.MigDeviceResource, 0)
		}
		res[r.GpuIndex] = append(res[r.GpuIndex], r)
	}
	return res
}

func (s MigState) Matches(specAnnotations []GPUSpecAnnotation) bool {
	getKey := func(migProfile string, gpuIndex int) string {
		return fmt.Sprintf("%d-%s", gpuIndex, migProfile)
	}

	specGpuIndexWithMigProfileQuantities := make(map[string]int)
	for _, a := range specAnnotations {
		key := getKey(a.GetMigProfileName(), a.GetGPUIndex())
		specGpuIndexWithMigProfileQuantities[key] += a.Quantity
	}

	stateGpuIndexWithMigProfileQuantities := make(map[string]int)
	groupedBy := s.Flatten().GroupBy(func(r types.MigDeviceResource) string {
		return getKey(r.GetMigProfileName(), r.GpuIndex)
	})
	for k, v := range groupedBy {
		stateGpuIndexWithMigProfileQuantities[k] = len(v)
	}

	return reflect.DeepEqual(specGpuIndexWithMigProfileQuantities, stateGpuIndexWithMigProfileQuantities)
}

func (s MigState) Flatten() types.MigDeviceResourceList {
	allResources := make(types.MigDeviceResourceList, 0)
	for _, r := range s {
		allResources = append(allResources, r...)
	}
	return allResources
}

func (s MigState) DeepCopy() MigState {
	return NewMigState(s.Flatten())
}

// WithoutMigProfiles returns the state obtained after removing all the resources matching the MIG profiles
// on the GPU index provided as inputs
func (s MigState) WithoutMigProfiles(gpuIndex int, migProfiles []string) MigState {
	res := s.DeepCopy()
	res[gpuIndex] = make([]types.MigDeviceResource, 0)
	for _, r := range s[gpuIndex] {
		if !util.InSlice(r.GetMigProfileName(), migProfiles) {
			res[gpuIndex] = append(res[gpuIndex], r)
		}
	}
	return res
}
