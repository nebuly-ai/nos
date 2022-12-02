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

	batch        []T
	idleTimer    *time.Timer
	timeoutTimer *time.Timer
}

func NewBatcher[T any](timeoutDuration time.Duration, idleDuration time.Duration) Batcher[T] {
	idleTimer := time.NewTimer(0 * time.Millisecond)
	timeoutTimer := time.NewTimer(0 * time.Millisecond)
	return Batcher[T]{
		trigger:         make(chan T),
		timeoutDuration: timeoutDuration,
		idleDuration:    idleDuration,
		batchChan:       make(chan []T, 1),
		idleTimer:       idleTimer,
		timeoutTimer:    timeoutTimer,
	}
}

func NewBufferedBatcher[T any](timeoutDuration time.Duration, idleDuration time.Duration, bufferSize int) Batcher[T] {
	idleTimer := time.NewTimer(0 * time.Millisecond)
	timeoutTimer := time.NewTimer(0 * time.Millisecond)
	return Batcher[T]{
		trigger:         make(chan T, bufferSize),
		timeoutDuration: timeoutDuration,
		idleDuration:    idleDuration,
		batchChan:       make(chan []T, 1),
		idleTimer:       idleTimer,
		timeoutTimer:    timeoutTimer,
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
	b.Reset()

	sendBatch := func() {
		select {
		case b.batchChan <- b.batch:
		default:
		}
	}

	// Run
	for {
		select {
		case item := <-b.trigger:
			if len(b.batch) == 0 {
				ResetTimer(b.timeoutTimer, b.timeoutDuration)
			}
			b.batch = append(b.batch, item)
			ResetTimer(b.idleTimer, b.idleDuration)
		case <-b.idleTimer.C:
			sendBatch()
			b.stopTimersAndInitBatch()
		case <-b.timeoutTimer.C:
			sendBatch()
			b.stopTimersAndInitBatch()
		case <-ctx.Done():
			// Stop
			StopTimer(b.timeoutTimer)
			StopTimer(b.idleTimer)
			b.running = false
			return nil
		}
	}
}

func (b *Batcher[T]) Ready() chan []T {
	return b.batchChan
}

// Reset resets the batcher by resetting the timers and clearing the current batch
func (b *Batcher[T]) Reset() {
	b.stopTimersAndInitBatch()
	select {
	case <-b.batchChan:
	default:
	}
}

func (b *Batcher[T]) stopTimersAndInitBatch() {
	StopTimer(b.idleTimer)
	StopTimer(b.timeoutTimer)
	b.batch = make([]T, 0)
}
