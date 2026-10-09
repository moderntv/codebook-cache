package codebook

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/rs/zerolog"

	"github.com/moderntv/codebook-cache/internal/test_utils"
)

func BenchmarkGet(b *testing.B) {
	keys := make([]string, 700)
	entries := make(map[string]*int, len(keys))
	for i := range keys {
		keys[i] = "channel" + strconv.Itoa(i)
		entries[keys[i]] = test_utils.IntPointer(i)
	}

	c, err := New(Params[string, int]{
		Context:         context.Background(),
		Log:             zerolog.Nop(),
		MetricsRegistry: test_utils.MetricsWithNamespace(),
		Name:            "bench_get",
		LoadAllFunc: func(ctx context.Context) (map[string]*int, error) {
			return entries, nil
		},
	})
	if err != nil {
		b.Fatal(err)
	}

	b.Run("serial", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			c.Get(keys[i%len(keys)])
		}
	})
	b.Run("parallel", func(b *testing.B) {
		var next atomic.Int64
		b.RunParallel(func(pb *testing.PB) {
			i := int(next.Add(97))
			for pb.Next() {
				i++
				c.Get(keys[i%len(keys)])
			}
		})
	})
}
