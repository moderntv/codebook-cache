package aggregator

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/moderntv/codebook-cache/internal/test_utils"
)

func TestAggregator(t *testing.T) {
	t.Run("Timing", testAggregatorTiming)
	t.Run("Context", testAggregatorContext)
}

func testAggregatorTiming(t *testing.T) {
	t.Parallel()

	var messageCounter atomic.Int64

	aggregateFunc := func() {
		messageCounter.Add(1)
	}

	aggregator := NewSimpleAggregator(
		context.Background(),
		test_utils.Logger(),
		3*time.Second,
		aggregateFunc,
	)

	// 0 sec
	assert.Equal(t, int64(0), messageCounter.Load())
	aggregator.Notify()
	time.Sleep(100 * time.Millisecond) // wait for goroutine to start
	assert.Equal(t, int64(1), messageCounter.Load())
	time.Sleep(2 * time.Second)
	// 2.1 sec
	assert.Equal(t, int64(1), messageCounter.Load())
	aggregator.Notify()
	aggregator.Notify()
	aggregator.Notify()
	time.Sleep(100 * time.Millisecond) // wait for goroutine to start
	assert.Equal(t, int64(1), messageCounter.Load())
	time.Sleep(2 * time.Second)
	// 4.2 sec
	assert.Equal(t, int64(2), messageCounter.Load())
	aggregator.Notify()
	time.Sleep(100 * time.Millisecond) // wait for goroutine to start
	assert.Equal(t, int64(2), messageCounter.Load())
	time.Sleep(1 * time.Second)
	// 5.3 sec
	assert.Equal(t, int64(2), messageCounter.Load())
	time.Sleep(2 * time.Second)
	// 7.3 sec
	assert.Equal(t, int64(3), messageCounter.Load())
	time.Sleep(4 * time.Second)
	// 11.3 sec
	assert.Equal(t, int64(3), messageCounter.Load())
	aggregator.Notify()
	time.Sleep(100 * time.Millisecond) // wait for goroutine to start
	assert.Equal(t, int64(4), messageCounter.Load())
	aggregator.Notify()
	time.Sleep(100 * time.Millisecond) // wait for goroutine to start
	assert.Equal(t, int64(4), messageCounter.Load())
	time.Sleep(1 * time.Second)
	// 12.5 sec
	assert.Equal(t, int64(4), messageCounter.Load())
}

func testAggregatorContext(t *testing.T) {
	t.Parallel()

	var messageCounter atomic.Int64

	ctx, cancel := context.WithCancel(context.Background())
	aggregator := NewSimpleAggregator(
		ctx,
		test_utils.Logger(),
		3*time.Second,
		func() {
			messageCounter.Add(1)
		},
	)

	// 0 sec
	assert.Equal(t, int64(0), messageCounter.Load())
	aggregator.Notify()
	time.Sleep(100 * time.Millisecond) // wait for goroutine to start
	assert.Equal(t, int64(1), messageCounter.Load())
	time.Sleep(2 * time.Second)
	// 2 sec
	assert.Equal(t, int64(1), messageCounter.Load())
	aggregator.Notify()
	aggregator.Notify()
	aggregator.Notify()
	assert.Equal(t, int64(1), messageCounter.Load())
	cancel()
	time.Sleep(5 * time.Second)
	// 7 sec
	assert.Equal(t, int64(1), messageCounter.Load())
}
