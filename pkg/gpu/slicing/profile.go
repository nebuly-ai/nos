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

package slicing

import (
	"fmt"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/gpu"
	v1 "k8s.io/api/core/v1"
	"regexp"
	"strconv"
	"strings"
)

var (
	profileNamePrefix = fmt.Sprintf("%s-", constant.ResourceNvidiaGPU.String())
	resourceRegexp    = regexp.MustCompile(`nvidia\.com/gpu-\d+gb`)
)

type ProfileName string

func (p ProfileName) SmallerThan(other gpu.Slice) bool {
	otherProfile, ok := other.(ProfileName)
	if !ok {
		return false
	}
	return p.GetMemorySizeGB() < otherProfile.GetMemorySizeGB()
}

func (p ProfileName) String() string {
	return string(p)
}

func NewProfile(sizeGb int) ProfileName {
	return ProfileName(fmt.Sprintf("%dgb", sizeGb))
}

func (p ProfileName) GetMemorySizeGB() int {
	trimmed := strings.TrimPrefix(p.String(), profileNamePrefix)
	trimmed = strings.TrimSuffix(trimmed, "gb")
	if i, err := strconv.Atoi(trimmed); err == nil {
		return i
	}
	return 0
}

func (p ProfileName) AsResourceName() v1.ResourceName {
	resourceNameStr := fmt.Sprintf("%s%s", profileNamePrefix, p)
	return v1.ResourceName(resourceNameStr)
}
