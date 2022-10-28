package mig

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKnownGeometries(t *testing.T) {
	testCases := []struct {
		name      string
		gpuModel  GPUModel
		maxMemory uint8
		maxGi     uint8
	}{
		{
			name:      "A100-40GB",
			gpuModel:  GPUModel_A100_SMX4_40GB,
			maxMemory: 40,
			maxGi:     7,
		},
		{
			name:      "A30",
			gpuModel:  GPUModel_A30,
			maxMemory: 24,
			maxGi:     7,
		},
	}

	for _, tt := range testCases {
		availableGeometries := gpuModelToAllowedMigGeometries[tt.gpuModel]
		for _, geometryList := range availableGeometries {
			var geometryTotalMemory uint8
			var geometryTotalGi uint8
			for profile, quantity := range geometryList {
				geometryTotalMemory += profile.getMemorySlices() * quantity
				geometryTotalGi += profile.getGiSlices() * quantity
			}
			assert.LessOrEqual(t, geometryTotalMemory, tt.maxMemory)
			assert.LessOrEqual(t, geometryTotalGi, tt.maxGi)
		}
	}
}
