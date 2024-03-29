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

package mps

import (
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/slicing"
	v1 "k8s.io/api/core/v1"
)

var _ gpu.SliceCalculator = sliceCalculator{}

type sliceCalculator struct {
}

func (s sliceCalculator) GetRequestedSlices(pod v1.Pod) map[gpu.Slice]int {
	requestedProfiles := slicing.GetRequestedProfiles(pod)
	res := make(map[gpu.Slice]int, len(requestedProfiles))
	for p, q := range requestedProfiles {
		res[p] = q
	}
	return res
}

func NewSliceCalculator() gpu.SliceCalculator {
	return sliceCalculator{}
}
