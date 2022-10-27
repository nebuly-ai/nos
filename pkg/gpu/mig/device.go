package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
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
	return ProfileName(strings.TrimPrefix(m.ResourceName.String(), "nvidia.com/mig-"))
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
