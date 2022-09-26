package state

import (
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	v1 "k8s.io/api/core/v1"
	quota "k8s.io/apiserver/pkg/quota/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type ClusterState interface {
	GetNodeInfo(nodeName string) (*framework.NodeInfo, error)
	GetLackingResources(pod v1.Pod) v1.ResourceList
	GetSnapshot() *ClusterSnapshot
}

func NewClusterState() ClusterState {
	return &clusterStateImpl{
		nodes:       make(map[string]node),
		allocatable: make(v1.ResourceList),
		available:   make(v1.ResourceList),
	}
}

type clusterStateImpl struct {
	nodes map[string]node

	allocatable v1.ResourceList
	available   v1.ResourceList
}

func (c *clusterStateImpl) GetLackingResources(pod v1.Pod) v1.ResourceList {
	podRequest := resource.ComputePodRequest(pod)
	diff := quota.Subtract(c.available, podRequest)
	result := make(v1.ResourceList)
	for r, q := range diff {
		// available - request < 0 means that there's a lack
		// of this resource for scheduling the Pod
		if q.CmpInt64(0) < 0 {
			result[r] = q
		}
	}
	return result
}

func (c *clusterStateImpl) GetNodeInfo(nodeName string) (*framework.NodeInfo, error) {
	return nil, nil
}

func (c *clusterStateImpl) GetSnapshot() *ClusterSnapshot {
	return nil
}

type node struct {
	// Name is the name of the node
	Name string
	// Allocatable is the total amount of resources that can be used by pods
	Allocatable v1.ResourceList
	// Available is allocatable minus anything allocated to pods.
	Available v1.ResourceList
}
