package util

import (
	"context"
	"fmt"
	"time"
)

type Batcher[T any] struct {
	trigger         chan T
	idleDuration    time.Duration
	timeoutDuration time.Duration
	batchChan       chan []T
	running         bool
}

func NewBatcher[T any](timeoutDuration time.Duration, idleDuration time.Duration) Batcher[T] {
	return Batcher[T]{
		trigger:         make(chan T),
		timeoutDuration: timeoutDuration,
		idleDuration:    idleDuration,
		batchChan:       make(chan []T),
	}
}

func NewBufferedBatcher[T any](timeoutDuration time.Duration, idleDuration time.Duration, bufferSize int) Batcher[T] {
	return Batcher[T]{
		trigger:         make(chan T, bufferSize),
		timeoutDuration: timeoutDuration,
		idleDuration:    idleDuration,
		batchChan:       make(chan []T),
	}
}

func (b *Batcher[T]) Add(item T) {
	select {
	case b.trigger <- item:
	default:
		return
	}
}

func (b *Batcher[T]) Start(ctx context.Context) error {
	// Check if the batcher has already been started
	if b.running {
		return fmt.Errorf("batcher already started")
	}
	b.running = true

	// Init
	var batch []T
	var idleTimer = time.NewTimer(0 * time.Millisecond)
	var timeoutTimer = time.NewTimer(0 * time.Millisecond)
	var reset = func() {
		batch = make([]T, 0)
		stopTimer(idleTimer)
		stopTimer(timeoutTimer)
	}
	reset()

	// Run
	for {
		select {
		case item := <-b.trigger:
			if len(batch) == 0 {
				resetTimer(timeoutTimer, b.timeoutDuration)
			}
			batch = append(batch, item)
			resetTimer(idleTimer, b.idleDuration)
		case <-idleTimer.C:
			b.batchChan <- batch
			reset()
		case <-timeoutTimer.C:
			b.batchChan <- batch
			reset()
		case <-ctx.Done():
			// Stop
			stopTimer(timeoutTimer)
			stopTimer(idleTimer)
			b.running = false
			return nil
		}
	}
}

func (b *Batcher[T]) Ready() chan []T {
	return b.batchChan
}

func resetTimer(timer *time.Timer, duration time.Duration) {
	stopTimer(timer)
	timer.Reset(duration)
}

// stopTimer stops and drains the provided timer
func stopTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}
