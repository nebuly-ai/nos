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

package slicing

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	v1 "k8s.io/api/core/v1"
	"strings"
)

// ExtractProfileName extracts the name of the slicing profile from the provided resource name,
// and returns an error if the resource name is not a valid NVIDIA slicing resource.
//
// Example:
//
//	nvidia.com/10gb => 10gb
//	nvidia.com/gpu => error
func ExtractProfileName(resourceName v1.ResourceName) (ProfileName, error) {
	if isTsResource := resourceRegexp.MatchString(string(resourceName)); !isTsResource {
		return "", fmt.Errorf("invalid input string, required format is %s", resourceRegexp.String())
	}
	name := strings.TrimPrefix(string(resourceName), profileNamePrefix)
	return ProfileName(name), nil
}

func ExtractProfileNameStr(r v1.ResourceName) (string, error) {
	profileName, err := ExtractProfileName(r)
	if err != nil {
		return "", err
	}
	return profileName.String(), err
}

// ExtractGpuId returns the GPU ID corresponding to the resource ID provided as argument.
func ExtractGpuId(resourceId string) string {
	before, _, found := strings.Cut(resourceId, ReplicaGpuIdSeparator)
	if !found {
		return resourceId
	}
	return before
}

func GetRequestedProfiles(pod v1.Pod) map[ProfileName]int {
	res := make(map[ProfileName]int)
	for r, quantity := range resource.ComputePodRequest(pod) {
		if profile, err := ExtractProfileName(r); err == nil {
			res[profile] += int(quantity.Value())
		}
	}
	return res
}

func IsGpuSlice(r v1.ResourceName) bool {
	return resourceRegexp.MatchString(r.String())
}

func AsResources(g gpu.Geometry) map[v1.ResourceName]int {
	res := make(map[v1.ResourceName]int)
	for p, v := range g {
		resourceName := v1.ResourceName(fmt.Sprintf("%s%s", profileNamePrefix, p))
		res[resourceName] += v
	}
	return res
}
