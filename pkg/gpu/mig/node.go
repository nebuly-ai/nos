package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type Node struct {
	Name string
	gpus []GPU
}

func NewNode(n framework.NodeInfo) (Node, error) {
	if n.Node() == nil {
		return Node{}, fmt.Errorf("cannot create Node from a nil v1.Node")
	}
	gpusModel, err := getGPUsModel(*n.Node())
	if err != nil {
		return Node{Name: n.Node().Name, gpus: make([]GPU, 0)}, nil
	}
	gpus, err := extractGPUs(n, gpusModel)
	if err != nil {
		return Node{}, err
	}
	return Node{Name: n.Node().Name, gpus: gpus}, nil
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

func (n *Node) UpdateGeometryFor(migResource v1.ResourceName) error {
	return nil
}

func (n *Node) GetGPUsGeometry() map[string]v1.ResourceList {
	return nil
}
