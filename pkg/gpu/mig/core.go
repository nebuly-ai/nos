package mig

import (
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type GPU struct {
}

func NewNode(n framework.NodeInfo) Node {
	return Node{
		Name: n.Node().Name,
		gpus: getGPUs(*n.Node()),
	}
}

func getGPUs(node v1.Node) []GPU {
	result := make([]GPU, 0)
	gpuModel := resource.GetNvidiaGPUsModel(node)
	if gpuModel == "" {
		return result
	}
	for i := 0; i < resource.GetNvidiaGPUsCount(node); i++ {
		gpu := GPU{
			//modelCode: gpuModel,
			//memoryMb:  resource.GetNvidiaGPUsMemoryMb(node),
		}
		result = append(result, gpu)
	}
	return result
}

type Node struct {
	Name string
	gpus []GPU
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
