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

package ts

import (
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/core"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/timeslicing"
	v1 "k8s.io/api/core/v1"
)

var _ core.SliceFilter = sliceFilter{}

type sliceFilter struct {
}

func (s sliceFilter) ExtractSlices(resources map[v1.ResourceName]int64) map[gpu.Slice]int {
	var res = make(map[gpu.Slice]int)
	for r, q := range resources {
		if timeslicing.IsTimeSlicingResource(r) {
			profileName, _ := timeslicing.ExtractProfileName(r)
			res[profileName] += int(q)
		}
	}
	return res
}

func NewSliceFilter() core.SliceFilter {
	return sliceFilter{}
}
