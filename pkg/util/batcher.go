package util

import (
	"context"
	"time"
)

type Batcher[T any] struct {
	trigger      chan T
	currentBatch []T
	idleDuration time.Duration
}

func NewBatcher[T any](idleDuration time.Duration) Batcher[T] {
	return Batcher[T]{
		trigger:      make(chan T),
		idleDuration: idleDuration,
	}
}

func NewBufferedBatcher[T any](idleDuration time.Duration, bufferSize int) Batcher[T] {
	return Batcher[T]{
		trigger:      make(chan T, bufferSize),
		idleDuration: idleDuration,
	}
}

func (b *Batcher[T]) Add(item T) {
	select {
	case b.trigger <- item:
	default:
		return
	}
}

func (b *Batcher[T]) WaitBatch(ctx context.Context, timeout time.Duration) []T {
	var batch = make([]T, 0)

	idleTimer := time.NewTimer(b.idleDuration)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case item := <-b.trigger:
			batch = append(batch, item)
			if !idleTimer.Stop() {
				<-idleTimer.C
			}
			idleTimer.Reset(b.idleDuration)
		case <-idleTimer.C:
			return batch
		case <-ctx.Done():
			return batch
		}
	}
}
