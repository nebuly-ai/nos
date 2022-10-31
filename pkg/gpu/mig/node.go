package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type Node struct {
	Name string
	GPUs []GPU
}

func NewNode(n framework.NodeInfo) (Node, error) {
	if n.Node() == nil {
		return Node{}, fmt.Errorf("cannot create Node from a nil v1.Node")
	}
	gpusModel, err := getGPUsModel(*n.Node())
	if err != nil {
		return Node{Name: n.Node().Name, GPUs: make([]GPU, 0)}, nil
	}
	gpus, err := extractGPUs(n, gpusModel)
	if err != nil {
		return Node{}, err
	}
	return Node{Name: n.Node().Name, GPUs: gpus}, nil
}

func extractGPUs(nodeInfo framework.NodeInfo, gpusModel GPUModel) ([]GPU, error) {
	result := make([]GPU, 0)

	statusAnnotations, _ := GetGPUAnnotationsFromNode(*nodeInfo.Node())
	for gpuIndex, gpuAnnotations := range statusAnnotations.GroupByGpuIndex() {
		usedMigDevices := make(map[ProfileName]int)
		freeMigDevices := make(map[ProfileName]int)
		for _, a := range gpuAnnotations {
			if a.IsUsed() {
				usedMigDevices[a.GetMigProfileName()] = a.Quantity
			}
			if a.IsFree() {
				freeMigDevices[a.GetMigProfileName()] = a.Quantity
			}
		}
		gpu, err := NewGPU(gpusModel, gpuIndex, usedMigDevices, freeMigDevices)
		if err != nil {
			return nil, err
		}
		result = append(result, gpu)
	}

	return result, nil
}

func getGPUsModel(node v1.Node) (GPUModel, error) {
	if val, ok := node.Labels[constant.LabelNvidiaProduct]; ok {
		return GPUModel(val), nil
	}
	return "", fmt.Errorf("cannot get NVIDIA GPU model: node does not have label %q", constant.LabelNvidiaProduct)
}

func (n *Node) UpdateGeometryFor(profile ProfileName, quantity int) error {
	return nil
}

func (n *Node) GetGeometry() Geometry {
	res := make(Geometry)
	for _, g := range n.GPUs {
		for p, q := range g.GetGeometry() {
			res[p] += q
		}
	}
	return res
}

func (n *Node) HasFreeMigResources() bool {
	if len(n.GPUs) == 0 {
		return false
	}
	for _, gpu := range n.GPUs {
		if len(gpu.freeMigDevices) > 0 {
			return true
		}
	}
	return false
}

// HasFree returns true if the node has enough free resources for providing (even by changing its MIG geometry)
// the amount of MIG profiles provided as argument.
//func (n *Node) HasFree(profile ProfileName, quantity int) bool {
//	// check if the node already has
//	for _, g := range n.GPUs {
//		for p, q := range g.freeMigDevices {
//			if p == profile && q >= quantity {
//				return true
//			}
//		}
//	}
//	return false
//}
