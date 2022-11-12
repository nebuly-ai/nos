package util_test

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"sync"
	"testing"
	"time"
)

func TestBatcher__Ready(t *testing.T) {
	const testTimeout = 3 * time.Second

	t.Run("Adding items to batch should never block", func(t *testing.T) {
		timeoutDuration := 10 * time.Millisecond
		idleDuration := 10 * time.Millisecond
		podBatcher := util.NewBatcher[v1.Pod](timeoutDuration, idleDuration)

		done := make(chan struct{})
		go func() {
			podBatcher.Add(v1.Pod{})
			done <- struct{}{}
		}()

		select {
		case <-done: // success
		case <-time.NewTimer(testTimeout).C:
			assert.Fail(t, "test timed out")
		}
	})

	t.Run("Items added before starting the batcher should be ignored", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		timeoutDuration := 10 * time.Millisecond
		idleDuration := 10 * time.Millisecond
		podBatcher := util.NewBatcher[v1.Pod](timeoutDuration, idleDuration)
		podBatcher.Add(v1.Pod{})
		podBatcher.Add(v1.Pod{})

		// Start batcher
		go func() {
			assert.NoError(t, podBatcher.Start(ctx))
		}()

		// Batch is empty, so it should never be ready
		timer := time.NewTimer(20 * time.Millisecond)
		select {
		case <-podBatcher.Ready():
			assert.Fail(t, "Batch was not expected to be ready")
		case <-timer.C:
			cancel()
		}
	})

	t.Run("Should be ready after idle duration if no other items are added to the batch", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		timeoutDuration := 200 * time.Millisecond
		idleDuration := 10 * time.Millisecond
		podBatcher := util.NewBufferedBatcher[v1.Pod](timeoutDuration, idleDuration, 1)

		// Start batcher
		go func() {
			assert.NoError(t, podBatcher.Start(ctx))
		}()

		// Start a batch
		podBatcher.Add(v1.Pod{})
		start := time.Now()

		select {
		case batch := <-podBatcher.Ready():
			now := time.Now()
			assert.Len(t, batch, 1)
			assert.WithinDuration(t, now, start.Add(idleDuration), 20*time.Millisecond)
		case <-time.NewTimer(testTimeout).C:
			assert.Fail(t, "test timed out")
		}
	})

	t.Run("Cancelling context should make the batch ready", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		timeoutDuration := 20 * time.Millisecond
		idleDuration := 10 * time.Millisecond
		podBatcher := util.NewBufferedBatcher[v1.Pod](timeoutDuration, idleDuration, 1)

		// Start the batcher
		go func() {
			assert.NoError(t, podBatcher.Start(ctx))
		}()

		var start time.Time
		var end time.Time
		go func() {
			start = time.Now()
			<-podBatcher.Ready()
			end = time.Now()
		}()

		cancel()
		assert.NotNil(t, start)
		assert.NotNil(t, end)
		assert.WithinDuration(t, end, start, 10*time.Millisecond)
	})

	t.Run("Adding an item should reset idle timeout", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		timeoutDuration := 500 * time.Millisecond
		idleDuration := 50 * time.Millisecond
		podBatcher := util.NewBufferedBatcher[v1.Pod](timeoutDuration, idleDuration, 1)

		// Start the batcher
		go func() {
			assert.NoError(t, podBatcher.Start(ctx))
		}()

		// Add some pods to the batch in order to reset the idle timer
		var start time.Time
		go func() {
			start = time.Now()
			podBatcher.Add(v1.Pod{})
			time.Sleep(25 * time.Millisecond)
			podBatcher.Add(v1.Pod{})
			time.Sleep(25 * time.Millisecond)
			podBatcher.Add(v1.Pod{})
		}()

		// Check idle timer gets reset after adding pods
		select {
		case <-podBatcher.Ready():
			assert.Greater(t, time.Since(start), idleDuration*2)
			assert.Less(t, time.Since(start), timeoutDuration)
		case <-time.NewTimer(testTimeout).C:
			assert.Fail(t, "test timed out")
		}
	})

	t.Run("Batch should be ready after timeout duration at most, even if items are still being added", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		timeoutDuration := 40 * time.Millisecond
		idleDuration := 20 * time.Millisecond
		podBatcher := util.NewBufferedBatcher[v1.Pod](timeoutDuration, idleDuration, 1)

		// Start the batcher
		go func() {
			assert.NoError(t, podBatcher.Start(ctx))
		}()

		// Add some pods to the batch in order to reset the idle timer
		var start time.Time
		go func() {
			start = time.Now()
			for i := 0; i < 10; i++ {
				podBatcher.Add(v1.Pod{})
				time.Sleep(5 * time.Millisecond)
			}
		}()

		// Check idle timer gets reset after adding pods
		select {
		case <-podBatcher.Ready():
			assert.Greater(t, time.Since(start), timeoutDuration)
		case <-time.NewTimer(testTimeout).C:
			assert.Fail(t, "test timed out")
		}
	})

	t.Run("Starting a batcher that is already running should return an error", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		timeoutDuration := 20 * time.Millisecond
		idleDuration := 10 * time.Millisecond
		podBatcher := util.NewBufferedBatcher[v1.Pod](timeoutDuration, idleDuration, 1)

		// Start the batcher
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			wg.Done()
			assert.NoError(t, podBatcher.Start(ctx))
		}()
		// Start again the batcher
		wg.Wait()
		assert.Error(t, podBatcher.Start(ctx))
	})

	t.Run("After a batcher stops it should be possible to start it again", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		timeoutDuration := 20 * time.Millisecond
		idleDuration := 10 * time.Millisecond
		podBatcher := util.NewBufferedBatcher[v1.Pod](timeoutDuration, idleDuration, 1)

		// Start the batcher
		go func() {
			assert.NoError(t, podBatcher.Start(ctx))
		}()
		time.Sleep(100 * time.Millisecond)
		// Stop the batcher
		cancel()
		time.Sleep(100 * time.Millisecond)

		// Start again the batcher
		ctx = context.Background()
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		go func() {
			assert.NoError(t, podBatcher.Start(ctx))
		}()
	})

	t.Run("Batch should include all added items", func(t *testing.T) {
		pods := []v1.Pod{
			factory.BuildPod("ns-1", "pd-1").Get(),
			factory.BuildPod("ns-1", "pd-2").Get(),
			factory.BuildPod("ns-1", "pd-3").Get(),
		}
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		timeoutDuration := 50 * time.Millisecond
		idleDuration := 10 * time.Millisecond
		podBatcher := util.NewBufferedBatcher[v1.Pod](timeoutDuration, idleDuration, 5)

		// Start the batcher
		go func() {
			assert.NoError(t, podBatcher.Start(ctx))
		}()

		// Add pods to batch
		go func() {
			for _, p := range pods {
				podBatcher.Add(p)
			}
		}()

		var batch []v1.Pod
		select {
		case batch = <-podBatcher.Ready():
		case <-time.NewTimer(testTimeout).C:
			assert.Fail(t, "test timed out")
		}
		expectedPodNames := make([]string, len(pods))
		for i, p := range pods {
			expectedPodNames[i] = p.Name
		}
		actualPodNames := make([]string, 0)
		for _, p := range batch {
			actualPodNames = append(actualPodNames, p.Name)
		}
		assert.Equal(t, expectedPodNames, actualPodNames)
	})
}
