# Codebook cache

Codebook cache is designed to store smaller sets of data (but not necessarily) where all items are always loaded at once and cannot be added into or removed from cache separately. Hence it can guarantee its key feature - **immutability**. This can be very handy when all items are being read as a complete set.

It also provides best performance for huge ammount of readings because it does not use mutexes to access the data, just `atomic.Value` which store a map of values for given keys.

Provided functions:

-   `Get(ID)` for given `ID` of type `K` returns pointer to value of type `T` (if exists) or `nil` (not exists)
-   `GetAll()` returns map of all items in cache in format `map[K]*T`
-   `InvalidateAll()` triggers items reload (immediate or delayed depending on `Timeouts.ReloadDelay` value)

Implementation of cache uses Go generics, so it can be instantiated for keys which must be `comparable` (referenced as `K`) and `any` items value (referenced as `T`).

## Lifecycle

When codebook cache is created by calling `New(...)` function, it tries to immediately load all items; if it fails, cache is not being created and ends with an error. This ensures that if cache is once successfully created, it **always holds and provides valid set of items** (can be empty though).

According to given `Timeouts`, data can be periodically reloaded. When reload successfully loads all items, data are replaced in cache. When reload fails, data in cache are not changed and warning is being logged. In both cases next periodic reload is planned according to `Timeouts.ReloadInterval` value.

Due to possible performance issues or heavy-load spikes, reload interval can be ranomized by setting `Timeouts.Randomizer` to value between (0, 1>. Each periodic reload interval is then being randomized.

## Disadvantages

As almost every cache, keep in mind that data stored in cache does not need to exist or be valid in original data storage.

It is important properly set `Timeouts` values according to desired behavior. It can be easily set as cache with _fresh_ data being reloaded (via NATS invalidation messages) immediatelly after original data changes OR serve as very efficient layer providing possibly expired / invalid data with very low original storage usage.

## Timeouts

TODO

## Params

TODO

## Metrics

TODO

## NATS invalidations

TODO
