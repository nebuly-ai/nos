/*
 * Copyright 2023 Nebuly.ai.
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
	"github.com/nebuly-ai/nos/pkg/gpu"
	v1 "k8s.io/api/core/v1"
	"k8s.io/component-helpers/scheduling/corev1"
	"sort"
)

var _ Sorter = SorterAdapter(nil)

type SorterAdapter func(pods []v1.Pod) []v1.Pod

func (f SorterAdapter) Sort(pods []v1.Pod) []v1.Pod {
	return f(pods)
}

func NewPodSorter(sliceCalculator gpu.SliceCalculator) SorterAdapter {
	var sorter = func(pods []v1.Pod) []v1.Pod {
		sorted := make([]v1.Pod, len(pods))
		copy(sorted, pods)

		sort.SliceStable(sorted, func(i, j int) bool {
			// check priority first
			firstPodPriority := corev1.PodPriority(&sorted[i])
			secondPodPriority := corev1.PodPriority(&sorted[j])
			if firstPodPriority != secondPodPriority {
				return firstPodPriority > secondPodPriority
			}

			// if priority is equal, sort by requested MIG resources, placing first
			// the pods that require smaller MIG profiles in order to
			// maximize the number of pods that can be scheduled
			firstPodMigResources := sliceCalculator.GetRequestedSlices(sorted[i])
			if len(firstPodMigResources) == 0 {
				return false
			}
			secondPodMigResources := sliceCalculator.GetRequestedSlices(sorted[j])
			if len(secondPodMigResources) == 0 {
				return false
			}
			for firstPodProfile := range firstPodMigResources {
				for secondPodProfile := range secondPodMigResources {
					// we assume that a Pod requests at most one MIG profile
					return firstPodProfile.SmallerThan(secondPodProfile)
				}
			}

			return false
		})

		return sorted
	}
	return sorter
}
