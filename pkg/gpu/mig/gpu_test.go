package mig_test

import (
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/stretchr/testify/assert"
	"testing"
)

func newGpuOrPanic(model mig.GPUModel, index int, usedMigDevices, freeMigDevices map[mig.ProfileName]int) mig.GPU {
	gpu, err := mig.NewGPU(model, index, usedMigDevices, freeMigDevices)
	if err != nil {
		panic(err)
	}
	return gpu
}

func TestGPU__GetCurrentMigGeometry(t *testing.T) {
	testCases := []struct {
		name             string
		gpu              mig.GPU
		expectedGeometry mig.Geometry
	}{
		{
			name:             "Empty GPU",
			gpu:              newGpuOrPanic(mig.GPUModel_A30, 0, make(map[mig.ProfileName]int), make(map[mig.ProfileName]int)),
			expectedGeometry: mig.Geometry{},
		},
		//{},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedGeometry, tt.gpu.GetCurrentMigGeometry())
		})
	}
}
