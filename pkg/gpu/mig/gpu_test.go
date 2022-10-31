package mig_test

import (
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"
)

func newGpuOrPanic(model mig.GPUModel, index int, usedMigDevices, freeMigDevices map[mig.ProfileName]int) mig.GPU {
	gpu, err := mig.NewGPU(model, index, usedMigDevices, freeMigDevices)
	if err != nil {
		panic(err)
	}
	return gpu
}

func TestGPU__GetMigGeometry(t *testing.T) {
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
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedGeometry, tt.gpu.GetGeometry())
		})
	}
}

func TestGeometry__AsResourceList(t *testing.T) {
	testCases := []struct {
		name     string
		geometry mig.Geometry
		expected v1.ResourceList
	}{
		{
			name:     "Empty geometry",
			geometry: mig.Geometry{},
			expected: make(v1.ResourceList),
		},
		{
			name: "Multiple resources",
			geometry: mig.Geometry{
				mig.Profile1g5gb:  3,
				mig.Profile1g10gb: 2,
			},
			expected: v1.ResourceList{
				mig.Profile1g5gb.AsResourceName():  *resource.NewQuantity(3, resource.DecimalSI),
				mig.Profile1g10gb.AsResourceName(): *resource.NewQuantity(2, resource.DecimalSI),
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.geometry.AsResourceList())
		})
	}
}
