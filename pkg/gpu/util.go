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
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/resource"
	"k8s.io/api/core/v1"
	"math"
	"strconv"
)

// GetModel returns the model of the GPUs on the node.
// It is assumed that all the GPUs of the node are of the same model.
func GetModel(node v1.Node) (Model, error) {
	val, ok := node.Labels[constant.LabelNvidiaProduct]
	if !ok {
		return "", fmt.Errorf(
			"cannot get GPU model from node %s labels: missing label %s",
			node.Name,
			constant.LabelNvidiaProduct,
		)
	}
	return Model(val), nil
}

// GetCount returns the number of GPUs on the node.
func GetCount(node v1.Node) (int, error) {
	val, ok := node.Labels[constant.LabelNvidiaCount]
	if !ok {
		return 0, fmt.Errorf(
			"cannot get GPU count from node labels, missing label %s",
			constant.LabelNvidiaCount,
		)
	}
	valAsInt, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return valAsInt, nil
}

// GetMemoryGB returns the amount of memory GB of the GPUs on the node.
func GetMemoryGB(node v1.Node) (int, error) {
	memoryStr, ok := node.Labels[constant.LabelNvidiaMemory]
	if !ok {
		return 0, fmt.Errorf(
			"cannot get GPU Memory GB from node labels, missing label %s",
			constant.LabelNvidiaMemory,
		)
	}
	memoryBytes, err := strconv.Atoi(memoryStr)
	if err != nil {
		return 0, err
	}
	memoryGb := math.Ceil(float64(memoryBytes) / 1000)
	return int(memoryGb), nil
}

func ComputeFreeDevicesAndUpdateStatus(used []Device, allocatable []Device) []Device {
	usedLookup := make(map[string]Device)
	for _, u := range used {
		usedLookup[u.DeviceId] = u
	}
	// Compute (allocatable - used)
	res := make([]Device, 0)
	for _, a := range allocatable {
		if _, isUsed := usedLookup[a.DeviceId]; !isUsed {
			a.Status = resource.StatusFree
			res = append(res, a)
		}
	}
	return res
}
