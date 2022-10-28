package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// Node is a k8s node with MIG GPUs
type Node struct {
	Name string
	gpus []GPU
}

func NewNode(n framework.NodeInfo) (Node, error) {
	gpusModel, err := getGPUsModel(*n.Node())
	if err != nil {
		return Node{}, err
	}
	gpus, err := getNodeGPUs(n, gpusModel)
	if err != nil {
		return Node{}, err
	}
	return Node{Name: n.Node().Name, gpus: gpus}, nil
}

func getNodeGPUs(nodeInfo framework.NodeInfo, gpusModel GPUModel) ([]GPU, error) {
	result := make([]GPU, 0)

	//statusAnnotations, _ := GetGPUAnnotationsFromNode(*nodeInfo.Node())
	//for _, a := range statusAnnotations {
	//	gpu, err := NewGPU(gpusModel, a.GetGPUIndex())
	//}

	//for resourceName, quantity := range nodeInfo.Requested.ScalarResources {
	//	if IsNvidiaMigDevice(resourceName) {
	//
	//	}
	//}

	return result, nil
}

func getGPUsModel(node v1.Node) (GPUModel, error) {
	if val, ok := node.Labels[constant.LabelNvidiaProduct]; ok {
		return GPUModel(val), nil
	}
	return "", fmt.Errorf("cannot get NVIDIA GPU model: node does not have label %q", constant.LabelNvidiaProduct)
}

func (n *Node) GetAllocatableScalarResources() v1.ResourceList {
	return make(v1.ResourceList)
}

func (n *Node) UpdateGeometryFor(migResource v1.ResourceName) error {
	return nil
}

func (n *Node) GetGPUsGeometry() map[string]v1.ResourceList {
	return nil
}
