package state

import v1 "k8s.io/api/core/v1"

type GPUPartitioning struct {
	GPUIndex  int
	Resources map[v1.ResourceName]int
}

type NodePartitioning struct {
	GPUs []GPUPartitioning
}

type PartitioningState map[string]NodePartitioning
