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

package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"k8s.io/api/core/v1"
	"regexp"
	"strconv"
	"strings"
)

var (
	resourceRegexp        = regexp.MustCompile(constant.RegexNvidiaMigResource)
	migDeviceMemoryRegexp = regexp.MustCompile(constant.RegexNvidiaMigFormatMemory)
	numberRegexp          = regexp.MustCompile(`\d+`)
)

func IsNvidiaMigDevice(resourceName v1.ResourceName) bool {
	return resourceRegexp.MatchString(string(resourceName))
}

// ExtractMigProfile extracts the name of the MIG profile from the provided resource name, and returns an error
// if the resource name is not a valid NVIDIA MIG resource.
//
// Example:
//
//	nvidia.com/mig-1g.10gb => 1g.10gb
func ExtractMigProfile(migFormatResourceName v1.ResourceName) (ProfileName, error) {
	if isMigResource := resourceRegexp.MatchString(string(migFormatResourceName)); !isMigResource {
		return "", fmt.Errorf("invalid input string, required format is %s", resourceRegexp.String())
	}
	name := strings.TrimPrefix(string(migFormatResourceName), "nvidia.com/mig-")
	return ProfileName(name), nil
}

func ExtractMemoryGBFromMigFormat(migFormatResourceName v1.ResourceName) (int64, error) {
	var err error
	var res int64

	if isMigResource := resourceRegexp.MatchString(string(migFormatResourceName)); !isMigResource {
		return res, fmt.Errorf("invalid input string, required format is %s", resourceRegexp.String())
	}

	matches := migDeviceMemoryRegexp.FindAllString(string(migFormatResourceName), -1)
	if len(matches) != 1 {
		return res, fmt.Errorf("invalid input string, expected 1 regexp match but found %d", len(matches))
	}
	if res, err = strconv.ParseInt(numberRegexp.FindString(matches[0]), 10, 64); err != nil {
		return res, err
	}

	return res, nil
}

func GetRequestedMigResources(pod v1.Pod) map[ProfileName]int {
	res := make(map[ProfileName]int)
	for r, quantity := range resource.ComputePodRequest(pod) {
		if migProfile, err := ExtractMigProfile(r); err == nil {
			res[migProfile] += int(quantity.Value())
		}
	}
	return res
}

// GetMigProfileName returns the name of the Mig profile associated to the device
//
// Example:
//
//	Resource name: nvidia.com/mig-1g.10gb
//	GetMigProfileName() -> 1g.10gb
func GetMigProfileName(device gpu.Device) ProfileName {
	return ProfileName(strings.TrimPrefix(device.ResourceName.String(), constant.NvidiaMigResourcePrefix))
}

func GroupDevicesByMigProfile(l gpu.DeviceList) map[Profile]gpu.DeviceList {
	result := make(map[Profile]gpu.DeviceList)
	for _, r := range l {
		key := Profile{
			GpuIndex: r.GpuIndex,
			Name:     GetMigProfileName(r),
		}
		if result[key] == nil {
			result[key] = make(gpu.DeviceList, 0)
		}
		result[key] = append(result[key], r)
	}
	return result
}

func GroupSpecAnnotationsByMigProfile(annotations gpu.SpecAnnotationList[ProfileName]) map[Profile]gpu.SpecAnnotationList[ProfileName] {
	result := make(map[Profile]gpu.SpecAnnotationList[ProfileName])
	for _, a := range annotations {
		key := Profile{
			GpuIndex: a.Index,
			Name:     a.ProfileName,
		}
		if result[key] == nil {
			result[key] = make(gpu.SpecAnnotationList[ProfileName], 0)
		}
		result[key] = append(result[key], a)
	}
	return result
}
