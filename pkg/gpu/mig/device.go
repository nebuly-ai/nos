package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"sort"
	"strings"
)

type DeviceResource struct {
	resource.Device
	// GpuId is the Index of the parent GPU to which the MIG device belongs to
	GpuIndex int
}

// FullResourceName returns the full resource name of the MIG device, including
// the name of the resource corresponding to the MIG profile and the index
// of the GPU to which it belongs to.
func (m DeviceResource) FullResourceName() string {
	return fmt.Sprintf("%d/%s", m.GpuIndex, m.ResourceName)
}

// GetMigProfileName returns the name of the Mig profile associated to the device
//
// Example:
//
//	Resource name: nvidia.com/mig-1g.10gb
//	GetMigProfileName() -> 1g.10gb
func (m DeviceResource) GetMigProfileName() ProfileName {
	return ProfileName(strings.TrimPrefix(m.ResourceName.String(), constant.NvidiaMigResourcePrefix))
}

type DeviceResourceList []DeviceResource

func (l DeviceResourceList) GroupBy(keyFunc func(resource DeviceResource) string) map[string]DeviceResourceList {
	result := make(map[string]DeviceResourceList)
	for _, r := range l {
		key := keyFunc(r)
		if result[key] == nil {
			result[key] = make(DeviceResourceList, 0)
		}
		result[key] = append(result[key], r)
	}
	return result
}

func (l DeviceResourceList) SortByDeviceId() DeviceResourceList {
	sorted := make(DeviceResourceList, len(l))
	for i, r := range l {
		sorted[i] = r
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].DeviceId < sorted[j].DeviceId
	})
	return sorted
}

func (l DeviceResourceList) GroupByGpuIndex() map[int]DeviceResourceList {
	result := make(map[int]DeviceResourceList)
	for _, r := range l {
		if result[r.GpuIndex] == nil {
			result[r.GpuIndex] = make(DeviceResourceList, 0)
		}
		result[r.GpuIndex] = append(result[r.GpuIndex], r)
	}
	return result
}

func (l DeviceResourceList) GetFree() DeviceResourceList {
	result := make(DeviceResourceList, 0)
	for _, r := range l {
		if r.IsFree() {
			result = append(result, r)
		}
	}
	return result
}

func (l DeviceResourceList) GroupByMigProfile() map[Profile]DeviceResourceList {
	result := make(map[Profile]DeviceResourceList)
	for _, r := range l {
		key := Profile{
			GpuIndex: r.GpuIndex,
			Name:     r.GetMigProfileName(),
		}
		if result[key] == nil {
			result[key] = make(DeviceResourceList, 0)
		}
		result[key] = append(result[key], r)
	}
	return result
}
