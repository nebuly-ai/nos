package mig

import (
	"fmt"
)

// Geometry corresponds to the MIG Geometry of a GPU,
// namely the MIG profiles of the GPU with the respective quantity.
type Geometry map[ProfileName]uint8

type GPUModel string

const (
	Model_A100_SMX4_40GB = "A100-SMX4-40GB"
	Model_A30            = "A30"
)

type GPU interface {
	GetAllowedMigGeometries() []Geometry
	GetCurrentMigGeometry() Geometry
	GetIndex() int
	GetModel() GPUModel
}

type baseGpu struct {
	index int
	model GPUModel
}

func (g baseGpu) GetIndex() int {
	return g.index
}

func (g baseGpu) GetModel() GPUModel {
	return g.model
}

func (g baseGpu) GetCurrentMigGeometry() Geometry {
	return Geometry{}
}

type A100_SMX4_40GB struct {
	baseGpu
}

func (a A100_SMX4_40GB) GetAllowedMigGeometries() []Geometry {
	return A100_SMX4_40GB_AllowedGeometries
}

type A30 struct {
	baseGpu
}

func (a A30) GetAllowedMigGeometries() []Geometry {
	return A30_AllowedGeometries
}

func NewGPU(model GPUModel, index int) (GPU, error) {
	if model == Model_A100_SMX4_40GB {
		return A100_SMX4_40GB{
			baseGpu: baseGpu{
				index: index,
				model: model,
			},
		}, nil
	}

	if model == Model_A30 {
		return A30{
			baseGpu: baseGpu{
				index: index,
				model: model,
			},
		}, nil
	}

	return nil, fmt.Errorf("model %q is not associated with any known GPU", model)
}
