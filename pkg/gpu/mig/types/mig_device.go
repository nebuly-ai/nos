package types

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	"strings"
)

type MigDeviceResource struct {
	resource.Device
	// GpuId is the Index of the parent GPU to which the MIG device belongs to
	GpuIndex int
}

// FullResourceName returns the full resource name of the MIG device, including
// the name of the resource corresponding to the MIG profile and the index
// of the GPU to which it belongs to.
func (m MigDeviceResource) FullResourceName() string {
	return fmt.Sprintf("%d/%s", m.GpuIndex, m.ResourceName)
}

// GetMigProfileName returns the name of the Mig profile associated to the device
//
// Example:
//
//	Resource name: nvidia.com/mig-1g.10gb
//	GetMigProfileName() -> 1g.10gb
func (m MigDeviceResource) GetMigProfileName() string {
	return strings.TrimPrefix(m.ResourceName.String(), "nvidia.com/mig-")
}

type MigDeviceResourceList []MigDeviceResource

func (l MigDeviceResourceList) GroupBy(keyFunc func(resource MigDeviceResource) string) map[string]MigDeviceResourceList {
	result := make(map[string]MigDeviceResourceList)
	for _, r := range l {
		key := keyFunc(r)
		if result[key] == nil {
			result[key] = make(MigDeviceResourceList, 0)
		}
		result[key] = append(result[key], r)
	}
	return result
}

func (l MigDeviceResourceList) GroupByMigProfile() map[MigProfile]MigDeviceResourceList {
	result := make(map[MigProfile]MigDeviceResourceList)
	for _, r := range l {
		key := MigProfile{
			GpuIndex: r.GpuIndex,
			Name:     r.GetMigProfileName(),
		}
		if result[key] == nil {
			result[key] = make(MigDeviceResourceList, 0)
		}
		result[key] = append(result[key], r)
	}
	return result
}
