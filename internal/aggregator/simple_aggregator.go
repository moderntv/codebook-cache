package aggregator

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

type SimpleAggregator struct {
	// static attributes
	ctx            context.Context
	log            zerolog.Logger
	flushInterval  time.Duration // protection interval during which all incoming messages will be aggregated
	aggregatedFunc func()
	// dynamic attributes
	blockUntilMillis atomic.Int64 // timestamp in milliseconds until all incoming messages are being aggregated
	queued           atomic.Bool  // whether there has been at least one message received during aggregation
}

func NewSimpleAggregator(
	ctx context.Context,
	log zerolog.Logger,
	flushInterval time.Duration,
	aggregatedFunc func(),
) (aggregator *SimpleAggregator) {
	aggregator = &SimpleAggregator{
		ctx:            ctx,
		log:            log,
		flushInterval:  flushInterval,
		aggregatedFunc: aggregatedFunc,
	}

	return
}

// processMessage receives one invalidation message and do aggregation logic:
// - if first message is received, immediately calls invalidation function
// - if second message is received, runs timer which will calls invalidation function after desired duration (flush time)
// - third and following messages are being ignored.
// TLDR: tryies to invalidate immediately if possible, otherwise keeps desired interval (flush time) between invalidations
func (a *SimpleAggregator) Notify() {
	nowMillis := time.Now().UnixMilli()
	blockUntilMillis := a.blockUntilMillis.Load()

	// first message (no messages are already aggregated) or blocking time expired
	if blockUntilMillis == 0 || blockUntilMillis <= nowMillis {
		// set "blocking" time when all incoming messages are aggregated
		a.blockUntilMillis.Store(nowMillis + a.flushInterval.Milliseconds())
		go a.aggregatedFunc()

		return
	}

	// ignore message (is already waiting to flush message)
	wasQueued := a.queued.Swap(true)
	if wasQueued {
		return
	}

	// first message to be aggregated -> start timer
	flushAfterDuration := time.Duration(blockUntilMillis-nowMillis) * time.Millisecond
	go a.startAggregationTimer(flushAfterDuration)
}

func (a *SimpleAggregator) startAggregationTimer(duration time.Duration) {
	select {
	case <-time.After(duration):
		a.blockUntilMillis.Store(time.Now().UnixMilli() + a.flushInterval.Milliseconds())
		a.queued.Store(false)
		a.aggregatedFunc()

	case <-a.ctx.Done():
	}
}
