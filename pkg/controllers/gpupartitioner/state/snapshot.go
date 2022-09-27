package state

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type ClusterSnapshot struct {
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

func (c *ClusterSnapshot) GetNodes(pod v1.Pod) []framework.NodeInfo {
	return nil
}

func (c *ClusterSnapshot) GetNode(name string) (framework.NodeInfo, bool) {
	return framework.NodeInfo{}, false
}

func (c *ClusterSnapshot) AddPod(node string, pod v1.Pod) error {
	return nil
}

func (c *ClusterSnapshot) UpdateAllocatableScalarResources(node string, scalarResources v1.ResourceList) error {
	return nil
}
