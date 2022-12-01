package migstate

import (
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"testing"
)

func TestNewClusterSnapshot(t *testing.T) {
	testCases := []struct {
		name                     string
		snapshotNodes            []v1.Node
		expectedMigSnapshotNodes []v1.Node
		expectedErr              bool
	}{
		{
			name:                     "Empty snapshot",
			snapshotNodes:            []v1.Node{},
			expectedMigSnapshotNodes: []v1.Node{},
			expectedErr:              false,
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
			expectedMigSnapshotNodes: []v1.Node{
				factory.BuildNode("node-2").WithLabels(map[string]string{
					v1alpha1.LabelGpuPartitioning: gpu.PartitioningKindMig.String(),
				}).Get(),
			},
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
			snapshot := state.NewClusterSnapshot(nodeInfos)

			// Init MIG cluster snapshot
			migSnapshot, err := NewClusterSnapshot(snapshot)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				snapshotNodes := make([]v1.Node, 0)
				for _, n := range migSnapshot.GetNodes() {
					snapshotNodes = append(snapshotNodes, *n.Node())
				}
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMigSnapshotNodes, snapshotNodes)
			}
		})
	}
}
