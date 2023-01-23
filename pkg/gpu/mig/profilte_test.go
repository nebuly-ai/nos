/*
 * Copyright 2023 nebuly.com.
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

func TestProfileName__getMemorySlices(t *testing.T) {
	assert.Equal(t, 20, Profile3g20gb.getMemorySlices())
}

func TestProfileName__getGiSlices(t *testing.T) {
	assert.Equal(t, 3, Profile3g20gb.getGiSlices())
}

func TestProfileList__GroupByGpuIndex(t *testing.T) {
	testCases := []struct {
		name     string
		list     ProfileList
		expected map[int]ProfileList
	}{
		{
			name:     "Empty list",
			list:     make(ProfileList, 0),
			expected: make(map[int]ProfileList),
		},
		{
			name: "Multiple GPUs",
			list: ProfileList{
				{
					GpuIndex: 0,
					Name:     Profile2g10gb,
				},
				{
					GpuIndex: 0,
					Name:     Profile1g5gb,
				},
				{
					GpuIndex: 1,
					Name:     Profile1g5gb,
				},
			},
			expected: map[int]ProfileList{
				0: {
					{
						GpuIndex: 0,
						Name:     Profile2g10gb,
					},
					{
						GpuIndex: 0,
						Name:     Profile1g5gb,
					},
				},
				1: {
					{
						GpuIndex: 1,
						Name:     Profile1g5gb,
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.list.GroupByGPU()
			assert.Equal(t, tt.expected, res)
		})
	}
}

func TestProfileName__GreaterThan(t *testing.T) {
	testCases := []struct {
		name     string
		profile  ProfileName
		other    ProfileName
		expected bool
	}{
		{
			name:     "ProfileName are equals",
			profile:  Profile1g6gb,
			other:    Profile1g6gb,
			expected: false,
		},
		{
			name:     "Same memory, higher Gi",
			profile:  Profile3g20gb,
			other:    Profile4g20gb,
			expected: true,
		},
		{
			name:     "Same Gi, higher memory",
			profile:  Profile1g5gb,
			other:    Profile1g10gb,
			expected: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.profile.SmallerThan(tt.other))
		})
	}
}
