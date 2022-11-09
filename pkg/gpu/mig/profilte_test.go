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
