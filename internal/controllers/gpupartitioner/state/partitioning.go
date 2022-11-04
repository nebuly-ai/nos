package state

import (
	v1 "k8s.io/api/core/v1"
	"reflect"
)

type GPUPartitioning struct {
	GPUIndex  int
	Resources map[v1.ResourceName]int
}

type NodePartitioning struct {
	GPUs []GPUPartitioning
}

func (n NodePartitioning) Equal(other NodePartitioning) bool {
	if len(n.GPUs) != len(other.GPUs) {
		return false
	}
	return reflect.DeepEqual(n.GPUs, other.GPUs)
}

type PartitioningState map[string]NodePartitioning

func (p PartitioningState) IsEmpty() bool {
	return len(p) == 0
}

func (p PartitioningState) Equal(other PartitioningState) bool {
	if len(p) != len(other) {
		return false
	}
	for node, nodePartitioning := range p {
		if !nodePartitioning.Equal(other[node]) {
			return false
		}
	}
	return true
}
