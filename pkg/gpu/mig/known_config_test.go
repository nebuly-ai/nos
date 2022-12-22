package mig_test

import (
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/timeslicing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidateDefaultKnownConfigs(t *testing.T) {
	assert.NoError(t, mig.ValidateConfigs(mig.GetKnownGeometries()))
}

func TestValidateConfigs(t *testing.T) {
	testCases := []struct {
		name        string
		configs     map[gpu.Model][]gpu.Geometry
		errExpected bool
	}{
		{
			name:        "Empty",
			configs:     map[gpu.Model][]gpu.Geometry{},
			errExpected: true,
		},
		{
			name: "Profile is not mig",
			configs: map[gpu.Model][]gpu.Geometry{
				gpu.GPUModel_A30: {
					{
						timeslicing.ProfileName("1gb"): 1,
						mig.Profile1g10gb:              2,
					},
					{
						mig.Profile1g10gb: 1,
					},
				},
			},
			errExpected: true,
		},
		{
			name: "Invalid MIG profile",
			configs: map[gpu.Model][]gpu.Geometry{
				gpu.GPUModel_A30: {
					{
						mig.ProfileName("invalid"): 1,
						mig.Profile1g10gb:          2,
					},
					{
						mig.Profile1g10gb: 1,
					},
				},
			},
			errExpected: true,
		},
		{
			name: "Negative quantity",
			configs: map[gpu.Model][]gpu.Geometry{
				gpu.GPUModel_A30: {
					{
						mig.Profile1g10gb: -1,
					},
					{
						mig.Profile1g10gb: 1,
					},
				},
			},
			errExpected: true,
		},
	}

	for _, tt := range testCases {
		err := mig.ValidateConfigs(tt.configs)
		if tt.errExpected {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}
