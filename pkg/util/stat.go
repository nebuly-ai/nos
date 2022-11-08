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
