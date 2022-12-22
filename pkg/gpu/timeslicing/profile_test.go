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

package timeslicing_test

import (
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/timeslicing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProfileName__GetMemorySizeGB(t *testing.T) {
	testCases := []struct {
		name        string
		profileName timeslicing.ProfileName
		expected    int
	}{
		{
			name:        "Invalid format, should return 0",
			profileName: "foo",
			expected:    0,
		},
		{
			name:        "Valid format",
			profileName: "nvidia.com/gpu-10gb",
			expected:    10,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.profileName.GetMemorySizeGB())
		})
	}
}

func TestProfileName__SmallerThan(t *testing.T) {
	testCases := []struct {
		name     string
		first    gpu.Slice
		second   gpu.Slice
		expected bool
	}{
		{
			name:     "Profiles are equal",
			first:    timeslicing.ProfileName("nvidia.com/gpu-10gb"),
			second:   timeslicing.ProfileName("nvidia.com/gpu-10gb"),
			expected: false,
		},
		{
			name:     "First is bigger",
			first:    timeslicing.ProfileName("nvidia.com/gpu-20gb"),
			second:   timeslicing.ProfileName("nvidia.com/gpu-10gb"),
			expected: false,
		},
		{
			name:     "First is smaller",
			first:    timeslicing.ProfileName("nvidia.com/gpu-10gb"),
			second:   timeslicing.ProfileName("nvidia.com/gpu-20gb"),
			expected: true,
		},
		{
			name:     "Not a valid format, memory should be considered 0",
			first:    timeslicing.ProfileName("nvidia.com/foo"),
			second:   timeslicing.ProfileName("nvidia.com/gpu-10gb"),
			expected: true,
		},
		{
			name:     "Not a time-slicing Profile, memory should be considered 0",
			first:    timeslicing.ProfileName("nvidia.com/gpu-10gb"),
			second:   mig.ProfileName("nvidia.com/mig-1g.5gb"),
			expected: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.first.SmallerThan(tt.second))
		})
	}
}
