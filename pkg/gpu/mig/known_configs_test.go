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

package mig

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKnownGeometries(t *testing.T) {
	testCases := []struct {
		name      string
		gpuModel  GPUModel
		maxMemory int
		maxGi     int
	}{
		{
			name:      "A100-40GB",
			gpuModel:  GPUModel_A100_SXM4_40GB,
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
			var geometryTotalMemory int
			var geometryTotalGi int
			for profile, quantity := range geometryList {
				assert.True(t, profile.isValid())
				geometryTotalMemory += profile.getMemorySlices() * quantity
				geometryTotalGi += profile.getGiSlices() * quantity
			}
			assert.LessOrEqual(t, geometryTotalMemory, tt.maxMemory)
			assert.LessOrEqual(t, geometryTotalGi, tt.maxGi)
		}
	}
}
