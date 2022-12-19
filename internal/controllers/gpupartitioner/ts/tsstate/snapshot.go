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

package tsstate

import (
	"fmt"
	deviceplugin "github.com/NVIDIA/k8s-device-plugin/api/config/v1"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/timeslicing"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type TimeSlicingClusterSnapshot struct {
	state.ClusterSnapshot
	data       *timeSlicingData
	forkedData *timeSlicingData
}

type timeSlicingData struct {
	tsNodes map[string]timeslicing.Node
}

func (d timeSlicingData) clone() *timeSlicingData {
	res := timeSlicingData{tsNodes: make(map[string]timeslicing.Node)}
	for k, v := range d.tsNodes {
		node := v.Clone()
		res.tsNodes[k] = node
	}
	return &res
}

func NewSnapshot(snapshot state.ClusterSnapshot, nvidiaDevicePluginCm v1.ConfigMap) (TimeSlicingClusterSnapshot, error) {
	// Extract nodes with MIG partitioning enabled
	nodes := make(map[string]framework.NodeInfo)
	for k, v := range snapshot.GetNodes() {
		if v.Node() == nil {
			continue
		}
		if !gpu.IsTimeSlicingPartitioningEnabled(*v.Node()) {
			continue
		}
		nodes[k] = v
	}
	filteredSnapshot := state.NewClusterSnapshot(nodes)

	// Extract time-slicing config of each node from CM
	nodesTimeSlicingConfig, err := getNodesTimeSlicingConfig(nvidiaDevicePluginCm)
	if err != nil {
		return TimeSlicingClusterSnapshot{}, err
	}

	// Extract time-slicing nodes
	var empty = TimeSlicingClusterSnapshot{}
	var tsNodes = make(map[string]timeslicing.Node)
	for nodeName, nodeInfo := range filteredSnapshot.GetNodes() {
		if nodeInfo.Node() == nil {
			return empty, fmt.Errorf("node %s is nil in cluster snapshot, this should never happen", nodeName)
		}
		// if device plugin CM does not contain time-slicing config then create a new one
		var timeSlicingConfig deviceplugin.TimeSlicing
		timeSlicingConfig, ok := nodesTimeSlicingConfig[nodeName]
		if !ok {
			timeSlicingConfig = deviceplugin.TimeSlicing{Resources: []deviceplugin.ReplicatedResource{}}
		}
		// init time-slicing node
		node, err := timeslicing.NewNode(*nodeInfo.Node(), timeSlicingConfig)
		if err != nil {
			return empty, err
		}
		tsNodes[nodeName] = node
	}

	return TimeSlicingClusterSnapshot{
		ClusterSnapshot: snapshot,
		data:            &timeSlicingData{tsNodes: tsNodes},
	}, nil
}

func getNodesTimeSlicingConfig(nvidiaDevicePluginCM v1.ConfigMap) (map[string]deviceplugin.TimeSlicing, error) {
	var result = make(map[string]deviceplugin.TimeSlicing)
	for node, nodeConfigYaml := range nvidiaDevicePluginCM.Data {
		nodeConfig := deviceplugin.Config{}
		if err := yaml.Unmarshal([]byte(nodeConfigYaml), &nodeConfig); err != nil {
			return result, err
		}
		result[node] = nodeConfig.Sharing.TimeSlicing
	}

	return result, nil
}

func (s *TimeSlicingClusterSnapshot) getData() *timeSlicingData {
	if s.forkedData != nil {
		return s.forkedData
	}
	return s.data
}

func (s *TimeSlicingClusterSnapshot) GetNodes() map[string]timeslicing.Node {
	return s.getData().tsNodes
}

func (s *TimeSlicingClusterSnapshot) Fork() error {
	if err := s.ClusterSnapshot.Fork(); err != nil {
		return err
	}
	s.forkedData = s.getData().clone()
	return nil
}

func (s *TimeSlicingClusterSnapshot) Commit() {
	s.ClusterSnapshot.Commit()
	if s.forkedData != nil {
		s.data = s.forkedData
		s.forkedData = nil
	}
}

//func (s *TimeSlicingClusterSnapshot) AddPod(nodeName string, pod v1.Pod) error {
//	if err := s.ClusterSnapshot.AddPod(nodeName, pod); err != nil {
//		return err
//	}
//	node, ok := s.getData().tsNodes[nodeName]
//	if !ok {
//		return fmt.Errorf("time-slicing node %s not found", nodeName)
//	}
//	if err := node.AddPod(pod); err != nil {
//		return err
//	}
//	s.getData().tsNodes[nodeName] = node
//	return nil
//}
