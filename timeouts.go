package codebook

import (
	"errors"
	"time"
)

type Timeouts struct {
	// ReloadInterval specifies how often should the cache be reloaded.
	// This duration is each time randomized by `Ranomizer` attribute.
	ReloadInterval time.Duration

	// ReloadDelay specifies how long should the cache block the next reload after last reload.
	// Useful when the cache is reloaded very often due to many invalidations (`InvalidateAll`
	// function calls) and data loading is slow or expensive.
	// When `InvalidateAll` function is called during this time, the cache is not immediatelly reloaded,
	// but the reload is postponed until `ReloadDelay` duration passes. Second and other `InvalidateAll`
	// calls during this "blocking" period are being ignored.
	// When `InvalidateAll` function is called outside of this "blocking" time, the cache is reloaded
	// immediatelly.
	// The next reload time is then set as ReloadInterval +/- ReloadInterval * Ranomizer.
	ReloadDelay time.Duration

	// Ranomizer specifies how much the timeouts/durations should be randomized.
	// Allowed values are from 0 to 1 (included).
	// Value 0 means no randomization, 0.1 means 10% randomization, etc.
	// is treated as 1.
	// e.g. real cache reload interval is set as `ReloadInterval` +/- `ReloadInterval` * `Ranomizer`.
	// All durations are being randomized each time they are set.
	Ranomizer float64
}

func (t *Timeouts) check() error {
	if t.ReloadDelay > t.ReloadInterval {
		return errors.New("ReloadDelay must be less than or equal to ReloadInterval")
	}

	if t.Ranomizer < 0 {
		return errors.New("Ranomizer cannot be negative")
	}
	if t.Ranomizer > 1 {
		return errors.New("Ranomizer cannot be greater than 1")
	}

	return nil
}
