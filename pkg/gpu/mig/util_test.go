package mig

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestIsNvidiaMigDevice(t *testing.T) {
	tests := []struct {
		name         string
		resourceName string
		expected     bool
	}{
		{
			name:         "Empty string",
			resourceName: "",
			expected:     false,
		},
		{
			name:         "Generic resource",
			resourceName: "nvidia.com/gpu",
			expected:     false,
		},
		{
			name:         "Malformed NVIDIA MIG",
			resourceName: "nvidia.com/mig-1ga1gb",
			expected:     false,
		},
		{
			name:         "Valid NVIDIA MIG",
			resourceName: "nvidia.com/mig-1g.1gb",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsNvidiaMigDevice(v1.ResourceName(tt.resourceName)))
		})
	}
}

func TestExtractMemoryGBFromMigDevice(t *testing.T) {
	tests := []struct {
		name          string
		resourceName  string
		errorExpected bool
		expected      int64
	}{
		{
			name:          "Empty string",
			resourceName:  "",
			errorExpected: true,
		},
		{
			name:          "Generic resource",
			resourceName:  "nvidia.com/gpu",
			errorExpected: true,
		},
		{
			name:          "Malformed NVIDIA MIG",
			resourceName:  "nvidia.com/mig-1g12agb",
			errorExpected: true,
		},
		{
			name:          "Malformed NVIDIA MIG - multiple occurrences",
			resourceName:  "nvidia.com/mig-1g.1gb15gb",
			errorExpected: true,
		},
		{
			name:          "Valid NVIDIA MIG",
			resourceName:  "nvidia.com/mig-1g.16gb",
			errorExpected: false,
			expected:      16,
		},
		{
			name:          "Valid NVIDIA MIG-like format",
			resourceName:  "foo/1g2gb",
			errorExpected: false,
			expected:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory, err := ExtractMemoryGBFromMigFormat(v1.ResourceName(tt.resourceName))
			if tt.errorExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, memory)
			}
		})
	}
}
