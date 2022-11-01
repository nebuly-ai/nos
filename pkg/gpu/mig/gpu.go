package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"reflect"
)

// Geometry corresponds to the MIG Geometry of a GPU,
// namely the MIG profiles of the GPU with the respective quantity.
type Geometry map[ProfileName]int

func (g Geometry) AsResourceList() v1.ResourceList {
	res := make(v1.ResourceList)
	for p, v := range g {
		resourceName := v1.ResourceName(fmt.Sprintf("%s%s", constant.NvidiaMigResourcePrefix, p))
		quantity := res[resourceName]
		quantity.Add(*resource.NewQuantity(int64(v), resource.DecimalSI))
		res[resourceName] = quantity
	}
	return res
}

type GPUModel string

type GPU struct {
	index                int
	model                GPUModel
	allowedMigGeometries []Geometry
	usedMigDevices       map[ProfileName]int
	freeMigDevices       map[ProfileName]int
}

func NewGPU(model GPUModel, index int, usedMigDevices, freeMigDevices map[ProfileName]int) (GPU, error) {
	allowedGeometries, ok := gpuModelToAllowedMigGeometries[model]
	if !ok {
		return GPU{}, fmt.Errorf("model %q is not associated with any known GPU", model)
	}
	return GPU{
		index:                index,
		model:                model,
		allowedMigGeometries: allowedGeometries,
		usedMigDevices:       usedMigDevices,
		freeMigDevices:       freeMigDevices,
	}, nil
}

func (g *GPU) GetIndex() int {
	return g.index
}

func (g *GPU) GetModel() GPUModel {
	return g.model
}

func (g *GPU) GetGeometry() Geometry {
	res := make(Geometry)

	for profile, quantity := range g.usedMigDevices {
		res[profile] += quantity
	}
	for profile, quantity := range g.freeMigDevices {
		res[profile] += quantity
	}

	return res
}

func (g *GPU) ApplyGeometry(geometry Geometry) error {
	// Check if geometry is allowed
	if !g.AllowsGeometry(geometry) {
		return fmt.Errorf("GPU model %s does not allow the provided MIG geometry", g.model)
	}
	// Check if new geometry deletes used devices
	for usedProfile, usedQuantity := range g.usedMigDevices {
		if geometry[usedProfile] < usedQuantity {
			return fmt.Errorf("cannot apply MIG geometry: cannot delete MIG devices being used")
		}
	}
	// Apply geometry by changing free devices
	for profile, quantity := range geometry {
		g.freeMigDevices[profile] = quantity
	}
	// Delete all free devices not included in the new geometry
	for profile := range g.freeMigDevices {
		if _, ok := geometry[profile]; !ok {
			delete(g.freeMigDevices, profile)
		}
	}
	return nil
}

func (g *GPU) AllowsGeometry(geometry Geometry) bool {
	for _, allowedGeometry := range g.GetAllowedGeometries() {
		if reflect.DeepEqual(geometry, allowedGeometry) {
			return true
		}
	}
	return false
}

func (g *GPU) GetAllowedGeometries() []Geometry {
	return g.allowedMigGeometries
}
