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

package util_test

import (
	"github.com/nebuly-ai/nebulnetes/pkg/util"
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
		generator := util.NewPermutationGenerator[string](tt.sourceSlice)
		t.Run(tt.name, func(t *testing.T) {
			perms := make([][]string, 0)
			for i := 0; generator.Next(); i++ {
				perms = append(perms, generator.Permutation())
			}
			assert.False(t, generator.Next())
			assert.ElementsMatch(t, tt.expectedPermutations, perms)
		})
	}
}
