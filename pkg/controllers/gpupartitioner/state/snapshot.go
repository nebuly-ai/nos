package state

import v1 "k8s.io/api/core/v1"

type ClusterSnapshot interface {
	GetLackingResources(pod v1.Pod) v1.ResourceList

	Fork()
	Commit()
	Revert()
}
