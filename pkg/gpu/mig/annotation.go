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

package mig

import (
	"github.com/google/go-cmp/cmp"
	"github.com/nebuly-ai/nos/pkg/gpu"
)

func SpecMatchesStatus(specAnnotations gpu.SpecAnnotationList, statusAnnotations gpu.StatusAnnotationList) bool {
	specMigProfilesWithQuantity := make(map[string]int)
	statusMigProfilesWithQuantity := make(map[string]int)
	for _, a := range specAnnotations {
		specMigProfilesWithQuantity[a.GetIndexWithProfile()] += a.Quantity
	}
	for _, a := range statusAnnotations {
		statusMigProfilesWithQuantity[a.GetIndexWithProfile()] += a.Quantity
	}

	return cmp.Equal(specMigProfilesWithQuantity, statusMigProfilesWithQuantity)
}

func GroupSpecAnnotationsByMigProfile(annotations gpu.SpecAnnotationList) map[Profile]gpu.SpecAnnotationList {
	result := make(map[Profile]gpu.SpecAnnotationList)
	for _, a := range annotations {
		key := Profile{
			GpuIndex: a.Index,
			Name:     ProfileName(a.ProfileName),
		}
		if result[key] == nil {
			result[key] = make(gpu.SpecAnnotationList, 0)
		}
		result[key] = append(result[key], a)
	}
	return result
}
