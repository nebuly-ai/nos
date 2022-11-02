package state

import v1 "k8s.io/api/core/v1"

type GPUPartitioning struct {
	GPUIndex  int
	Resources v1.ResourceList
}

type NodePartitioning struct {
	GPUs []GPUPartitioning
}

type ClusterPartitioning map[string]NodePartitioning
