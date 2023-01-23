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

package resource

import (
	"fmt"
	"github.com/nebuly-ai/nos/pkg/constant"
	"k8s.io/api/core/v1"
	"strings"
)

type Status string

func ParseStatus(status string) (Status, error) {
	if strings.ToLower(status) == "free" {
		return StatusFree, nil
	}
	if strings.ToLower(status) == "used" {
		return StatusUsed, nil
	}
	if strings.ToLower(status) == "unknown" {
		return StatusUnknown, nil
	}
	return "", fmt.Errorf("invalid status %s", status)
}

const (
	StatusUsed    Status = "used"
	StatusFree    Status = "free"
	StatusUnknown Status = "unknown"
)

type Device struct {
	// ResourceName is the name of the resource exposed to k8s
	// (e.g. nvidia.com/gpu, nvidia.com/mig-2g10gb, etc.)
	ResourceName v1.ResourceName
	// DeviceId is the actual ID of the underlying device
	// (e.g. ID of the GPU, ID of the MIG device, etc.)
	DeviceId string
	// Status represents the status of the k8s resource (e.g. free or used)
	Status Status
}

func (d Device) IsUsed() bool {
	return d.Status == StatusUsed
}

func (d Device) IsFree() bool {
	return d.Status == StatusFree
}

func (d Device) IsNvidiaResource() bool {
	return strings.HasPrefix(d.ResourceName.String(), constant.NvidiaResourcePrefix)
}
