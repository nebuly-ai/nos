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
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/gpu"
	v1 "k8s.io/api/core/v1"
	"regexp"
	"strconv"
	"strings"
)

const (
	ProfileEmpty ProfileName = ""

	Profile1g6gb  ProfileName = "1g.6gb"
	Profile2g12gb ProfileName = "2g.12gb"
	Profile4g24gb ProfileName = "4g.24gb"

	Profile1g5gb  ProfileName = "1g.5gb"
	Profile2g10gb ProfileName = "2g.10gb"
	Profile3g20gb ProfileName = "3g.20gb"
	Profile4g20gb ProfileName = "4g.20gb"
	Profile7g40gb ProfileName = "7g.40gb"

	Profile1g10gb ProfileName = "1g.10gb"
	Profile2g20gb ProfileName = "2g.20gb"
	Profile3g40gb ProfileName = "3g.40gb"
	Profile4g40gb ProfileName = "4g.40gb"
	Profile7g79gb ProfileName = "7g.79gb"
)

var (
	migProfileRegex = regexp.MustCompile(constant.RegexNvidiaMigProfile)
	migGiRegex      = regexp.MustCompile(`\d+g`)
	migMemoryRegex  = regexp.MustCompile(`\d+gb`)
)

type ProfileName string

func (p ProfileName) isValid() bool {
	return migProfileRegex.MatchString(string(p))
}

func (p ProfileName) String() string {
	return string(p)
}

func (p ProfileName) AsResourceName() v1.ResourceName {
	resourceNameStr := fmt.Sprintf("%s%s", constant.NvidiaMigResourcePrefix, p)
	return v1.ResourceName(resourceNameStr)
}

func (p ProfileName) getMemorySlices() int {
	asString := migMemoryRegex.FindString(string(p))
	asString = strings.TrimSuffix(asString, "gb")
	asInt, _ := strconv.Atoi(asString)
	return asInt
}

func (p ProfileName) getGiSlices() int {
	asString := migGiRegex.FindString(string(p))
	asString = strings.TrimSuffix(asString, "g")
	asInt, _ := strconv.Atoi(asString)
	return asInt
}

func (p ProfileName) SmallerThan(other gpu.Slice) bool {
	otherMig, ok := other.(ProfileName)
	if !ok {
		return false
	}
	if p.getMemorySlices() < otherMig.getMemorySlices() {
		return true
	}
	if p.getGiSlices() < otherMig.getGiSlices() {
		return true
	}
	return false
}

type Profile struct {
	GpuIndex int
	Name     ProfileName
}

type ProfileList []Profile

func (p ProfileList) GroupByGPU() map[int]ProfileList {
	res := make(map[int]ProfileList)
	for _, profile := range p {
		if res[profile.GpuIndex] == nil {
			res[profile.GpuIndex] = make(ProfileList, 0)
		}
		res[profile.GpuIndex] = append(res[profile.GpuIndex], profile)
	}
	return res
}
