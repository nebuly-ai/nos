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
	"context"
	core2 "github.com/nebuly-ai/nebulnetes/internal/partitioning/core"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Planner struct {
	client.Client
	// nvidiaDevicePluginConfigMapName is the name of the ConfigMap containing the NVIDIA device plugin configuration
	nvidiaDevicePluginConfigMapName string
	// nvidiaDevicePluginConfigMapNamespace is the namespace of the ConfigMap containing the NVIDIA device plugin configuration
	nvidiaDevicePluginConfigMapNamespace string
}

func (p *Planner) Plan(ctx context.Context, s core2.Snapshot, pendingPods []v1.Pod) (core2.PartitioningPlan, error) {
	//pendingPods = util.Filter(pendingPods, hasGpuMemoryLabel)

	// Fetch NVIDIA device plugin CM containing the time slicing config of all nodes
	var cm v1.ConfigMap
	cmKey := client.ObjectKey{Name: p.nvidiaDevicePluginConfigMapName, Namespace: p.nvidiaDevicePluginConfigMapNamespace}
	if err := p.Get(ctx, cmKey, &cm); err != nil {
		return core2.PartitioningPlan{}, nil
	}

	// Init time-slicing snapshot
	//_, err := NewSnapshot(s)
	//if err != nil {
	//	return core.PartitioningPlan{}, fmt.Errorf("failed to initialize time-slicing snapshot: %w", err)
	//}

	return core2.PartitioningPlan{}, nil
}

//func hasGpuMemoryLabel(pod v1.Pod) bool {
//	_, ok := pod.Labels[v1alpha1.LabelGpuMemory]
//	return ok
//}
