package mig

import "fmt"

// Geometry corresponds to the MIG Geometry of a GPU,
// namely the MIG profiles of the GPU with the respective quantity.
type Geometry map[ProfileName]int

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

func (g GPU) GetIndex() int {
	return g.index
}

func (g GPU) GetModel() GPUModel {
	return g.model
}

func (g GPU) GetCurrentMigGeometry() Geometry {
	res := make(Geometry)

	for profile, quantity := range g.usedMigDevices {
		res[profile] += quantity
	}
	for profile, quantity := range g.freeMigDevices {
		res[profile] += quantity
	}

	return res
}

func (g GPU) GetAllowedMigGeometries() []Geometry {
	return g.allowedMigGeometries
}
