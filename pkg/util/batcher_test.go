package util_test

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
	"time"
)

func TestBatcher__WaitBatch(t *testing.T) {
	t.Run("Items added before starting a batch should be ignored", func(t *testing.T) {
		podBatcher := util.NewBatcher[v1.Pod](10 * time.Millisecond)
		podBatcher.Add(v1.Pod{})
		podBatcher.Add(v1.Pod{})

		batch := podBatcher.WaitBatch(context.Background(), 10*time.Millisecond)
		assert.Empty(t, batch)
	})

	t.Run("Should block for timeout duration at most", func(t *testing.T) {
		podBatcher := util.NewBatcher[v1.Pod](10 * time.Second)
		timeout := 20 * time.Millisecond
		start := time.Now()
		podBatcher.WaitBatch(context.Background(), timeout)
		now := time.Now()
		assert.WithinDuration(t, now, start.Add(timeout), 2*time.Millisecond)
	})

	t.Run("Cancelling context should stop waiting", func(t *testing.T) {
		podBatcher := util.NewBatcher[v1.Pod](10 * time.Second)
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)

		var start time.Time
		var end time.Time
		go func() {
			start = time.Now()
			podBatcher.WaitBatch(ctx, 10*time.Second)
			end = time.Now()
		}()

		cancel()
		assert.NotNil(t, start)
		assert.NotNil(t, end)
		assert.WithinDuration(t, end, start, 10*time.Millisecond)
	})

	t.Run("If no items are added, should wait until idle duration elapses", func(t *testing.T) {
		idleDuration := 100 * time.Millisecond
		podBatcher := util.NewBatcher[v1.Pod](idleDuration)
		start := time.Now()
		podBatcher.WaitBatch(context.Background(), 10*time.Second)
		assert.WithinDuration(t, time.Now(), start.Add(idleDuration), 2*time.Millisecond)
	})

	t.Run("Adding an item should reset idle timeout", func(t *testing.T) {
		idleDuration := 30 * time.Millisecond
		podBatcher := util.NewBatcher[v1.Pod](idleDuration)
		go func() {
			podBatcher.Add(v1.Pod{})
			time.Sleep(20 * time.Millisecond)
			podBatcher.Add(v1.Pod{})
		}()
		start := time.Now()
		podBatcher.WaitBatch(context.Background(), 10*time.Second)
		assert.WithinDuration(t, time.Now(), start.Add(idleDuration*2), 10*time.Millisecond)
	})

	t.Run("Batch should include all added items", func(t *testing.T) {
		pods := []v1.Pod{
			factory.BuildPod("ns-1", "pd-1").Get(),
			factory.BuildPod("ns-1", "pd-2").Get(),
			factory.BuildPod("ns-1", "pd-3").Get(),
		}
		batcher := util.NewBufferedBatcher[v1.Pod](100*time.Millisecond, 3)

		var batchChan = make(chan []v1.Pod)
		go func() {
			batch := batcher.WaitBatch(context.Background(), 5*time.Second)
			batchChan <- batch
		}()
		for _, p := range pods {
			batcher.Add(p)
		}

		batch := <-batchChan
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
