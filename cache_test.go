package codebook

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/moderntv/codebook-cache/internal/test_utils"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	t.Run("testCacheGet", testCacheGet)
	t.Run("parallelism", testCacheParallelism)
	t.Run("testCacheInvalidateWithoutAggregator", testCacheInvalidateWithoutAggregator)
	t.Run("testCacheInvalidateWithggregator", testCacheInvalidateWithggregator)
	t.Run("testCacheInvalidateAllSpeed", testCacheInvalidateAllSpeed)
	t.Run("testCacheMetrics", testCacheMetrics)
	t.Run("testCacheMemsizeCalculated", testCacheMemsizeCalculated)
	t.Run("TestCacheMemsizeManual", TestCacheMemsizeManual)
	t.Run("testBlockingPreload", testBlockingPreload)
	t.Run("testNonBlockingPreload", testNonBlockingPreload)
	t.Run("testBlockingPreloadWithError", testBlockingPreloadWithError)
	t.Run("testOnReload", testOnReload)
}

func testCacheGet(t *testing.T) {
	t.Parallel()

	valueMultiplier := 10

	c, err := New(Params[string, int]{
		Context: context.Background(),
		Log:     test_utils.Logger(),
		Name:    "testing_cache",
		LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
			return map[string]*int{
				"key1": test_utils.IntPointer(1 * valueMultiplier),
				"key2": test_utils.IntPointer(2 * valueMultiplier),
				"key3": test_utils.IntPointer(3 * valueMultiplier),
				"key4": test_utils.IntPointer(4 * valueMultiplier),
				"key5": test_utils.IntPointer(5 * valueMultiplier),
			}, nil
		},
		Timeouts: Timeouts{
			ReloadInterval: 2 * time.Second,
			ReloadDelay:    1 * time.Second,
			Randomizer:     0,
		},
	})

	assert.NoError(t, err)

	valueMultiplier = 100

	assert.Equal(t, test_utils.IntPointer(10), c.Get("key1"))
	assert.Equal(t, test_utils.IntPointer(20), c.Get("key2"))
	assert.Equal(t, test_utils.IntPointer(30), c.Get("key3"))
	assert.Equal(t, test_utils.IntPointer(40), c.Get("key4"))
	assert.Equal(t, test_utils.IntPointer(50), c.Get("key5"))
	assert.Nil(t, c.Get("key0"))
	assert.True(t, c.Get("key0") == nil)

	time.Sleep(2500 * time.Millisecond)

	assert.Equal(t, test_utils.IntPointer(100), c.Get("key1"))
	assert.Equal(t, test_utils.IntPointer(200), c.Get("key2"))
	assert.Equal(t, test_utils.IntPointer(300), c.Get("key3"))
	assert.Equal(t, test_utils.IntPointer(400), c.Get("key4"))
	assert.Equal(t, test_utils.IntPointer(500), c.Get("key5"))
	assert.Nil(t, c.Get("key0"))
	assert.True(t, c.Get("key0") == nil)

	expectedEntries := map[string]*int{
		"key1": test_utils.IntPointer(100),
		"key2": test_utils.IntPointer(200),
		"key3": test_utils.IntPointer(300),
		"key4": test_utils.IntPointer(400),
		"key5": test_utils.IntPointer(500),
	}

	assert.Equal(t, expectedEntries, c.GetAll())
}

func testCacheParallelism(t *testing.T) {
	t.Parallel()

	c, err := New(Params[string, int]{
		Context: context.Background(),
		Log:     test_utils.Logger(),
		Name:    "testing_cache",
		LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
			return map[string]*int{
				"key1": test_utils.IntPointer(1),
				"key2": test_utils.IntPointer(2),
				"key3": test_utils.IntPointer(3),
				"key4": test_utils.IntPointer(4),
				"key5": test_utils.IntPointer(5),
			}, nil
		},
		Timeouts: Timeouts{
			ReloadInterval: 10 * time.Millisecond,
			ReloadDelay:    0,
			Randomizer:     0,
		},
	})

	assert.NoError(t, err)

	routines := 100
	requests := 10000

	expectedEntries := map[string]*int{
		"key1": test_utils.IntPointer(1),
		"key2": test_utils.IntPointer(2),
		"key3": test_utils.IntPointer(3),
		"key4": test_utils.IntPointer(4),
		"key5": test_utils.IntPointer(5),
	}

	wg := sync.WaitGroup{}

	for i := 0; i < routines; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < requests; j++ {
				assert.Equal(t, test_utils.IntPointer(1), c.Get("key1"))
				assert.Equal(t, test_utils.IntPointer(2), c.Get("key2"))
				assert.Equal(t, test_utils.IntPointer(3), c.Get("key3"))
				assert.Equal(t, test_utils.IntPointer(4), c.Get("key4"))
				assert.Equal(t, test_utils.IntPointer(5), c.Get("key5"))
				assert.Nil(t, c.Get("key0"))
				assert.True(t, c.Get("key0") == nil)

				assert.Equal(t, expectedEntries, c.GetAll())
			}

			wg.Done()
		}()
	}

	wg.Wait()
}

func testCacheInvalidateWithoutAggregator(t *testing.T) {
	t.Parallel()

	valueMultiplier := 10

	c, err := New(Params[string, int]{
		Context: context.Background(),
		Log:     test_utils.Logger(),
		Name:    "testing_cache",
		LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
			return map[string]*int{
				"key1": test_utils.IntPointer(1 * valueMultiplier),
				"key2": test_utils.IntPointer(2 * valueMultiplier),
				"key3": test_utils.IntPointer(3 * valueMultiplier),
				"key4": test_utils.IntPointer(4 * valueMultiplier),
				"key5": test_utils.IntPointer(5 * valueMultiplier),
			}, nil
		},
		Timeouts: Timeouts{
			ReloadInterval: 5 * time.Second,
			ReloadDelay:    0,
			Randomizer:     0,
		},
	})

	assert.NoError(t, err)

	valueMultiplier = 100

	assert.Equal(t, test_utils.IntPointer(10), c.Get("key1"))
	assert.Equal(t, test_utils.IntPointer(20), c.Get("key2"))
	assert.Equal(t, test_utils.IntPointer(30), c.Get("key3"))
	assert.Equal(t, test_utils.IntPointer(40), c.Get("key4"))
	assert.Equal(t, test_utils.IntPointer(50), c.Get("key5"))
	assert.Nil(t, c.Get("key0"))

	c.InvalidateAll()

	time.Sleep(1 * time.Second)

	assert.Equal(t, test_utils.IntPointer(100), c.Get("key1"))
	assert.Equal(t, test_utils.IntPointer(200), c.Get("key2"))
	assert.Equal(t, test_utils.IntPointer(300), c.Get("key3"))
	assert.Equal(t, test_utils.IntPointer(400), c.Get("key4"))
	assert.Equal(t, test_utils.IntPointer(500), c.Get("key5"))
	assert.Nil(t, c.Get("key0"))

	expectedEntries := map[string]*int{
		"key1": test_utils.IntPointer(100),
		"key2": test_utils.IntPointer(200),
		"key3": test_utils.IntPointer(300),
		"key4": test_utils.IntPointer(400),
		"key5": test_utils.IntPointer(500),
	}

	assert.Equal(t, expectedEntries, c.GetAll())

	valueMultiplier = 1000
	time.Sleep(1000 * time.Millisecond)

	c.InvalidateAll()

	time.Sleep(1 * time.Second)

	key1Value := c.Get("key1")
	assert.Equal(t, test_utils.IntPointer(1000), key1Value, "key1 value is %v", *key1Value)
	assert.Equal(t, test_utils.IntPointer(2000), c.Get("key2"))
	assert.Equal(t, test_utils.IntPointer(3000), c.Get("key3"))
	assert.Equal(t, test_utils.IntPointer(4000), c.Get("key4"))
	assert.Equal(t, test_utils.IntPointer(5000), c.Get("key5"))
	assert.Nil(t, c.Get("key0"))
}

func testCacheInvalidateWithggregator(t *testing.T) {
	t.Parallel()

	valueMultiplier := 10
	c, err := New(Params[string, int]{
		Context: context.Background(),
		Log:     test_utils.Logger(),
		Name:    "testing_cache",
		LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
			return map[string]*int{
				"key1": test_utils.IntPointer(1 * valueMultiplier),
				"key2": test_utils.IntPointer(2 * valueMultiplier),
				"key3": test_utils.IntPointer(3 * valueMultiplier),
				"key4": test_utils.IntPointer(4 * valueMultiplier),
				"key5": test_utils.IntPointer(5 * valueMultiplier),
			}, nil
		},
		Timeouts: Timeouts{
			ReloadInterval: 5 * time.Second,
			ReloadDelay:    2 * time.Second,
			Randomizer:     0,
		},
	})

	assert.NoError(t, err)

	valueMultiplier = 100

	assert.Equal(t, test_utils.IntPointer(10), c.Get("key1"))
	assert.Equal(t, test_utils.IntPointer(20), c.Get("key2"))
	assert.Equal(t, test_utils.IntPointer(30), c.Get("key3"))
	assert.Equal(t, test_utils.IntPointer(40), c.Get("key4"))
	assert.Equal(t, test_utils.IntPointer(50), c.Get("key5"))
	assert.Nil(t, c.Get("key0"))

	c.InvalidateAll()

	time.Sleep(500 * time.Millisecond)

	// 0.5 s
	assert.Equal(t, test_utils.IntPointer(100), c.Get("key1"))
	assert.Equal(t, test_utils.IntPointer(200), c.Get("key2"))
	assert.Equal(t, test_utils.IntPointer(300), c.Get("key3"))
	assert.Equal(t, test_utils.IntPointer(400), c.Get("key4"))
	assert.Equal(t, test_utils.IntPointer(500), c.Get("key5"))
	assert.Nil(t, c.Get("key0"))

	valueMultiplier = 1000
	c.InvalidateAll()
	time.Sleep(500 * time.Millisecond)

	// 1 s
	assert.Equal(t, test_utils.IntPointer(100), c.Get("key1"))
	assert.Equal(t, test_utils.IntPointer(200), c.Get("key2"))
	assert.Equal(t, test_utils.IntPointer(300), c.Get("key3"))
	assert.Equal(t, test_utils.IntPointer(400), c.Get("key4"))
	assert.Equal(t, test_utils.IntPointer(500), c.Get("key5"))
	assert.Nil(t, c.Get("key0"))

	time.Sleep(2 * time.Second)

	// 4 s
	assert.Equal(t, test_utils.IntPointer(1000), c.Get("key1"))
	assert.Equal(t, test_utils.IntPointer(2000), c.Get("key2"))
	assert.Equal(t, test_utils.IntPointer(3000), c.Get("key3"))
	assert.Equal(t, test_utils.IntPointer(4000), c.Get("key4"))
	assert.Equal(t, test_utils.IntPointer(5000), c.Get("key5"))
	assert.Nil(t, c.Get("key0"))
}

func testCacheInvalidateAllSpeed(t *testing.T) {
	t.Parallel()

	var loadDelay time.Duration

	c, err := New(Params[string, int]{
		Context: context.Background(),
		Log:     test_utils.Logger(),
		Name:    "testing_cache",
		LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
			time.Sleep(loadDelay)

			return map[string]*int{
				"key1": test_utils.IntPointer(1),
				"key2": test_utils.IntPointer(2),
				"key3": test_utils.IntPointer(3),
				"key4": test_utils.IntPointer(4),
				"key5": test_utils.IntPointer(5),
			}, nil
		},
		Timeouts: Timeouts{
			ReloadInterval: 20 * time.Second,
			ReloadDelay:    0,
			Randomizer:     0,
		},
	})

	assert.NoError(t, err)

	routines := 100
	requests := 10

	testStart := time.Now()
	loadDelay = 5 * time.Second

	wg := sync.WaitGroup{}

	for i := 0; i < routines; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < requests; j++ {
				c.InvalidateAll()
				assert.Equal(t, test_utils.IntPointer(1), c.Get("key1"))

				time.Sleep(100 * time.Millisecond)
			}

			wg.Done()
		}()
	}

	wg.Wait()

	if time.Since(testStart) > 3*time.Second {
		t.Error("Test took too long")
	}
}

func testCacheMetrics(t *testing.T) {
	t.Parallel()

	valueMultiplier := 10

	c, err := New(Params[string, int]{
		Context:         context.Background(),
		Log:             test_utils.Logger(),
		MetricsRegistry: test_utils.Metrics("testing_cache"),
		Name:            "testing_cache",
		LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
			return map[string]*int{
				"key1": test_utils.IntPointer(1 * valueMultiplier),
				"key2": test_utils.IntPointer(2 * valueMultiplier),
				"key3": test_utils.IntPointer(3 * valueMultiplier),
				"key4": test_utils.IntPointer(4 * valueMultiplier),
				"key5": test_utils.IntPointer(5 * valueMultiplier),
			}, nil
		},
		Timeouts: Timeouts{
			ReloadInterval: 2 * time.Second,
			ReloadDelay:    0,
			Randomizer:     0,
		},
	})

	assert.NoError(t, err)

	valueMultiplier = 100

	assert.Equal(t, test_utils.IntPointer(10), c.Get("key1"))
	assert.Equal(t, test_utils.IntPointer(20), c.Get("key2"))
	assert.Equal(t, test_utils.IntPointer(30), c.Get("key3"))
	assert.Equal(t, test_utils.IntPointer(40), c.Get("key4"))
	assert.Equal(t, test_utils.IntPointer(50), c.Get("key5"))
	assert.Nil(t, c.Get("key0"))

	c.InvalidateAll()

	time.Sleep(300 * time.Millisecond)

	assert.Equal(t, test_utils.IntPointer(100), c.Get("key1"))
	assert.Equal(t, test_utils.IntPointer(200), c.Get("key2"))
	assert.Equal(t, test_utils.IntPointer(300), c.Get("key3"))
	assert.Equal(t, test_utils.IntPointer(400), c.Get("key4"))
	assert.Equal(t, test_utils.IntPointer(500), c.Get("key5"))
	assert.Nil(t, c.Get("key0"))

	expectedEntries := map[string]*int{
		"key1": test_utils.IntPointer(100),
		"key2": test_utils.IntPointer(200),
		"key3": test_utils.IntPointer(300),
		"key4": test_utils.IntPointer(400),
		"key5": test_utils.IntPointer(500),
	}

	assert.Equal(t, expectedEntries, c.GetAll())

	valueMultiplier = 1000

	c.InvalidateAll()

	time.Sleep(300 * time.Millisecond)

	assert.Equal(t, test_utils.IntPointer(1000), c.Get("key1"))
	assert.Equal(t, test_utils.IntPointer(2000), c.Get("key2"))
	assert.Equal(t, test_utils.IntPointer(3000), c.Get("key3"))
	assert.Equal(t, test_utils.IntPointer(4000), c.Get("key4"))
	assert.Equal(t, test_utils.IntPointer(5000), c.Get("key5"))
	assert.Nil(t, c.Get("key0"))
}

func testBlockingPreload(t *testing.T) {
	t.Parallel()

	loadCalled := make(chan bool, 1)
	loadComplete := make(chan bool, 1)

	c, err := New(Params[string, int]{
		Context: context.Background(),
		Log:     test_utils.Logger(),
		Name:    "testing_cache_blocking",
		LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
			loadCalled <- true
			time.Sleep(100 * time.Millisecond) // simulate slow load
			loadComplete <- true
			return map[string]*int{
				"key1": test_utils.IntPointer(1),
				"key2": test_utils.IntPointer(2),
			}, nil
		},
		Timeouts: Timeouts{
			ReloadInterval: 5 * time.Second,
			ReloadDelay:    0,
			Randomizer:     0,
		},
		NonBlockingPreload: false,
	})

	// In blocking mode, New should not return until load is complete
	// Check that load was called and completed before New returned
	select {
	case <-loadCalled:
		// Good, load was called
	case <-time.After(50 * time.Millisecond):
		t.Error("Load function was not called")
	}

	select {
	case <-loadComplete:
		// Good, load completed before New returned
	case <-time.After(100 * time.Millisecond):
		t.Error("Load did not complete before New returned in blocking mode")
	}

	assert.NoError(t, err)
	assert.NotNil(t, c)

	// Data should be available immediately after New returns
	assert.Equal(t, test_utils.IntPointer(1), c.Get("key1"))
	assert.Equal(t, test_utils.IntPointer(2), c.Get("key2"))
}

func testNonBlockingPreload(t *testing.T) {
	t.Parallel()

	loadCalled := make(chan bool, 1)
	loadComplete := make(chan bool, 1)

	c, err := New(Params[string, int]{
		Context: context.Background(),
		Log:     test_utils.Logger(),
		Name:    "testing_cache_nonblocking",
		LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
			loadCalled <- true
			time.Sleep(100 * time.Millisecond) // simulate slow load
			loadComplete <- true
			return map[string]*int{
				"key1": test_utils.IntPointer(1),
				"key2": test_utils.IntPointer(2),
			}, nil
		},
		Timeouts: Timeouts{
			ReloadInterval: 5 * time.Second,
			ReloadDelay:    0,
			Randomizer:     0,
		},
		NonBlockingPreload: true,
	})

	// In non-blocking mode, New should return immediately
	// even if load is still in progress
	assert.NoError(t, err)
	assert.NotNil(t, c)

	// Data might not be available immediately
	// Check that New returned before load completed
	select {
	case <-loadComplete:
		t.Error("Load completed before New returned in non-blocking mode")
	default:
		// Good, New returned before load completed
	}

	// Wait for load to complete
	select {
	case <-loadCalled:
		// Good, load was called
	case <-time.After(50 * time.Millisecond):
		t.Error("Load function was not called")
	}

	// Wait for load to complete
	select {
	case <-loadComplete:
		// Good, load completed
	case <-time.After(200 * time.Millisecond):
		t.Error("Load did not complete")
	}

	// Now data should be available
	time.Sleep(50 * time.Millisecond) // Give a bit more time for data to be stored
	assert.Equal(t, test_utils.IntPointer(1), c.Get("key1"))
	assert.Equal(t, test_utils.IntPointer(2), c.Get("key2"))
}

func testBlockingPreloadWithError(t *testing.T) {
	t.Parallel()

	testError := errors.New("load failed")

	c, err := New(Params[string, int]{
		Context: context.Background(),
		Log:     test_utils.Logger(),
		Name:    "testing_cache_blocking_error",
		LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
			return nil, testError
		},
		Timeouts: Timeouts{
			ReloadInterval: 5 * time.Second,
			ReloadDelay:    0,
			Randomizer:     0,
		},
		NonBlockingPreload: false,
	})

	// In blocking mode, when loadAllFunc fails, reload() returns the error
	// and New() returns early with the error and the cache instance (which was already created)
	assert.Error(t, err)
	assert.Equal(t, testError, err)
	assert.NotNil(t, c) // Cache is created before reload is called
}

func testOnReload(t *testing.T) {
	t.Parallel()

	t.Run("successfulInitialLoad", func(t *testing.T) {
		t.Parallel()

		var count atomic.Int32

		_, err := New(Params[string, int]{
			Context: context.Background(),
			Log:     test_utils.Logger(),
			Name:    "testing_cache_on_reload_initial",
			LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
				return map[string]*int{
					"key1": test_utils.IntPointer(1),
				}, nil
			},
			Timeouts: Timeouts{
				ReloadInterval: 5 * time.Second,
				ReloadDelay:    0,
				Randomizer:     0,
			},
			OnReload: func(entries map[string]*int) {
				count.Add(1)
			},
		})

		assert.NoError(t, err)
		assert.Eventually(t, func() bool {
			return count.Load() == 1
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("invalidateAllReload", func(t *testing.T) {
		t.Parallel()

		var count atomic.Int32
		valueMultiplier := 10

		c, err := New(Params[string, int]{
			Context: context.Background(),
			Log:     test_utils.Logger(),
			Name:    "testing_cache_on_reload_invalidate",
			LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
				return map[string]*int{
					"key1": test_utils.IntPointer(1 * valueMultiplier),
				}, nil
			},
			Timeouts: Timeouts{
				ReloadInterval: 5 * time.Second,
				ReloadDelay:    0,
				Randomizer:     0,
			},
			OnReload: func(entries map[string]*int) {
				count.Add(1)
			},
		})

		assert.NoError(t, err)
		assert.Eventually(t, func() bool {
			return count.Load() == 1
		}, time.Second, 10*time.Millisecond)

		valueMultiplier = 100
		c.InvalidateAll()

		assert.Eventually(t, func() bool {
			return count.Load() == 2
		}, time.Second, 10*time.Millisecond)
		assert.Equal(t, test_utils.IntPointer(100), c.Get("key1"))
	})

	t.Run("failedReload", func(t *testing.T) {
		t.Parallel()

		var count atomic.Int32
		loadCount := 0

		c, err := New(Params[string, int]{
			Context: context.Background(),
			Log:     test_utils.Logger(),
			Name:    "testing_cache_on_reload_failed",
			LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
				loadCount++
				if loadCount == 1 {
					return map[string]*int{
						"key1": test_utils.IntPointer(1),
					}, nil
				}
				return nil, errors.New("load failed")
			},
			Timeouts: Timeouts{
				ReloadInterval: 5 * time.Second,
				ReloadDelay:    0,
				Randomizer:     0,
			},
			OnReload: func(entries map[string]*int) {
				count.Add(1)
			},
		})

		assert.NoError(t, err)
		assert.Eventually(t, func() bool {
			return count.Load() == 1
		}, time.Second, 10*time.Millisecond)

		c.InvalidateAll()

		time.Sleep(1 * time.Second)

		assert.Equal(t, int32(1), count.Load())
		assert.Equal(t, test_utils.IntPointer(1), c.Get("key1"))
	})

	t.Run("skippedConcurrentReload", func(t *testing.T) {
		t.Parallel()

		var count atomic.Int32

		c, err := New(Params[string, int]{
			Context: context.Background(),
			Log:     test_utils.Logger(),
			Name:    "testing_cache_on_reload_skip",
			LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
				time.Sleep(500 * time.Millisecond)
				return map[string]*int{
					"key1": test_utils.IntPointer(1),
				}, nil
			},
			Timeouts: Timeouts{
				ReloadInterval: 5 * time.Second,
				ReloadDelay:    0,
				Randomizer:     0,
			},
			OnReload: func(entries map[string]*int) {
				count.Add(1)
			},
		})

		assert.NoError(t, err)
		assert.Eventually(t, func() bool {
			return count.Load() == 1
		}, time.Second, 10*time.Millisecond)

		go c.InvalidateAll()
		go c.InvalidateAll()

		assert.Eventually(t, func() bool {
			return count.Load() == 2
		}, 3*time.Second, 10*time.Millisecond)

		time.Sleep(500 * time.Millisecond)
		assert.Equal(t, int32(2), count.Load())
	})
}
