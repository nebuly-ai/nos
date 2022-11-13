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

package migstate

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type MigClusterSnapshot struct {
	state.ClusterSnapshot
	data       *migData
	forkedData *migData
}

type migData struct {
	migNodes map[string]mig.Node
}

func (d migData) clone() *migData {
	res := migData{migNodes: make(map[string]mig.Node)}
	for k, v := range d.migNodes {
		node := v.Clone()
		res.migNodes[k] = node
	}
	return &res
}

func NewClusterSnapshot(snapshot state.ClusterSnapshot) (MigClusterSnapshot, error) {
	migNodes, err := asMigNodes(snapshot.GetNodes())
	if err != nil {
		return MigClusterSnapshot{}, err
	}
	return MigClusterSnapshot{
		ClusterSnapshot: snapshot,
		data:            &migData{migNodes: migNodes},
	}, nil
}

func asMigNodes(nodes map[string]framework.NodeInfo) (map[string]mig.Node, error) {
	res := make(map[string]mig.Node)
	for _, v := range nodes {
		migNode, err := mig.NewNode(*v.Node())
		if err != nil {
			return res, err
		}
		res[migNode.Name] = migNode
	}
	return res, nil
}

func (s *MigClusterSnapshot) getData() *migData {
	if s.forkedData != nil {
		return s.forkedData
	}
	return s.data
}

// GetCandidateNodes returns the Nodes with free MIG devices or available MIG capacity
func (s *MigClusterSnapshot) GetCandidateNodes() []mig.Node {
	result := make([]mig.Node, 0)
	for _, n := range s.getData().migNodes {
		if n.HasFreeMigCapacity() {
			result = append(result, n)
		}
	}
	return result
}

func (s *MigClusterSnapshot) GetPartitioningState() state.PartitioningState {
	migNodes := make([]mig.Node, 0)
	for _, v := range s.GetNodes() {
		if node, err := mig.NewNode(*v.Node()); err == nil {
			migNodes = append(migNodes, node)
		}
	}
	return fromMigNodesToPartitioningState(migNodes)
}

// GetLackingMigProfile returns (if any) the MIG profile requested by the Pod but currently not
// available in the ClusterSnapshot.
//
// As described in "Supporting MIG GPUs in Kubernetes" document, it is assumed that
// Pods request only one MIG device per time and with quantity 1, according to the
// idea that users should ask for a larger, single instance as opposed to multiple
// smaller instances.
func (s *MigClusterSnapshot) GetLackingMigProfile(pod v1.Pod) (mig.ProfileName, bool) {
	for r := range s.GetLackingResources(pod).ScalarResources {
		if mig.IsNvidiaMigDevice(r) {
			profileName, _ := mig.ExtractMigProfile(r)
			return profileName, true
		}
	}
	return "", false
}

func (s *MigClusterSnapshot) SetNode(node mig.Node) error {
	nodeInfo, found := s.GetNode(node.Name)
	if !found {
		return fmt.Errorf("cannot set MIG node %q because corresponding NodeInfo does not exist", node.Name)
	}
	scalarResources := getUpdatedScalarResources(nodeInfo, node)
	nodeInfo.Allocatable.ScalarResources = scalarResources
	s.ClusterSnapshot.SetNode(nodeInfo)
	s.getData().migNodes[node.Name] = node
	return nil
}

func (s *MigClusterSnapshot) Fork() error {
	if err := s.ClusterSnapshot.Fork(); err != nil {
		return err
	}
	s.forkedData = s.getData().clone()
	return nil
}

func (s *MigClusterSnapshot) Commit() {
	s.ClusterSnapshot.Commit()
	if s.forkedData != nil {
		s.data = s.forkedData
		s.forkedData = nil
	}
}

func (s *MigClusterSnapshot) AddPod(nodeName string, pod v1.Pod) error {
	if err := s.ClusterSnapshot.AddPod(nodeName, pod); err != nil {
		return err
	}
	migNode, ok := s.getData().migNodes[nodeName]
	if !ok {
		return fmt.Errorf("MIG node %v not found in cluster snapshot", nodeName)
	}
	if err := migNode.AddPod(pod); err != nil {
		return err
	}
	s.getData().migNodes[nodeName] = migNode
	return nil
}

// getUpdatedScalarResources returns the scalar resources of the nodeInfo provided as argument updated for
// matching the MIG resources defied by the specified mig.Node
func getUpdatedScalarResources(nodeInfo framework.NodeInfo, node mig.Node) map[v1.ResourceName]int64 {
	res := make(map[v1.ResourceName]int64)

	// Set all non-MIG scalar resources
	for r, v := range nodeInfo.Allocatable.ScalarResources {
		if !mig.IsNvidiaMigDevice(r) {
			res[r] = v
		}
	}
	// Set MIG scalar resources
	for r, v := range node.GetGeometry().AsResources() {
		res[r] = int64(v)
	}

	return res
}

func fromMigNodesToPartitioningState(nodes []mig.Node) state.PartitioningState {
	res := make(map[string]state.NodePartitioning)
	for _, node := range nodes {
		res[node.Name] = FromMigNodeToNodePartitioning(node)
	}
	return res
}
