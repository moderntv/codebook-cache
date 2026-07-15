# Codebook cache

Codebook cache is designed to store smaller sets of data (but not necessarily) where all items are always loaded at once and cannot be added into or removed from cache separately. Hence it can guarantee its key feature - **immutability**. This can be very handy when all items are being read as a complete set.

It provides high read performance by using `atomic.Value` to store the data map, so reads do not use mutexes.

Provided functions:

- `Get(ID)` — returns `*T` for key `K`, or `nil` if not found
- `GetAll()` — returns `map[K]*T` (do not modify; it is shared with the cache)
- `InvalidateAll()` — triggers a reload (immediate or delayed when `ReloadDelay` > 0)

The cache uses Go generics: key type `K` must be `comparable`, value type `T` is arbitrary.

## Lifecycle

`New(...)` loads all items via `LoadAllFunc`. With `NonBlockingPreload == false` (default), `New` **blocks** until the first load finishes—on failure it returns an error and no cache is created. With `NonBlockingPreload == true`, `New` returns immediately and the first load runs in the background; do not call `Get` or `GetAll` before that load completes. After the first successful load, the cache holds a valid set of items (possibly empty).

If `ReloadInterval` > 0, the cache periodically reloads. Each interval is randomized by `Randomizer` (range `[0, 1]`, e.g. `0.1` = ±10%). On success, data are replaced; on failure, existing data are kept and a warning is logged. The next reload is always scheduled from `ReloadInterval` (randomized).

## Caveats

- Cached data may not reflect the current state of the underlying storage.
- Only full reload is supported; you cannot add, remove, or update single items.
- Set `Timeouts` to match your needs: e.g. frequent NATS-driven reloads for fresher data, or longer intervals for lower load on the source.

## Timeouts

- **ReloadInterval** – Period between periodic reloads; each is randomized by `Randomizer`. Use `0` to disable periodic reload (invalidations and `InvalidateAll` still run).
- **ReloadDelay** – After a reload, further `InvalidateAll` calls are deferred for this duration and coalesced into one reload; outside this window, reload runs immediately. Must be ≤ `ReloadInterval`. Use `0` to disable (every invalidation triggers an immediate reload).
- **Randomizer** – `[0, 1]`. `0` = no jitter; `0.1` = ±10%. Applied to `ReloadInterval` to reduce thundering herd.

## Params

- **Context** – `context.Context` for shutdown and `LoadAllFunc` calls. Required.
- **Log** – `zerolog.Logger` for cache logs. Required.
- **MetricsRegistry** – `*cadre_metrics.Registry`; if set, Prometheus metrics are registered. Optional.
- **Invalidations** – `*Invalidations` for NATS-driven invalidation. If nil, NATS is disabled; only manual `InvalidateAll` and periodic reload (when `ReloadInterval` > 0) apply. Optional.
- **Name** – Cache name used in logs and metrics. Required.
- **LoadAllFunc** – `func(ctx context.Context) (map[K]*T, error)` that loads the full dataset. Required. Called on initial load, periodic reload, and after `InvalidateAll`.
- **Timeouts** – Reload interval, delay, and randomization. Required; see `Timeouts` for rules (e.g. `ReloadDelay` ≤ `ReloadInterval`).
- **NonBlockingPreload** – If `true`, `New` returns before the first load completes; initial load runs in a goroutine. If `false`, `New` blocks until the first load succeeds (or fails and returns an error).
- **MemsizeEnabled** – If `true`, memory usage of cached entries is estimated and exposed via the `memory_usage` metric.
- **OnReload** – Optional callback invoked asynchronously after each successful reload (including initial preload). Not called when reload is skipped or `LoadAllFunc` fails. Use `Get` / `GetAll` inside the callback to read fresh data.

## Metrics

When `MetricsRegistry` is set, these Prometheus metrics are registered (subsystem `codebook_cache`, label `name`):

| Metric                        | Type    | Description                                                |
| ----------------------------- | ------- | ---------------------------------------------------------- |
| `items_count`                 | Gauge   | Number of cached items                                     |
| `load_count`                  | Counter | Load attempts (success or failure)                         |
| `received_nats_invalidations` | Counter | NATS invalidation messages received                        |
| `memory_usage`                | Gauge   | Estimated size of entries in bytes (`MemsizeEnabled` only) |
| `reads_count`                 | Counter | `Get` / `GetAll` calls                                     |

## NATS invalidations

If `Invalidations` is set, the cache subscribes to NATS and calls `InvalidateAll` on each message. `Invalidations`:

- **Nats** – `*nats.Conn`. Required when `Invalidations` is non-nil.
- **Prefix** – Prepended to each subject.
- **Messages** – `map[string]proto.Message`: key = subject suffix, value = proto used to unmarshal. At least one entry required.

Each received message is unmarshalled and triggers `InvalidateAll` (subject to `ReloadDelay` when > 0).

---

## Usage examples

### Minimal example (in-memory, no NATS)

```go
package main

import (
	"context"
	"time"

	codebook "github.com/moderntv/codebook-cache"
	"github.com/rs/zerolog"
)

func main() {
	ctx := context.Background()
	log := zerolog.Nop()

	cache, err := codebook.New(codebook.Params[string, string]{
		Context:  ctx,
		Log:      log,
		Name:     "my-codebook",
		LoadAllFunc: func(ctx context.Context) (map[string]*string, error) {
			m := map[string]*string{"a": ptr("A"), "b": ptr("B")}
			return m, nil
		},
		Timeouts: codebook.Timeouts{
			ReloadInterval: 5 * time.Minute,
			ReloadDelay:   10 * time.Second,
			Randomizer:    0.1,
		},
	})
	if err != nil {
		panic(err)
	}

	_ = cache.Get("a")    // returns *string "A"
	_ = cache.Get("x")    // returns nil
	_ = cache.GetAll()    // returns map[string]*string
	cache.InvalidateAll() // triggers reload (respecting ReloadDelay if set)
}

func ptr(s string) *string { return &s }
```

### Practical example: repository with DB and NATS

Typical pattern: a repository holds a `*codebook.Cache` and delegates `Get`/`GetAll` to it. The cache is created with `LoadAllFunc` that reads from DB and `Invalidations` that subscribes to NATS.

```go
package country

import (
	"context"
	"time"

	cadre_metrics "github.com/moderntv/cadre/metrics"
	codebook "github.com/moderntv/codebook-cache"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	inv "your.org/yourproject/invalidation" // SubjectPrefix, SubjectCountry, CountryInvalidation
)

type Country struct {
	ID   string
	Name string
	// ...
}

type Repository struct {
	cache *codebook.Cache[string, Country]
}

func NewRepository(ctx context.Context, log zerolog.Logger, db *gorm.DB, nc *nats.Conn, metricsRegistry *cadre_metrics.Registry) (*Repository, error) {
	params := codebook.Params[string, Country]{
		Context:         ctx,
		Log:             log,
		MetricsRegistry: metricsRegistry,
		Invalidations: &codebook.Invalidations{
			Nats:   nc,
			Prefix: inv.SubjectPrefix,
			Messages: map[string]proto.Message{
				inv.SubjectCountry: &inv.CountryInvalidation{},
			},
		},
		Name: "country",
		LoadAllFunc: func(ctx context.Context) (map[string]*Country, error) {
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			var rows []*Country
			if err := db.WithContext(ctx).Find(&rows).Error; err != nil {
				return nil, err
			}
			out := make(map[string]*Country)
			for _, c := range rows {
				out[c.ID] = c
			}
			return out, nil
		},
		Timeouts: codebook.Timeouts{
			ReloadInterval: 10 * time.Minute,
			ReloadDelay:    30 * time.Second,
			Randomizer:     0.1,
		},
		MemsizeEnabled: true,
	}

	c, err := codebook.New(params)
	if err != nil {
		return nil, err
	}
	return &Repository{cache: c}, nil
}

func (r *Repository) Country(id string) *Country {
	return r.cache.Get(id)
}

func (r *Repository) All() map[string]*Country {
	return r.cache.GetAll()
}
```

- **LoadAllFunc**: loads all countries from DB into `map[string]*Country` (key = `Country.ID`).
- **Invalidations**: subscribes to `Prefix + SubjectCountry`; on each `CountryInvalidation` message, `InvalidateAll` is called (and aggregated if `ReloadDelay` > 0).
- **Timeouts**: periodic reload every ~10 minutes; invalidation bursts within 30s are coalesced into one reload.
- **MemsizeEnabled**: enables the `memory_usage` metric for this cache.
