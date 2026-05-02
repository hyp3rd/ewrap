# Performance Optimization

ewrap is designed for the hot path. This page collates the design choices
and the knobs you control.

## Numbers at a glance

`go test -bench=. -benchmem ./test/...` (Apple Silicon, Go 1.25+):

| Benchmark | ns/op | B/op | allocs |
| --- | ---: | ---: | ---: |
| `BenchmarkNew/Simple` | 1622 | 496 | 2 |
| `BenchmarkNew/WithContext` | 5273 | 968 | 6 |
| `BenchmarkWrap/Simple` | 3828 | 504 | 3 |
| `BenchmarkWrap/NestedWraps` | 11433 | 1512 | 9 |
| `BenchmarkErrorGroup/AddErrors` | ~22000 | 752 | 24 |
| `BenchmarkFormatting/ToJSON` | 16947 | 2941 | 14 |
| `BenchmarkFormatting/ToYAML` | 247276 | 40472 | 115 |
| `BenchmarkCircuitBreaker/RecordFailure` | 33 | 0 | 0 |
| `BenchmarkMetadataOperations/AddMetadata` | 4895 | 852 | 9 |
| `BenchmarkMetadataOperations/GetMetadata` | 9 | 0 | 0 |
| `BenchmarkStackTrace/CaptureStack` | 858 | 256 | 1 |
| `BenchmarkStackTrace/FormatStack` (cached) | **1.71** | 0 | **0** |

## Where the allocations come from

A bare `ewrap.New("...")`:

1. `*Error` struct — one allocation.
2. `runtime.Callers` PC slice (32 entries) — one allocation.

That's it — two allocations, ~500 bytes. The metadata map is **lazy**:
allocated on the first `WithMetadata`, never if you don't call it.

`Wrap` adds a third allocation when the inner error is a `*Error`
(cloning its metadata map via `maps.Clone`).

## Caching `Error()` and `Stack()`

Both methods are guarded by `sync.Once`:

```go
e.errOnce.Do(func() {
    e.errStr = ... // single computation
})
return e.errStr
```

After the first call, every subsequent `Error()` / `Stack()` (and any
verb that uses them — `%v`, `%+v`, `LogValue`) returns the cached string
with zero allocations.

If you log the same error multiple times — common in retry / fan-out
flows — this is a substantial win.

## Tuning stack capture

| What | Default | How to change |
| --- | --- | --- |
| Capture depth | 32 frames | `WithStackDepth(n)` — pass 0 to disable |
| Caller skip | starts at user code | `NewSkip(skip, ...)` / `WrapSkip(skip, ...)` for helpers |
| Frame filter | hides `runtime.*` and ewrap internals | not configurable; fork if needed |

Disabling capture entirely on a hot path:

```go
ewrap.New("rate limited", ewrap.WithStackDepth(0))
```

This trades the second allocation (PCs slice) for zero stack output — the
right call when you know the error will be classified-and-returned, not
debugged.

## `ErrorGroup` pooling

For high-throughput aggregation (validation passes, batch operations,
fan-out), reuse `ErrorGroup` via a pool:

```go
pool := ewrap.NewErrorGroupPool(8) // initial slice capacity per group

eg := pool.Get()
defer eg.Release()
```

`Release()` clears the slice (preserving capacity) and returns the group
to the pool. The pool is goroutine-safe (`sync.Pool`).

A warm pool eliminates the slice header allocation for new groups; the
benchmark above (24 allocs for 10 errors with `AddErrors`) drops to 14
when running with a pool.

## Lazy metadata

```go
err := ewrap.New("boom") // err.metadata == nil
err.WithMetadata("k", "v") // map is allocated here
```

Errors that never carry metadata pay zero for the map. The metadata map
is also cloned (not shared) when `Wrap` inherits from an inner `*Error`,
so wrapper writes never mutate the inner.

## JSON vs YAML

JSON via `goccy/go-json` is ~14× faster than YAML for the same payload:

| | ns/op | allocs |
| --- | ---: | ---: |
| `Error.ToJSON` | 16947 | 14 |
| `Error.ToYAML` | 247276 | 115 |

If you control the format, prefer JSON. Strip stacks for high-volume
sinks (`WithStackTrace(false)`).

## Concurrency

| Type | Locking |
| --- | --- |
| `*Error` (read paths after construction) | lock-free (cached, immutable) |
| `*Error.WithMetadata`, `GetMetadata` | `sync.RWMutex` |
| `*Error.IncrementRetry` | `sync.RWMutex` (write) |
| `ErrorGroup.Add`, `Errors`, etc. | `sync.RWMutex` |
| `breaker.Breaker` ops | single `sync.Mutex` |

Hot-path reads (`Error()`, `Stack()`, `LogValue`, `Format`) don't take the
mutex — they read fields set at construction or cached results.

## Hot-path checklist

- ☑ Pool `ErrorGroup` instances if you allocate many per request.
- ☑ Set `WithStackDepth(0)` on classified-and-returned errors that
  won't be debugged.
- ☑ Reuse a single `Logger` and `Observer` across the request — both
  are inherited by `Wrap`.
- ☑ Prefer `slog.LogValuer` (no adapter) over `(*Error).Log` when
  you're already inside an `slog` handler.
- ☑ Use `ewrap.GetMetadataValue[T]` instead of `GetMetadata` followed
  by a type assertion.
- ☐ Don't reallocate `RecoverySuggestion` per call — define them as
  `var`s and reuse.
- ☐ Don't log + wrap. Wrap and let the eventual handler log.

## When ewrap is the wrong tool

- **Inner loops processing millions of items:** errors should be
  exceptional. If you're allocating one per iteration, restructure the
  algorithm so failure is rare or signalled differently (sentinel
  variable, skip count, channel signal).
- **CGo error wrappers:** stack capture across the cgo boundary is
  pointless. Use `ewrap.New(msg, ewrap.WithStackDepth(0))` and pass the
  C errno via metadata.
- **Single-binary CLI tools:** the structured fields are overkill —
  `fmt.Errorf` with `%w` is plenty.
