package mig

import (
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type Profile struct {
	Name               string
	MemoryGB           int
	NumSM              int
	AvailableInstances int
}

type GPUModel struct {
	Name          string
	TotalSM       int
	TotalMemoryGB int
	MIGProfiles   []Profile
}

var GPUModels = map[string]GPUModel{
	"A30": {
		Name:          "A100-SXM4",
		TotalSM:       4,
		TotalMemoryGB: 40,
		MIGProfiles: []Profile{
			{
				Name:               "1g6gb",
				AvailableInstances: 4,
				NumSM:              1,
				MemoryGB:           6,
			},
			{
				Name:               "2g12gb",
				AvailableInstances: 2,
				NumSM:              2,
				MemoryGB:           12,
			},
			{
				Name:               "4g24gb",
				AvailableInstances: 1,
				NumSM:              4,
				MemoryGB:           24,
			},
		},
	},
	"A100-SXM4": {
		Name:          "A100-SXM4",
		TotalSM:       7,
		TotalMemoryGB: 40,
		MIGProfiles: []Profile{
			{
				Name:               "1g5gb",
				AvailableInstances: 7,
				NumSM:              1,
				MemoryGB:           5,
			},
			{
				Name:               "2g10gb",
				AvailableInstances: 3,
				NumSM:              2,
				MemoryGB:           10,
			},
			{
				Name:               "3g20gb",
				AvailableInstances: 2,
				NumSM:              3,
				MemoryGB:           20,
			},
			{
				Name:               "4g20gb",
				AvailableInstances: 1,
				NumSM:              4,
				MemoryGB:           20,
			},
			{
				Name:               "7g40gb",
				AvailableInstances: 1,
				NumSM:              7,
				MemoryGB:           40,
			},
		},
	},
	"A100-SXM4-80GB": {
		Name:          "A100-SXM4-80GB",
		TotalSM:       7,
		TotalMemoryGB: 80,
		MIGProfiles: []Profile{
			{
				Name:               "1g10gb",
				AvailableInstances: 7,
				NumSM:              1,
				MemoryGB:           10,
			},
			{
				Name:               "2g20gb",
				AvailableInstances: 3,
				NumSM:              2,
				MemoryGB:           20,
			},
			{
				Name:               "3g40gb",
				AvailableInstances: 2,
				NumSM:              3,
				MemoryGB:           40,
			},
			{
				Name:               "4g40gb",
				AvailableInstances: 1,
				NumSM:              4,
				MemoryGB:           40,
			},
			{
				Name:               "7g80gb",
				AvailableInstances: 1,
				NumSM:              7,
				MemoryGB:           80,
			},
		},
	},
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
			modelCode: gpuModel,
			memoryMb:  resource.GetNvidiaGPUsMemoryMb(node),
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

type GPU struct {
	modelCode string
	memoryMb  int
}
