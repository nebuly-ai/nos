package state

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sync"
)

func NewClusterState() ClusterState {
	return ClusterState{
		nodes:    make(map[string]framework.NodeInfo),
		bindings: make(map[types.NamespacedName]string),
	}
}

type ClusterState struct {
	nodes    map[string]framework.NodeInfo
	bindings map[types.NamespacedName]string // lookup table: Pod => NodeName

	mtx sync.RWMutex
}

func (c *ClusterState) GetNode(nodeName string) (framework.NodeInfo, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	node, ok := c.nodes[nodeName]
	return node, ok
}

func (c *ClusterState) GetSnapshot() ClusterSnapshot {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return NewClusterSnapshot(c.nodes)
}

func (c *ClusterState) deleteNode(name string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	delete(c.nodes, name)
	for key, nodeName := range c.bindings {
		if nodeName == name {
			delete(c.bindings, key)
		}
	}
}

func (c *ClusterState) updateNode(node v1.Node, pods []v1.Pod) {
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
}

func (c *ClusterState) deletePod(namespacedName types.NamespacedName) error {
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

func (c *ClusterState) updateUsage(pod v1.Pod) {
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
