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

package gpu

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"sort"
)

type Device struct {
	resource.Device
	GpuIndex int
}

// FullResourceName returns the full resource name of the MIG device, including
// the name of the resource corresponding to the MIG profile and the index
// of the GPU to which it belongs to.
func (m Device) FullResourceName() string {
	return fmt.Sprintf("%d/%s", m.GpuIndex, m.ResourceName)
}

type DeviceResourceList []Device

func (l DeviceResourceList) GroupBy(keyFunc func(resource Device) string) map[string]DeviceResourceList {
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
	copy(sorted, l)
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

func (l DeviceResourceList) GetUsed() DeviceResourceList {
	result := make(DeviceResourceList, 0)
	for _, r := range l {
		if r.IsUsed() {
			result = append(result, r)
		}
	}
	return result
}
