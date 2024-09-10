package codebook

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/moderntv/codebook-cache/internal/aggregator"
	"github.com/moderntv/codebook-cache/internal/invalidation"
	"github.com/moderntv/codebook-cache/internal/memsize"
	metrics_pkg "github.com/moderntv/codebook-cache/internal/metrics"
	"github.com/moderntv/codebook-cache/internal/utils"
)

type Cache[K comparable, T any] struct {
	// static attributes (does not change its value after initialization)
	ctx            context.Context
	log            zerolog.Logger
	metrics        *metrics_pkg.Metrics
	name           string
	timeouts       Timeouts
	loadAllFunc    LoadAllFunc[K, T]
	reloadChan     chan bool
	aggregator     *aggregator.SimpleAggregator
	memSizeEnabled bool
	// dynamic attributes (not using mutex)
	memSizeValue atomic.Uint64
	data         atomic.Value
	// attributes protected by mutex
	mu          sync.Mutex
	isReloading bool
	nextReload  *time.Time
}

func New[K comparable, T any](params Params[K, T]) (c *Cache[K, T], err error) {
	err = params.check()
	if err != nil {
		return
	}

	var metrics *metrics_pkg.Metrics
	if params.MetricsRegistry != nil {
		metrics, err = metrics_pkg.New(params.Name, params.MetricsRegistry)
		if err != nil {
			return
		}
	}

	log := params.Log.With().Str("cache", params.Name).Logger()

	c = &Cache[K, T]{
		ctx:            params.Context,
		log:            log,
		metrics:        metrics,
		name:           params.Name,
		timeouts:       params.Timeouts,
		loadAllFunc:    params.LoadAllFunc,
		reloadChan:     make(chan bool, 1),
		memSizeEnabled: params.MemsizeEnabled,
	}

	if params.Timeouts.ReloadDelay > 0 {
		c.aggregator = aggregator.NewSimpleAggregator(
			params.Context,
			log,
			params.Timeouts.ReloadDelay,
			func() {
				_ = c.reload(true)
			},
		)
	} else {
		c.log.Warn().Msg("invalidations aggregation is disabled")
	}

	err = c.reload(true)
	if err != nil {
		return
	}

	// set next reload and time checker
	c.initPeriodicReload()

	// invalidation messages
	if params.Invalidations != nil {
		c.initInvalidations(params.Invalidations)
	} else {
		c.log.Warn().Msg("invalidations are disabled")
	}

	return
}

func (c *Cache[K, T]) Get(ID K) *T {
	entries := c.GetAll()
	// no additional locking is needed here, because the cache is never modified (just replaced)
	entry, exists := entries[ID]
	if !exists {
		return nil
	}

	return entry
}

func (c *Cache[K, T]) GetAll() (entries map[K]*T) {
	entries = c.data.Load().(map[K]*T) // cache is always set
	return
}

func (c *Cache[K, T]) InvalidateAll() {
	if c.aggregator != nil {
		c.aggregator.Notify()
		return
	}

	go c.reload(true)
}

func (c *Cache[K, T]) initInvalidations(invalidations *Invalidations) {
	natsHelper := invalidation.NewNatsHelper(c.log, invalidations.Nats, invalidations.Prefix)

	for subject, message := range invalidations.Messages {
		// subscribe to invalidation message
		natsHelper.Subscribe(subject, message, func(_ proto.Message) {
			// invalidate whole repository
			c.log.Trace().Msg("Invalidate")
			if c.metrics != nil {
				c.metrics.ReceivedNatsInvalidations.Inc()
			}
			c.InvalidateAll()
		})
	}
}

func (c *Cache[K, T]) initPeriodicReload() {
	baseInterval := c.timeouts.ReloadInterval
	if baseInterval == 0 {
		c.log.Warn().Msg("periodic reload disabled")
		return
	}

	// reload already performed, we will set nextReload in future
	nextReloadTime := time.Now().Add(utils.RandomizeDuration(baseInterval, c.timeouts.Ranomizer))
	c.nextReload = &nextReloadTime

	// start goroutine for automatic reloading
	// when another reload occures, it will send message to reloadChan, which
	// will stop the timer and start new one
	go func() {
		for {
			c.mu.Lock()
			duration := time.Until(*c.nextReload)
			c.mu.Unlock()

			timer := time.NewTimer(duration)

			select {
			case _, more := <-c.reloadChan:
				timer.Stop()
				if !more {
					c.log.Debug().Msg("periodic reload stopped")
					return
				}

			case <-timer.C:
				_ = c.reload(false)

			case <-c.ctx.Done():
				close(c.reloadChan)
			}
		}
	}()
}

func (c *Cache[K, T]) setLoading(start time.Time, force bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isReloading {
		return errors.New("already reloading")
	}

	if !force && (c.nextReload != nil && start.Before(*c.nextReload)) {
		return errors.New("cannot reload yet")
	}

	c.isReloading = true
	return nil
}

// reloads cache and sets reload timer
func (c *Cache[K, T]) reload(force bool) (err error) {
	start := time.Now()
	err = c.setLoading(start, force)
	if err != nil {
		c.log.Trace().Err(err).Msg("reload skipped")
		return
	}

	c.log.Debug().Msg("loading started")

	entries, err := c.loadAllFunc(c.ctx)
	if err == nil {
		c.data.Store(entries)

		if c.memSizeEnabled {
			go c.updateMemSize()
		}

	} else {
		c.log.Warn().
			Err(err).
			Float64("duration_s", time.Since(start).Round(time.Millisecond).Seconds()).
			Msg("loading failed")
	}

	if c.metrics != nil {
		c.metrics.LoadCount.Inc()
	}

	logEvent := c.log.Debug()

	var newNextReloadTime *time.Time
	if c.timeouts.ReloadInterval > 0 {
		t := start.Add(utils.RandomizeDuration(c.timeouts.ReloadInterval, c.timeouts.Ranomizer))
		newNextReloadTime = &t
		logEvent = logEvent.Time("next_reload", t)
	}

	// critical section start
	c.mu.Lock()
	c.isReloading = false
	if newNextReloadTime != nil {
		c.nextReload = newNextReloadTime
	}
	c.mu.Unlock()
	// critical section end

	logEvent.
		Int("count", len(entries)).
		Float64("duration_s", time.Since(start).Round(time.Millisecond).Seconds()).
		Msg("loading finished")
	if c.metrics != nil {
		c.metrics.ItemsCount.Set(float64(len(entries)))
	}

	// notify periodic reload goroutine that next reload time changed
	if newNextReloadTime != nil {
		c.reloadChan <- true
	}

	return
}

func (c *Cache[K, T]) updateMemSize() {
	// handle potential panic (calculating size should not affect running app)
	defer func() {
		err := recover()
		if err != nil {
			c.log.Warn().
				Interface("err", err).
				Msg("panic occurred during cache size calculation")
		}
	}()

	entries := c.GetAll()

	// c.log.Trace().
	// 	Int("entries_count", len(entries)).
	// 	Msg("calculating cache size in memory")

	// entries size
	size := memsize.Entries(entries)
	c.memSizeValue.Store(size)
	if c.metrics != nil {
		c.metrics.MemoryUsage.Set(float64(size))
	}

	c.log.Trace().
		Uint64("B", size).
		Float32("MB", (float32(size) / 1000000)).
		Msg("memory size calculating finished")
}
