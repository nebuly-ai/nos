package iter_test

import (
	"github.com/nebuly-ai/nebulnetes/pkg/util/iter"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPermutationGenerator(t *testing.T) {
	testCases := []struct {
		name                 string
		sourceSlice          []string
		expectedPermutations [][]string
	}{
		{
			name:                 "Empty source slice",
			sourceSlice:          make([]string, 0),
			expectedPermutations: make([][]string, 0),
		},
		{
			name:        "Slice with unique elements",
			sourceSlice: []string{"A", "B", "C"},
			expectedPermutations: [][]string{
				{"A", "B", "C"},
				{"A", "C", "B"},
				{"B", "A", "C"},
				{"B", "C", "A"},
				{"C", "A", "B"},
				{"C", "B", "A"},
			},
		},
		{
			name:        "Slice with repeated elements",
			sourceSlice: []string{"A", "B", "A"},
			expectedPermutations: [][]string{
				{"A", "B", "A"},
				{"A", "A", "B"},
				{"B", "A", "A"},
				{"B", "A", "A"},
				{"A", "A", "B"},
				{"A", "B", "A"},
			},
		},
	}

	for _, tt := range testCases {
		generator := iter.NewPermutationGenerator[string](tt.sourceSlice)
		t.Run(tt.name, func(t *testing.T) {
			perms := make([][]string, 0)
			for i := 0; i < len(tt.expectedPermutations); i++ {
				assert.True(t, generator.Next())
				perms = append(perms, generator.Permutation())
			}
			assert.False(t, generator.Next())
			assert.ElementsMatch(t, tt.expectedPermutations, perms)
		})
	}
}
