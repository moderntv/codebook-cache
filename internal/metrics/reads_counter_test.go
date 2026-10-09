package metrics

import (
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moderntv/codebook-cache/internal/test_utils"
)

func TestReadsCounterSumsConcurrentReads(t *testing.T) {
	c := newReadsCounter(prometheus.NewDesc("reads", "", nil, nil))

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10000; i++ {
				c.Inc()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, uint64(8*10000), c.Value())
	assert.Equal(t, float64(8*10000), testutil.ToFloat64(c))
}

func TestReadsCounterKeepsMetricOutput(t *testing.T) {
	registry := test_utils.MetricsWithNamespace()
	m, err := New("channels", registry)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		m.ReadsCount.Inc()
	}

	expected := `
# HELP appapi_test_codebook_cache_reads_count Total number of cache reads
# TYPE appapi_test_codebook_cache_reads_count counter
appapi_test_codebook_cache_reads_count{name="channels"} 5
`
	err = testutil.GatherAndCompare(
		registry.GetPrometheusRegistry(),
		strings.NewReader(expected),
		"appapi_test_codebook_cache_reads_count",
	)
	assert.NoError(t, err)
}

func TestReadsCounterSpreadsGoroutines(t *testing.T) {
	c := newReadsCounter(prometheus.NewDesc("reads", "", nil, nil))

	// the goroutines stay alive together, so each of them has its own stack
	release := make(chan struct{})
	var counted, done sync.WaitGroup
	for g := 0; g < 64; g++ {
		counted.Add(1)
		done.Add(1)
		go func() {
			defer done.Done()
			c.Inc()
			counted.Done()
			<-release
		}()
	}
	counted.Wait()
	close(release)
	done.Wait()

	used := 0
	for i := range c.slots {
		if c.slots[i].n.Load() > 0 {
			used++
		}
	}
	assert.GreaterOrEqual(t, used, 8)
}

func TestReadsCounterDoesNotAllocate(t *testing.T) {
	c := newReadsCounter(prometheus.NewDesc("reads", "", nil, nil))

	allocs := testing.AllocsPerRun(100, c.Inc)
	assert.Zero(t, allocs)
}

func BenchmarkReadsCounter(b *testing.B) {
	plain := prometheus.NewCounter(prometheus.CounterOpts{Name: "reads"})
	slotted := newReadsCounter(plain.Desc())

	b.Run("plain", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			plain.Inc()
		}
	})
	b.Run("slotted", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slotted.Inc()
		}
	})
	b.Run("plainParallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				plain.Inc()
			}
		})
	})
	b.Run("slottedParallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				slotted.Inc()
			}
		})
	})
}
