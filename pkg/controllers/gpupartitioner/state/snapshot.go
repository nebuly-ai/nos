package state

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type ClusterSnapshot struct {
	nodes map[string]framework.NodeInfo
}

func (c *ClusterSnapshot) Fork() {

}

func (c *ClusterSnapshot) Commit() {

}

func (c *ClusterSnapshot) Revert() {

}

func (c *ClusterSnapshot) GetLackingScalarResources(pod v1.Pod) v1.ResourceList {
	//podRequest := resource.ComputePodRequest(pod)
	//diff := quota.Subtract(c.available, podRequest)
	//result := make(v1.ResourceList)
	//for r, q := range diff {
	//	// available - request < 0 means that there's a lack
	//	// of this resource for scheduling the Pod
	//	if q.CmpInt64(0) < 0 {
	//		result[r] = q
	//	}
	//}
	//return result
	return nil
}

func (c *ClusterSnapshot) GetNodes() []framework.NodeInfo {
	res := make([]framework.NodeInfo, len(c.nodes))
	i := 0
	for _, n := range c.nodes {
		res[i] = n
		i++
	}
	return res
}

func (c *ClusterSnapshot) GetNode(name string) (framework.NodeInfo, bool) {
	node, found := c.nodes[name]
	return node, found
}

func (c *ClusterSnapshot) AddPod(nodeName string, pod v1.Pod) error {
	node, found := c.GetNode(nodeName)
	if !found {
		return fmt.Errorf("could not find node %s in cluster snapshot", nodeName)
	}
	node.AddPod(&pod)
	return nil
}

func (c *ClusterSnapshot) UpdateAllocatableScalarResources(nodeName string, scalarResources v1.ResourceList) error {
	node, found := c.GetNode(nodeName)
	if !found {
		return fmt.Errorf("could not find node %s in cluster snapshot", nodeName)
	}
	node.Allocatable.ScalarResources = resource.FromListToFramework(scalarResources).ScalarResources
	c.nodes[nodeName] = node
	return nil
}
