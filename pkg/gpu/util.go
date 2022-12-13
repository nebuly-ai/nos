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
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"k8s.io/api/core/v1"
	"math"
	"strconv"
)

// GetModel returns the model of the GPUs on the node.
// It is assumed that all the GPUs of the node are of the same model.
func GetModel(node v1.Node) (Model, bool) {
	if val, ok := node.Labels[constant.LabelNvidiaProduct]; ok {
		return Model(val), true
	}
	return "", false
}

// GetCount returns the number of GPUs on the node.
func GetCount(node v1.Node) (int, bool) {
	if val, ok := node.Labels[constant.LabelNvidiaCount]; ok {
		if valAsInt, err := strconv.Atoi(val); err == nil {
			return valAsInt, true
		}
	}
	return 0, false
}

// GetMemoryGB returns the amount of memory GB of the GPUs on the node.
func GetMemoryGB(node v1.Node) (int, bool) {
	memoryStr, ok := node.Labels[constant.LabelNvidiaMemory]
	if !ok {
		return 0, false
	}
	memoryBytes, err := strconv.Atoi(memoryStr)
	if err != nil {
		return 0, false
	}
	memoryGb := math.Ceil(float64(memoryBytes) / 1000)
	return int(memoryGb), true
}
