package components

import v1 "k8s.io/api/core/v1"

type NodeSelector struct {
}

// GetNodeAffinity returns the NodeAffinity that best suits the hardware kinds provided as argument.
// The method received as input a map containing hardware kinds (for instance Nvidia-GPU-A100) and the respective
// priority
func (n *NodeSelector) GetNodeAffinity(hardwareKinds map[string]int) v1.NodeAffinity {
	// TODO: implement
	return v1.NodeAffinity{}
}
