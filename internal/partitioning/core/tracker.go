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

package core

import (
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
)

// SliceTracker is a utility struct for tracking the lacking GPU slices of a list of pods
type SliceTracker struct {
	requestedSlices     map[gpu.Slice]int
	lackingSlices       map[gpu.Slice]int
	lackingSlicesLookup map[string]map[gpu.Slice]int // Pod => lacking slices

	calculator SliceCalculator
}

func NewSliceTracker(snapshot Snapshot, calculator SliceCalculator, pods []v1.Pod) SliceTracker {
	requestedSlices := make(map[gpu.Slice]int)
	lackingSlices := make(map[gpu.Slice]int)
	podsLackingSlices := make(map[string]map[gpu.Slice]int)
	for _, pod := range pods {
		podKey := util.GetNamespacedName(&pod).String()
		if podsLackingSlices[podKey] == nil {
			podsLackingSlices[podKey] = make(map[gpu.Slice]int)
		}
		for slice, quantity := range snapshot.GetLackingSlices(pod) {
			lackingSlices[slice] += quantity
			podsLackingSlices[podKey][slice] += quantity
		}
		for slice, quantity := range calculator.GetRequestedSlices(pod) {
			requestedSlices[slice] += quantity
		}
	}
	return SliceTracker{
		requestedSlices:     requestedSlices,
		lackingSlices:       lackingSlices,
		lackingSlicesLookup: podsLackingSlices,
		calculator:          calculator,
	}
}

func (t SliceTracker) GetLackingSlices() map[gpu.Slice]int {
	return t.lackingSlices
}

func (t SliceTracker) GetRequestedSlices() map[gpu.Slice]int {
	return t.requestedSlices
}

func (t SliceTracker) Remove(pod v1.Pod) {
	// Update requested slices
	for slice, quantity := range t.calculator.GetRequestedSlices(pod) {
		t.requestedSlices[slice] -= quantity
		if t.requestedSlices[slice] <= 0 {
			delete(t.requestedSlices, slice)
		}
	}
	// Update lacking slices
	if lackingSlices, ok := t.lackingSlicesLookup[util.GetNamespacedName(&pod).String()]; ok {
		for slice, quantity := range lackingSlices {
			t.lackingSlices[slice] -= quantity
			lackingSlices[slice] -= quantity
			if lackingSlices[slice] <= 0 {
				delete(lackingSlices, slice)
			}
			if t.lackingSlices[slice] <= 0 {
				delete(t.lackingSlices, slice)
			}
		}
	}
}
