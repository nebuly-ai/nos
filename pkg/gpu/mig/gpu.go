package mig

import (
	"fmt"
)

// Geometry corresponds to the MIG Geometry of a GPU,
// namely the MIG profiles of the GPU with the respective quantity.
type Geometry map[ProfileName]uint8

type GPUModel string

type GPU struct {
	index                int
	model                GPUModel
	allowedMigGeometries []Geometry
	usedMigDevices       map[ProfileName]int
	freeMigDevices       map[ProfileName]int
}

func (g GPU) GetIndex() int {
	return g.index
}

func (g GPU) GetModel() GPUModel {
	return g.model
}

func (g GPU) GetCurrentMigGeometry() Geometry {
	return Geometry{}
}

func (g GPU) GetAllowedMigGeometries() []Geometry {
	return g.allowedMigGeometries
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
