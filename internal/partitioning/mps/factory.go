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
	"github.com/nebuly-ai/nos/internal/controllers/gpupartitioner"
	"github.com/nebuly-ai/nos/internal/partitioning/core"
	"github.com/nebuly-ai/nos/internal/partitioning/state"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func NewActuator(client client.Client, devicePluginCM types.NamespacedName, devicePluginDelay time.Duration) core.Actuator {
	return core.NewActuator(
		client,
		NewPartitioner(
			client,
			devicePluginCM,
			devicePluginDelay,
		),
	)
}

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
	devicePluginCM types.NamespacedName,
	devicePluginDelay time.Duration,
) gpupartitioner.Controller {

	return gpupartitioner.NewController(
		scheme,
		client,
		podBatcher,
		clusterState,
		gpu.PartitioningKindMps,
		NewPlanner(scheduler),
		NewActuator(client, devicePluginCM, devicePluginDelay),
		NewSnapshotTaker(),
	)
}
