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

package timeslicing

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
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
	//TODO implement me
	panic("implement me")
}

func (p ProfileName) String() string {
	return string(p)
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
