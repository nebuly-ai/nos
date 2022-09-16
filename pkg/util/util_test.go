package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetKeys(t *testing.T) {
	tests := []struct {
		name     string
		maps     []map[string]int
		expected []string
	}{
		{
			name:     "empty args list",
			maps:     make([]map[string]int, 0),
			expected: make([]string, 0),
		},
		{
			name: "multiple maps with overlapping keys",
			maps: []map[string]int{
				{
					"key-1": 1,
					"key-2": 2,
					"key-3": 3,
				},
				{
					"key-1": 1,
					"key-4": 5,
					"key-5": 4,
				},
				{
					"key-1": 1,
					"key-2": 5,
				},
			},
			expected: []string{
				"key-1",
				"key-2",
				"key-3",
				"key-4",
				"key-5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := GetKeys(tt.maps...)
			assert.ElementsMatch(t, tt.expected, keys)
		})
	}
}

func TestMax(t *testing.T) {
	testsInt := []struct {
		name     string
		v1       int
		v2       int
		expected int
	}{
		{
			name:     "v1 == v2",
			v1:       10,
			v2:       10,
			expected: 10,
		},
		{
			name:     "v1 > v2",
			v1:       11,
			v2:       10,
			expected: 11,
		},
		{
			name:     "v1 < v2",
			v1:       9,
			v2:       10,
			expected: 10,
		},
	}

	for _, tt := range testsInt {
		t.Run(tt.name, func(t *testing.T) {
			max := Max(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, max)
		})
	}

	testsFloat := []struct {
		name     string
		v1       float64
		v2       float64
		expected float64
	}{
		{
			name:     "v1 == v2",
			v1:       10.001,
			v2:       10.001,
			expected: 10.001,
		},
		{
			name:     "v1 > v2",
			v1:       10.1,
			v2:       10,
			expected: 10.1,
		},
		{
			name:     "v1 < v2",
			v1:       10,
			v2:       10.1,
			expected: 10.1,
		},
	}

	for _, tt := range testsFloat {
		t.Run(tt.name, func(t *testing.T) {
			max := Max(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, max)
		})
	}
}

func TestMin(t *testing.T) {
	testsInt := []struct {
		name     string
		v1       int
		v2       int
		expected int
	}{
		{
			name:     "v1 == v2",
			v1:       10,
			v2:       10,
			expected: 10,
		},
		{
			name:     "v1 > v2",
			v1:       11,
			v2:       10,
			expected: 10,
		},
		{
			name:     "v1 < v2",
			v1:       9,
			v2:       10,
			expected: 9,
		},
	}

	for _, tt := range testsInt {
		t.Run(tt.name, func(t *testing.T) {
			max := Min(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, max)
		})
	}

	testsFloat := []struct {
		name     string
		v1       float64
		v2       float64
		expected float64
	}{
		{
			name:     "v1 == v2",
			v1:       10.001,
			v2:       10.001,
			expected: 10.001,
		},
		{
			name:     "v1 > v2",
			v1:       10.1,
			v2:       10,
			expected: 10,
		},
		{
			name:     "v1 < v2",
			v1:       10,
			v2:       10.1,
			expected: 10,
		},
	}

	for _, tt := range testsFloat {
		t.Run(tt.name, func(t *testing.T) {
			max := Min(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, max)
		})
	}
}
