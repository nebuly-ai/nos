package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"strings"
)

type Device struct {
	resource.Device
	// GpuId is the Index of the parent GPU to which the MIG device belongs to
	GpuIndex int
}

// FullResourceName returns the full resource name of the MIG device, including
// the name of the resource corresponding to the MIG profile and the index
// of the GPU to which it belongs to.
func (m Device) FullResourceName() string {
	return fmt.Sprintf("%d/%s", m.GpuIndex, m.ResourceName)
}

// GetMIGProfileName returns the name of the MIG profile associated to the device
//
// Example:
//
//	Resource name: nvidia.com/mig-1g.10gb
//	GetMIGProfileName() -> 1g.10gb
func (m Device) GetMIGProfileName() string {
	return strings.TrimPrefix(m.ResourceName.String(), "nvidia.com/mig-")
}

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
