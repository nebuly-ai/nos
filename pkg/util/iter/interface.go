package iter

type Generator[K any] interface {
	Next() []K
}
