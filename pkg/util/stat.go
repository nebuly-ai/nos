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

package util

import (
	"gonum.org/v1/gonum/stat/combin"
)

type PermutationGenerator[K any] struct {
	generator      *combin.PermutationGenerator
	sourceSlice    []K
	sourceSliceLen int
}

func NewPermutationGenerator[K any](sourceSlice []K) PermutationGenerator[K] {
	n := len(sourceSlice)
	return PermutationGenerator[K]{
		generator:      combin.NewPermutationGenerator(n, n),
		sourceSlice:    sourceSlice,
		sourceSliceLen: n,
	}
}

func (p *PermutationGenerator[K]) Permutation() []K {
	perm := p.generator.Permutation(nil)
	res := make([]K, p.sourceSliceLen)
	for i, index := range perm {
		res[i] = p.sourceSlice[index]
	}
	return res
}

func (p *PermutationGenerator[K]) Next() bool {
	if p.sourceSliceLen == 0 {
		return false
	}
	return p.generator.Next()
}

// IterPermutations calls `f` providing as argument each possible permutation `slice`.
// It stops iterating if `f` returns either `false` or an error, and
// it returns an error if any call to `f` returns error.
func IterPermutations[K any](slice []K, f func(k []K) (bool, error)) error {
	gen := NewPermutationGenerator(slice)
	for i := 0; gen.Next(); i++ {
		perm := gen.Permutation()
		continueIterating, err := f(perm)
		if err != nil {
			return err
		}
		if !continueIterating {
			return nil
		}
	}
	return nil
}
