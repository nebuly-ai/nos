package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
)

type Node struct {
	Name string
	GPUs []GPU
}

func NewNode(n v1.Node) (Node, error) {
	gpusModel, err := getGPUsModel(n)
	if err != nil {
		return Node{Name: n.Name, GPUs: make([]GPU, 0)}, nil
	}
	gpus, err := extractGPUs(n, gpusModel)
	if err != nil {
		return Node{}, err
	}
	return Node{Name: n.Name, GPUs: gpus}, nil
}

func extractGPUs(node v1.Node, gpusModel GPUModel) ([]GPU, error) {
	result := make([]GPU, 0)

	statusAnnotations, _ := GetGPUAnnotationsFromNode(node)
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

// UpdateGeometryFor tries to update the MIG geometry of the GPUs of the node in order to create the MIG profile
// provided as argument. It does that by either creating a new MIG profile (if there is enough capacity) or by
// deleting free (e.g. unused) MIG profiles to make up space and create the required profile, according to the
// allowed MIG geometries of each GPU.
//
// UpdateGeometryFor returns an error if is not possible to update the GPUs geometry for creating
// the specified MIG profile.
func (n *Node) UpdateGeometryFor(profile ProfileName) error {
	// If there are no GPUs, then there's nothing to do
	if len(n.GPUs) == 0 {
		return fmt.Errorf("cannot update geometry because node does not have any MIG GPU")
	}

	for _, gpu := range n.GPUs {
		// If Node already provides required profiles, then there's nothing to do
		if gpu.freeMigDevices[profile] > 0 {
			return nil
		}
		// Try to apply candidate geometries
		for _, allowedGeometry := range gpu.GetAllowedGeometries() {
			nFreeProfilesWithGeometry := allowedGeometry[profile] - gpu.usedMigDevices[profile]
			if nFreeProfilesWithGeometry > 0 {
				if err := gpu.ApplyGeometry(allowedGeometry); err == nil {
					// New geometry applied, we're done
					return nil
				}
			}
		}
	}

	return fmt.Errorf("")
}

// GetGeometry returns the overall MIG geometry of the node, which corresponds to the sum of the MIG geometry of all
// the GPUs present in the Node.
func (n *Node) GetGeometry() Geometry {
	res := make(Geometry)
	for _, g := range n.GPUs {
		for p, q := range g.GetGeometry() {
			res[p] += q
		}
	}
	return res
}

// HasFreeMigResources returns true if the Node has at least one MIG GPU with a free MIG resource.
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
