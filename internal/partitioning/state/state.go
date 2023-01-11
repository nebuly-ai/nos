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
	"fmt"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sync"
)

func NewEmptyClusterState() *ClusterState {
	state := ClusterState{
		nodes:             make(map[string]framework.NodeInfo),
		bindings:          make(map[types.NamespacedName]string),
		partitioningKinds: make(map[gpu.PartitioningKind]int),
	}
	state.refreshPartitioningKinds()
	return &state
}

func NewClusterState(nodes map[string]framework.NodeInfo) *ClusterState {
	state := ClusterState{
		nodes:             nodes,
		bindings:          make(map[types.NamespacedName]string),
		partitioningKinds: make(map[gpu.PartitioningKind]int),
	}
	state.refreshPartitioningKinds()
	return &state
}

type ClusterState struct {
	nodes             map[string]framework.NodeInfo
	bindings          map[types.NamespacedName]string // lookup table: Pod => NodeName
	partitioningKinds map[gpu.PartitioningKind]int    // lookup table: PartitioningKind => number of nodes with that kind of partitioning

	mtx sync.RWMutex
}

func (c *ClusterState) GetNode(nodeName string) (framework.NodeInfo, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	node, ok := c.nodes[nodeName]
	return node, ok
}

func (c *ClusterState) GetNodes() map[string]framework.NodeInfo {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return c.nodes
}

func (c *ClusterState) DeleteNode(name string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	delete(c.nodes, name)
	for key, nodeName := range c.bindings {
		if nodeName == name {
			delete(c.bindings, key)
		}
	}

	c.refreshPartitioningKinds()
}

func (c *ClusterState) UpdateNode(node v1.Node, pods []v1.Pod) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// Update nodes
	nodeInfo := framework.NewNodeInfo()
	nodeInfo.SetNode(&node)
	for _, p := range pods {
		p := p
		if p.Status.Phase == v1.PodRunning {
			nodeInfo.AddPod(&p)
		}
	}
	c.nodes[node.Name] = *nodeInfo

	// Update Pod lookup table
	for k, n := range c.bindings {
		if n == node.Name {
			delete(c.bindings, k)
		}
	}
	for _, p := range pods {
		c.bindings[util.GetNamespacedName(&p)] = node.Name
	}

	// Update partitioning kinds lookup table
	c.refreshPartitioningKinds()
}

func (c *ClusterState) DeletePod(namespacedName types.NamespacedName) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// Check if pod is known
	nodeName, bindingKnown := c.bindings[namespacedName]
	if !bindingKnown {
		return fmt.Errorf(
			"cannot delete pod %s/%s from cluster state: pod not found",
			namespacedName.Namespace,
			namespacedName.Name,
		)
	}

	// Always delete pod from lookup table
	defer delete(c.bindings, namespacedName)

	// If Pod's node does not exist
	// then nothing to do
	node, nodeFound := c.nodes[nodeName]
	if !nodeFound {
		return nil
	}

	// Remove Pod from node
	for _, pi := range node.Pods {
		if util.GetNamespacedName(pi.Pod) == namespacedName {
			if err := node.RemovePod(pi.Pod); err != nil {
				return err
			}
			c.nodes[nodeName] = node
			return nil
		}
	}

	return nil
}

func (c *ClusterState) UpdateUsage(pod v1.Pod) {
	// avoid acquiring lock if Pod is not assigned to any node yet
	if pod.Spec.NodeName == "" {
		return
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	// If the state does not contain the Pod's node
	// then we don't update anything
	nodeInfo, nodeFound := c.nodes[pod.Spec.NodeName]
	if !nodeFound {
		return
	}

	namespacedName := util.GetNamespacedName(&pod)
	cachedNodeName, bindingKnown := c.bindings[namespacedName]
	if bindingKnown {
		c.updateUsageForKnownPod(cachedNodeName, pod)
	} else if pod.Status.Phase == v1.PodRunning {
		nodeInfo.AddPod(&pod)
		c.nodes[pod.Spec.NodeName] = nodeInfo
	}

	// Update lookup table
	c.bindings[namespacedName] = pod.Spec.NodeName
}

func (c *ClusterState) refreshPartitioningKinds() {
	c.partitioningKinds = make(map[gpu.PartitioningKind]int)
	for _, n := range c.nodes {
		if node := n.Node(); node != nil {
			if kind, ok := gpu.GetPartitioningKind(*n.Node()); ok {
				c.partitioningKinds[kind]++
			}
		}
	}
}

func (c *ClusterState) updateUsageForKnownPod(cachedNodeName string, pod v1.Pod) {
	namespacedName := util.GetNamespacedName(&pod)
	nodeInfo := c.nodes[pod.Spec.NodeName]

	if pod.Spec.NodeName != cachedNodeName {
		// pod changed node, update old and new nodes
		delete(c.bindings, namespacedName)
		oldNode, oldNodeFound := c.nodes[cachedNodeName]
		if oldNodeFound {
			_ = oldNode.RemovePod(&pod)
		}
		if pod.Status.Phase == v1.PodRunning {
			nodeInfo.AddPod(&pod)
		}
	} else if pod.Status.Phase != v1.PodRunning {
		// pod is still on the same cached node, remove it if status changed
		_ = nodeInfo.RemovePod(&pod)
	}

	c.nodes[pod.Spec.NodeName] = nodeInfo
}

// IsPartitioningEnabled returns true if there is at least one node enabled for the provided kind of GPU partitioning
func (c *ClusterState) IsPartitioningEnabled(kind gpu.PartitioningKind) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	nNodes := c.partitioningKinds[kind]
	return nNodes > 0
}
