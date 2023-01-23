/*
 * Copyright 2023 nebuly.com.
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
	"github.com/nebuly-ai/nos/pkg/resource"
	v1 "k8s.io/api/core/v1"
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

func (m Device) String() string {
	return fmt.Sprintf("%d/%s/%s/%s", m.GpuIndex, m.ResourceName, m.DeviceId, m.Status)
}

type DeviceList []Device

func (l DeviceList) GroupBy(keyFunc func(resource Device) string) map[string]DeviceList {
	result := make(map[string]DeviceList)
	for _, r := range l {
		key := keyFunc(r)
		if result[key] == nil {
			result[key] = make(DeviceList, 0)
		}
		result[key] = append(result[key], r)
	}
	return result
}

func (l DeviceList) SortByDeviceId() DeviceList {
	sorted := make(DeviceList, len(l))
	copy(sorted, l)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].DeviceId < sorted[j].DeviceId
	})
	return sorted
}

func (l DeviceList) GroupByGpuIndex() map[int]DeviceList {
	result := make(map[int]DeviceList)
	for _, r := range l {
		if result[r.GpuIndex] == nil {
			result[r.GpuIndex] = make(DeviceList, 0)
		}
		result[r.GpuIndex] = append(result[r.GpuIndex], r)
	}
	return result
}

func (l DeviceList) GetFree() DeviceList {
	result := make(DeviceList, 0)
	for _, r := range l {
		if r.IsFree() {
			result = append(result, r)
		}
	}
	return result
}

func (l DeviceList) GetUsed() DeviceList {
	result := make(DeviceList, 0)
	for _, r := range l {
		if r.IsUsed() {
			result = append(result, r)
		}
	}
	return result
}

func (l DeviceList) GroupByStatus() map[resource.Status]DeviceList {
	result := make(map[resource.Status]DeviceList)
	for _, r := range l {
		if result[r.Status] == nil {
			result[r.Status] = make(DeviceList, 0)
		}
		result[r.Status] = append(result[r.Status], r)
	}
	return result
}

func (l DeviceList) GroupByResourceName() map[v1.ResourceName]DeviceList {
	result := make(map[v1.ResourceName]DeviceList)
	for _, r := range l {
		if result[r.ResourceName] == nil {
			result[r.ResourceName] = make(DeviceList, 0)
		}
		result[r.ResourceName] = append(result[r.ResourceName], r)
	}
	return result
}

type extractProfileName func(name v1.ResourceName) (string, error)

func (l DeviceList) AsStatusAnnotation(getProfile extractProfileName) StatusAnnotationList {
	result := make(StatusAnnotationList, 0)
	for gpuIndex, devices := range l.GroupByGpuIndex() {
		for resourceName, devices := range devices.GroupByResourceName() {
			for status, devices := range devices.GroupByStatus() {
				if profileName, err := getProfile(resourceName); err == nil {
					result = append(result, StatusAnnotation{
						ProfileName: profileName,
						Status:      status,
						Index:       gpuIndex,
						Quantity:    len(devices),
					})
				}
			}
		}
	}
	return result
}
