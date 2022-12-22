/*
 * Copyright 2022 Nebuly.ai
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
