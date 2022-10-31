package state

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type ClusterSnapshot struct {
	Nodes map[string]framework.NodeInfo
}

func (c *ClusterSnapshot) Fork() {

}

func (c *ClusterSnapshot) Commit() {

}

func (c *ClusterSnapshot) Revert() {

}

func (c *ClusterSnapshot) GetLackingResources(pod v1.Pod) framework.Resource {
	podRequest := resource.ComputePodRequest(pod)
	totalAllocatable := framework.Resource{}
	totalRequested := framework.Resource{}
	for _, n := range c.Nodes {
		totalAllocatable = resource.Sum(totalAllocatable, *n.Allocatable)
		totalRequested = resource.Sum(totalRequested, *n.Requested)
	}
	available := resource.Subtract(totalAllocatable, totalRequested)

	res := resource.Subtract(available, resource.FromListToFramework(podRequest))
	return resource.Abs(res)
}

func (c *ClusterSnapshot) GetNodes() []framework.NodeInfo {
	res := make([]framework.NodeInfo, len(c.Nodes))
	i := 0
	for _, n := range c.Nodes {
		res[i] = n
		i++
	}
	return res
}

func (c *ClusterSnapshot) GetCurrentPartitioning() map[string]NodePartitioning {
	res := make(map[string]NodePartitioning)
	return res
}

func (c *ClusterSnapshot) GetNode(name string) (framework.NodeInfo, bool) {
	node, found := c.Nodes[name]
	return node, found
}

func (c *ClusterSnapshot) SetNode(nodeInfo framework.NodeInfo) {
	c.Nodes[nodeInfo.Node().Name] = nodeInfo
}

func (c *ClusterSnapshot) AddPod(nodeName string, pod v1.Pod) error {
	node, found := c.GetNode(nodeName)
	if !found {
		return fmt.Errorf("could not find node %s in cluster snapshot", nodeName)
	}
	node.AddPod(&pod)
	return nil
}
