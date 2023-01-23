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

package mig

import (
	"fmt"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/resource"
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

// ExtractProfileName extracts the Name of the MIG profile from the provided resource Name, and returns an error
// if the resource Name is not a valid NVIDIA MIG resource.
//
// Example:
//
//	nvidia.com/mig-1g.10gb => 1g.10gb
func ExtractProfileName(resourceName v1.ResourceName) (ProfileName, error) {
	if isMigResource := resourceRegexp.MatchString(string(resourceName)); !isMigResource {
		return "", fmt.Errorf("invalid input string, required format is %s", resourceRegexp.String())
	}
	name := strings.TrimPrefix(string(resourceName), "nvidia.com/mig-")
	return ProfileName(name), nil
}

// ExtractProfileNameStr extracts the Name of the MIG profile from the provided resource Name, and returns an error
// if the resource Name is not a valid NVIDIA MIG resource.
//
// Example:
//
//	nvidia.com/mig-1g.10gb => 1g.10gb
func ExtractProfileNameStr(resourceName v1.ResourceName) (string, error) {
	profileName, err := ExtractProfileName(resourceName)
	if err != nil {
		return "", err
	}
	return profileName.String(), nil
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

func GetRequestedProfiles(pod v1.Pod) map[ProfileName]int {
	res := make(map[ProfileName]int)
	for r, quantity := range resource.ComputePodRequest(pod) {
		if migProfile, err := ExtractProfileName(r); err == nil {
			res[migProfile] += int(quantity.Value())
		}
	}
	return res
}

// GetMigProfileName returns the Name of the Mig profile associated to the device
//
// Example:
//
//	Resource Name: nvidia.com/mig-1g.10gb
//	GetMigProfileName() -> 1g.10gb
func GetMigProfileName(device gpu.Device) ProfileName {
	if profile, err := ExtractProfileName(device.ResourceName); err == nil {
		return profile
	}
	return ""
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

func AsResources(g gpu.Geometry) map[v1.ResourceName]int {
	res := make(map[v1.ResourceName]int)
	for p, v := range g {
		resourceName := v1.ResourceName(fmt.Sprintf("%s%s", constant.NvidiaMigResourcePrefix, p))
		res[resourceName] += v
	}
	return res
}
