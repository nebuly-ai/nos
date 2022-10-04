package gpu

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	resourceutil "github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type resourceWithDeviceId struct {
	resourceName v1.ResourceName
	deviceId     string
}

func (r resourceWithDeviceId) isMIGDevice() bool {
	return resourceutil.IsNvidiaMigDevice(r.resourceName)
}

type Device struct {
	// ResourceName is the name of the resource exposed to k8s
	// (e.g. nvidia.com/gpu, nvidia.com/mig-2g10gb, etc.)
	ResourceName v1.ResourceName
	// DeviceId is the actual ID of the underlying device
	// (e.g. ID of the GPU, ID of the MIG device, etc.)
	DeviceId string
}

type MIGDevice struct {
	Device
	// GpuId is the Index of the parent GPU to which the MIG device belongs to
	GpuIndex int
}

// FullResourceName returns the full resource name of the MIG device, including
// the name of the resource corresponding to the MIG profile and the index
// of the GPU to which it belongs to.
func (m MIGDevice) FullResourceName() string {
	return fmt.Sprintf("%d/%s", m.GpuIndex, m.ResourceName)
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
		//gpu := GPU{
		//	modelCode: gpuModel,
		//	memoryMb:  resource.GetNvidiaGPUsMemoryMb(node),
		//}
		//result = append(result, gpu)
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
