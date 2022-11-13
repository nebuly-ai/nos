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
	"context"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	v1 "k8s.io/api/core/v1"
)

type Planner interface {
	Plan(ctx context.Context, snapshot state.ClusterSnapshot, pendingPods []v1.Pod) (state.PartitioningState, error)
}

type Actuator interface {
	Apply(ctx context.Context, snapshot state.ClusterSnapshot, partitioning state.PartitioningState) error
}
