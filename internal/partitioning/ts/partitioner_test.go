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

package ts_test

import (
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/ts"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestToNvidiaSharing(t *testing.T) {
	t.Run("Empty node partitioning", func(t *testing.T) {
		nodePartitioning := state.NodePartitioning{GPUs: []state.GPUPartitioning{}}
		nvidiaSharing := ts.ToNvidiaSharing(nodePartitioning)
		assert.Empty(t, nvidiaSharing.TimeSlicing.Resources)
	})

	t.Run("Multiple GPUs, multiple resources with replicas", func(t *testing.T) {
		nodePartitioning := state.NodePartitioning{
			GPUs: []state.GPUPartitioning{
				{
					GPUIndex: 0,
					Resources: map[v1.ResourceName]int{
						"nvidia.com/gpu-10gb": 2,
						"nvidia.com/gpu-5gb":  2,
					},
				},
				{
					GPUIndex: 1,
					Resources: map[v1.ResourceName]int{
						"nvidia.com/gpu-1gb": 3,
						"nvidia.com/gpu-2gb": 2,
					},
				},
			},
		}
		nvidiaSharing := ts.ToNvidiaSharing(nodePartitioning)
		assert.Len(t, nvidiaSharing.TimeSlicing.Resources, 4)
	})
}
