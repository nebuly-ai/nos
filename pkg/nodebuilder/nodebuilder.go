package nodebuilder

import (
	v1 "k8s.io/api/core/v1"
)

type NodeBuilder struct {
	// channel where we should send the node built
	nodeBuilt chan *v1.Node
}

func NewNodeBuilder(nodeBuilt chan *v1.Node) *NodeBuilder {
	nodeBuilder := NodeBuilder{
		nodeBuilt: nodeBuilt,
	}
	return &nodeBuilder
}

func (nb *NodeBuilder) BuildNode(cloudProvider string, machineType string) {

}
