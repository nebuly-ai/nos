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

package state

import (
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"testing"
)

func TestClusterState_GetNode(t *testing.T) {
	testCases := []struct {
		name     string
		nodes    map[string]framework.NodeInfo
		nodeName string

		expectedNode  framework.NodeInfo
		expectedFound bool
	}{
		{
			name: "Node does not exist",
			nodes: map[string]framework.NodeInfo{
				"node-1": {},
			},
			nodeName: "node-2",

			expectedNode:  framework.NodeInfo{},
			expectedFound: false,
		},
		{
			name: "Node exists",
			nodes: map[string]framework.NodeInfo{
				"node-1": {
					Generation: 1,
				},
				"node-2": {
					Generation: 2,
				},
			},
			nodeName: "node-2",

			expectedNode: framework.NodeInfo{
				Generation: 2,
			},
			expectedFound: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			clusterState := NewEmptyClusterState()
			clusterState.nodes = tt.nodes
			node, found := clusterState.GetNode(tt.nodeName)
			assert.Equal(t, tt.expectedNode, node)
			assert.Equal(t, tt.expectedFound, found)
		})
	}
}

func TestClusterState_deleteNode(t *testing.T) {
	testCases := []struct {
		name     string
		nodes    map[string]framework.NodeInfo
		bindings map[types.NamespacedName]string
		node     string

		expectedNodes    map[string]framework.NodeInfo
		expectedBindings map[types.NamespacedName]string
	}{
		{
			name: "Node does not exist",
			nodes: map[string]framework.NodeInfo{
				"node-1": {
					Generation: 1,
				},
			},
			bindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
			},
			node: "node-2",

			expectedNodes: map[string]framework.NodeInfo{
				"node-1": {
					Generation: 1,
				},
			},
			expectedBindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
			},
		},
		{
			name: "Node with multiple pods - lookup table should be cleaned",
			nodes: map[string]framework.NodeInfo{
				"node-1": {
					Generation: 1,
				},
				"node-2": {
					Generation: 2,
				},
			},
			bindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
				types.NamespacedName{Name: "pod-2", Namespace: "ns-1"}: "node-1",
				types.NamespacedName{Name: "pod-3", Namespace: "ns-1"}: "node-1",
				types.NamespacedName{Name: "pod-1", Namespace: "ns-2"}: "node-2",
			},
			node: "node-1",

			expectedNodes: map[string]framework.NodeInfo{
				"node-2": {
					Generation: 2,
				},
			},
			expectedBindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-2"}: "node-2",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			state := ClusterState{
				nodes:    tt.nodes,
				bindings: tt.bindings,
			}
			state.deleteNode(tt.node)
			assert.Equal(t, tt.expectedNodes, state.nodes)
			assert.Equal(t, tt.expectedBindings, state.bindings)
		})
	}
}

func TestClusterState_deletePod(t *testing.T) {
	testCases := []struct {
		name        string
		nodes       map[string]framework.NodeInfo
		bindings    map[types.NamespacedName]string
		podToDelete types.NamespacedName

		expectedNodes    map[string]framework.NodeInfo
		expectedBindings map[types.NamespacedName]string
		errorExpected    bool
	}{
		{
			name: "Pod does not exist",
			nodes: map[string]framework.NodeInfo{
				"node-1": {
					Generation: 1,
				},
			},
			bindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
			},
			podToDelete: types.NamespacedName{
				Namespace: "foo",
				Name:      "bar",
			},

			errorExpected: true,
			expectedNodes: map[string]framework.NodeInfo{
				"node-1": {
					Generation: 1,
				},
			},
			expectedBindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
			},
		},
		{
			name: "Pod's node does not exist",
			nodes: map[string]framework.NodeInfo{
				"node-1": {
					Generation: 1,
				},
			},
			bindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-2",
			},
			podToDelete: types.NamespacedName{
				Namespace: "ns-1",
				Name:      "pod-1",
			},

			errorExpected: false,
			expectedNodes: map[string]framework.NodeInfo{
				"node-1": {
					Generation: 1,
				},
			},
			expectedBindings: map[types.NamespacedName]string{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			state := ClusterState{
				nodes:    tt.nodes,
				bindings: tt.bindings,
			}
			err := state.deletePod(tt.podToDelete)
			if tt.errorExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedNodes, state.nodes)
			assert.Equal(t, tt.expectedBindings, state.bindings)
		})
	}

	t.Run("Pod gets removed, bindings and nodes get updated", func(t *testing.T) {
		// current state
		podOne := factory.BuildPod("ns-1", "pod-1").
			WithContainer(factory.BuildContainer("test", "test").WithCPUMilliRequest(100).Get()).
			WithPhase(v1.PodRunning).
			WithUID(util.RandomStringLowercase(10)).
			Get()
		podTwo := factory.BuildPod("ns-1", "pod-2").
			WithContainer(factory.BuildContainer("test", "test").WithCPUMilliRequest(200).Get()).
			WithPhase(v1.PodRunning).
			WithUID(util.RandomStringLowercase(10)).
			Get()
		nodeOne := factory.BuildNode("node-1").Get()
		nodeInfoOne := framework.NewNodeInfo(&podOne, &podTwo)
		nodeInfoOne.SetNode(&nodeOne)
		clusterState := ClusterState{
			nodes: map[string]framework.NodeInfo{
				"node-1": *nodeInfoOne,
			},
			bindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
				types.NamespacedName{Name: "pod-2", Namespace: "ns-1"}: "node-1",
			},
		}

		// expected
		expectedNodeInfo := framework.NewNodeInfo(&podTwo)
		expectedNodeInfo.SetNode(&nodeOne)
		expectedNodes := map[string]framework.NodeInfo{
			"node-1": *expectedNodeInfo,
		}
		expectedBindings := map[types.NamespacedName]string{
			types.NamespacedName{Name: "pod-2", Namespace: "ns-1"}: "node-1",
		}

		// delete pod and check
		err := clusterState.deletePod(util.GetNamespacedName(&podOne))
		assert.NoError(t, err)
		assert.Equal(t, expectedBindings, clusterState.bindings)
		assert.Equal(t, len(expectedNodes), len(clusterState.nodes))
		for k, n := range expectedNodes {
			actualNode, ok := clusterState.nodes[k]
			assert.True(t, ok)
			assert.Equal(t, n.Node(), actualNode.Node())
			assert.Equal(t, n.Allocatable, actualNode.Allocatable)
			assert.Equal(t, n.Pods, actualNode.Pods)
			assert.Equal(t, n.Requested, actualNode.Requested)
		}
	})
}

func TestClusterState_updateNode(t *testing.T) {
	t.Run("Add new node - usage is computed from only Running pods", func(t *testing.T) {
		// current state
		nodeOne := factory.BuildNode("node-1").Get()
		nodeInfoOne := framework.NewNodeInfo()
		nodeInfoOne.SetNode(&nodeOne)
		clusterState := ClusterState{
			nodes: map[string]framework.NodeInfo{
				"node-1": *nodeInfoOne,
			},
			bindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
			},
		}

		// new node and pods
		newNode := factory.BuildNode("node-2").Get()
		newPods := []v1.Pod{
			factory.BuildPod("ns-1", "pod-2").
				WithContainer(factory.BuildContainer("test", "test").WithCPUMilliRequest(100).Get()).
				WithPhase(v1.PodRunning).
				Get(),
			factory.BuildPod("ns-1", "pod-3").
				WithContainer(factory.BuildContainer("test", "test").WithCPUMilliRequest(200).Get()).
				Get(),
		}

		// expected
		expectedNewNodeInfo := framework.NewNodeInfo()
		expectedNewNodeInfo.SetNode(&newNode)
		expectedNewNodeInfo.AddPod(&newPods[0])
		expectedNodes := map[string]framework.NodeInfo{
			"node-1": *nodeInfoOne,
			"node-2": *expectedNewNodeInfo,
		}
		expectedBindings := map[types.NamespacedName]string{
			types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
			types.NamespacedName{Name: "pod-2", Namespace: "ns-1"}: "node-2",
			types.NamespacedName{Name: "pod-3", Namespace: "ns-1"}: "node-2", // bindings shall include even non-running pods
		}

		// Update and check
		clusterState.updateNode(newNode, newPods)
		assert.Equal(t, expectedBindings, clusterState.bindings)
		assert.Equal(t, len(expectedNodes), len(clusterState.nodes))
		for k, n := range expectedNodes {
			actualNode, ok := clusterState.nodes[k]
			assert.True(t, ok)
			assert.Equal(t, n.Node(), actualNode.Node())
			assert.Equal(t, n.Allocatable, actualNode.Allocatable)
			assert.Equal(t, n.Pods, actualNode.Pods)
			assert.Equal(t, n.Requested, actualNode.Requested)
		}
	})

	t.Run("Update existing node - bindings shall be updated too", func(t *testing.T) {
		// current state
		podOne := factory.BuildPod("ns-1", "pod-1").
			WithContainer(factory.BuildContainer("test", "test").WithCPUMilliRequest(100).Get()).
			WithPhase(v1.PodRunning).
			Get()
		podTwo := factory.BuildPod("ns-1", "pod-2").
			WithContainer(factory.BuildContainer("test", "test").WithCPUMilliRequest(200).Get()).
			Get()
		nodeOne := factory.BuildNode("node-1").Get()
		nodeInfoOne := framework.NewNodeInfo(&podOne, &podTwo)
		nodeInfoOne.SetNode(&nodeOne)
		clusterState := ClusterState{
			nodes: map[string]framework.NodeInfo{
				"node-1": *nodeInfoOne,
			},
			bindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
				types.NamespacedName{Name: "pod-2", Namespace: "ns-1"}: "node-1",
			},
		}

		// new pod
		newPod := factory.BuildPod("ns-1", "pod-3").
			WithContainer(factory.BuildContainer("test", "test").WithCPUMilliRequest(10).Get()).
			WithPhase(v1.PodRunning).
			Get()

		// expected
		expectedNodeInfo := framework.NewNodeInfo()
		expectedNodeInfo.SetNode(&nodeOne)
		expectedNodeInfo.AddPod(&newPod)
		expectedNodes := map[string]framework.NodeInfo{
			"node-1": *expectedNodeInfo,
		}
		expectedBindings := map[types.NamespacedName]string{
			types.NamespacedName{Name: "pod-3", Namespace: "ns-1"}: "node-1",
		}

		// Update and check
		clusterState.updateNode(nodeOne, []v1.Pod{newPod})
		assert.Equal(t, expectedBindings, clusterState.bindings)
		assert.Equal(t, len(expectedNodes), len(clusterState.nodes))
		for k, n := range expectedNodes {
			actualNode, ok := clusterState.nodes[k]
			assert.True(t, ok)
			assert.Equal(t, n.Node(), actualNode.Node())
			assert.Equal(t, n.Allocatable, actualNode.Allocatable)
			assert.Equal(t, n.Pods, actualNode.Pods)
			assert.Equal(t, n.Requested, actualNode.Requested)
		}
	})
}

func TestClusterState_updateUsage(t *testing.T) {
	testCases := []struct {
		name     string
		nodes    map[string]framework.NodeInfo
		bindings map[types.NamespacedName]string
		pod      v1.Pod

		expectedNodes    map[string]framework.NodeInfo
		expectedBindings map[types.NamespacedName]string
	}{
		{
			name: "Pod not assigned to any node - usage shall remain unchanged",
			nodes: map[string]framework.NodeInfo{
				"node-1": {
					Generation: 1,
				},
			},
			bindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
			},

			pod: factory.BuildPod("ns-1", "pod-1").Get(),
			expectedNodes: map[string]framework.NodeInfo{
				"node-1": {
					Generation: 1,
				},
			},
			expectedBindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
			},
		},
		{
			name: "Pod's node does not exist",
			nodes: map[string]framework.NodeInfo{
				"node-1": {
					Generation: 1,
				},
			},
			bindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-2",
			},

			pod: factory.BuildPod("ns-1", "pod-1").WithNodeName("node-3").Get(),
			expectedNodes: map[string]framework.NodeInfo{
				"node-1": {
					Generation: 1,
				},
			},
			expectedBindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-2",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			state := ClusterState{
				nodes:    tt.nodes,
				bindings: tt.bindings,
			}
			state.updateUsage(tt.pod)
			assert.Equal(t, tt.expectedNodes, state.nodes)
			assert.Equal(t, tt.expectedBindings, state.bindings)
		})
	}

	t.Run("Update usage, unknown pod binding", func(t *testing.T) {
		// current state
		nodeOne := factory.BuildNode("node-1").Get()
		nodeInfoOne := framework.NewNodeInfo()
		nodeInfoOne.SetNode(&nodeOne)
		clusterState := ClusterState{
			nodes: map[string]framework.NodeInfo{
				"node-1": *nodeInfoOne,
			},
			bindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
			},
		}

		// new pod
		newPod := factory.BuildPod("ns-1", "pod-2").
			WithContainer(factory.BuildContainer("test", "test").WithCPUMilliRequest(10).Get()).
			WithNodeName("node-1").
			WithPhase(v1.PodRunning).
			Get()

		// expected
		expectedNodeInfo := framework.NewNodeInfo()
		expectedNodeInfo.SetNode(&nodeOne)
		expectedNodeInfo.AddPod(&newPod)
		expectedNodes := map[string]framework.NodeInfo{
			"node-1": *expectedNodeInfo,
		}
		expectedBindings := map[types.NamespacedName]string{
			types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
			types.NamespacedName{Name: "pod-2", Namespace: "ns-1"}: "node-1",
		}

		// Update and check
		clusterState.updateUsage(newPod)
		assert.Equal(t, expectedBindings, clusterState.bindings)
		assert.Equal(t, len(expectedNodes), len(clusterState.nodes))
		for k, n := range expectedNodes {
			actualNode, ok := clusterState.nodes[k]
			assert.True(t, ok)
			assert.Equal(t, n.Node(), actualNode.Node())
			assert.Equal(t, n.Allocatable, actualNode.Allocatable)
			assert.Equal(t, n.Pods, actualNode.Pods)
			assert.Equal(t, n.Requested, actualNode.Requested)
		}
	})

	t.Run("update usage, known pod binding, pod changed node", func(t *testing.T) {
		// current state
		nodeOne := factory.BuildNode("node-1").Get()
		nodeTwo := factory.BuildNode("node-2").Get()
		nodeInfoOne := framework.NewNodeInfo()
		nodeInfoOne.SetNode(&nodeOne)
		nodeInfoTwo := framework.NewNodeInfo()
		nodeInfoTwo.SetNode(&nodeTwo)
		clusterState := ClusterState{
			nodes: map[string]framework.NodeInfo{
				"node-1": *nodeInfoOne,
				"node-2": *nodeInfoTwo,
			},
			bindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
				types.NamespacedName{Name: "pod-2", Namespace: "ns-1"}: "node-1",
			},
		}

		// updated pod
		updatedPod := factory.BuildPod("ns-1", "pod-1").
			WithContainer(factory.BuildContainer("test", "test").WithCPUMilliRequest(10).Get()).
			WithUID(util.RandomStringLowercase(10)).
			WithNodeName("node-2"). // node name is different from the one registered in state bindings
			WithPhase(v1.PodRunning).
			Get()

		// expected
		expectedNodeInfoOne := framework.NewNodeInfo()
		expectedNodeInfoOne.SetNode(&nodeOne)
		expectedNodeInfoTwo := framework.NewNodeInfo(&updatedPod)
		expectedNodeInfoTwo.SetNode(&nodeTwo)
		expectedNodes := map[string]framework.NodeInfo{
			"node-1": *expectedNodeInfoOne,
			"node-2": *expectedNodeInfoTwo,
		}
		expectedBindings := map[types.NamespacedName]string{
			types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-2",
			types.NamespacedName{Name: "pod-2", Namespace: "ns-1"}: "node-1",
		}

		// Update and check
		clusterState.updateUsage(updatedPod)
		assert.Equal(t, expectedBindings, clusterState.bindings)
		assert.Equal(t, len(expectedNodes), len(clusterState.nodes))
		for k, n := range expectedNodes {
			actualNode, ok := clusterState.nodes[k]
			assert.True(t, ok)
			assert.Equal(t, n.Node(), actualNode.Node())
			assert.Equal(t, n.Allocatable, actualNode.Allocatable)
			assert.Equal(t, n.Pods, actualNode.Pods)
			assert.Equal(t, n.Requested, actualNode.Requested)
		}
	})

	t.Run("update usage, known pod binding, pod on same node but status changed", func(t *testing.T) {
		// current state
		nodeOne := factory.BuildNode("node-1").Get()
		nodeTwo := factory.BuildNode("node-2").Get()
		nodeInfoOne := framework.NewNodeInfo()
		nodeInfoOne.SetNode(&nodeOne)
		nodeInfoTwo := framework.NewNodeInfo()
		nodeInfoTwo.SetNode(&nodeTwo)
		clusterState := ClusterState{
			nodes: map[string]framework.NodeInfo{
				"node-1": *nodeInfoOne,
				"node-2": *nodeInfoTwo,
			},
			bindings: map[types.NamespacedName]string{
				types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
				types.NamespacedName{Name: "pod-2", Namespace: "ns-1"}: "node-1",
			},
		}

		// updated pod
		updatedPod := factory.BuildPod("ns-1", "pod-1").
			WithContainer(factory.BuildContainer("test", "test").WithCPUMilliRequest(10).Get()).
			WithUID(util.RandomStringLowercase(10)).
			WithNodeName("node-1").
			WithPhase(v1.PodSucceeded).
			Get()

		// expected
		expectedNodeInfoOne := framework.NewNodeInfo()
		expectedNodeInfoOne.SetNode(&nodeOne)
		expectedNodeInfoTwo := framework.NewNodeInfo()
		expectedNodeInfoTwo.SetNode(&nodeTwo)
		expectedNodes := map[string]framework.NodeInfo{
			"node-1": *expectedNodeInfoOne,
			"node-2": *expectedNodeInfoTwo,
		}
		expectedBindings := map[types.NamespacedName]string{
			types.NamespacedName{Name: "pod-1", Namespace: "ns-1"}: "node-1",
			types.NamespacedName{Name: "pod-2", Namespace: "ns-1"}: "node-1",
		}

		// Update and check
		clusterState.updateUsage(updatedPod)
		assert.Equal(t, expectedBindings, clusterState.bindings)
		assert.Equal(t, len(expectedNodes), len(clusterState.nodes))
		for k, n := range expectedNodes {
			actualNode, ok := clusterState.nodes[k]
			assert.True(t, ok)
			assert.Equal(t, n.Node(), actualNode.Node())
			assert.Equal(t, n.Allocatable, actualNode.Allocatable)
			assert.Equal(t, n.Pods, actualNode.Pods)
			assert.Equal(t, n.Requested, actualNode.Requested)
		}
	})
}
