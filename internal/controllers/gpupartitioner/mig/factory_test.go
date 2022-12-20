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

package mig_test

import (
	mig_partitioner "github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/mig"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"testing"
)

func TestSnapshotTaker(t *testing.T) {
	testCases := []struct {
		name                  string
		snapshotNodes         []v1.Node
		expectedSnapshotNodes []string
		expectedErr           bool
	}{
		{
			name:                  "Empty snapshot",
			snapshotNodes:         []v1.Node{},
			expectedSnapshotNodes: []string{},
			expectedErr:           false,
		},
		{
			name: "MIG Snapshot should include only nodes with gpu-partitioning=MIG",
			snapshotNodes: []v1.Node{
				factory.BuildNode("node-1").Get(),
				factory.BuildNode("node-2").WithLabels(map[string]string{
					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
				}).Get(),
				factory.BuildNode("node-3").WithLabels(map[string]string{
					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindTimeSlicing.String(),
				}).Get(),
			},
			expectedSnapshotNodes: []string{"node-2"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Init cluster snapshot
			nodeInfos := make(map[string]framework.NodeInfo)
			for _, n := range tt.snapshotNodes {
				n := n
				ni := framework.NewNodeInfo()
				ni.SetNode(&n)
				nodeInfos[n.Name] = *ni
			}

			snapshotTaker := mig_partitioner.NewSnapshotTaker()
			clusterState := state.NewClusterState(nodeInfos)

			// Take snapshot
			snapshot, err := snapshotTaker.TakeSnapshot(&clusterState)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				snapshotNodeNames := make([]string, 0)
				for n := range snapshot.GetNodes() {
					snapshotNodeNames = append(snapshotNodeNames, n)
				}
				assert.Equal(t, tt.expectedSnapshotNodes, snapshotNodeNames)
			}
		})
	}
}
