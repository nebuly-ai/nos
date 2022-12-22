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
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/core"
	"github.com/nebuly-ai/nebulnetes/internal/partitioning/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewPlanner(scheduler framework.Framework) core.Planner {
	return core.NewPlanner(
		NewPartitionCalculator(),
		NewSliceCalculator(),
		scheduler,
	)
}

func NewController(
	scheme *runtime.Scheme,
	client client.Client,
	podBatcher util.Batcher[v1.Pod],
	clusterState *state.ClusterState,
	scheduler framework.Framework,
) gpupartitioner.Controller {

	return gpupartitioner.NewController(
		scheme,
		client,
		podBatcher,
		clusterState,
		gpu.PartitioningKindTimeSlicing,
		NewPlanner(scheduler),
		NewActuator(client),
		NewSnapshotTaker(),
	)
}
