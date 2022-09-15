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
