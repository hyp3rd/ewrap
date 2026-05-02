# Error Creation

ewrap exposes four constructors. They all capture a stack trace at the call
site (configurable; see [Stack Traces](stack-traces.md)) and return a
`*Error` that satisfies the `error` interface.

| Constructor | Use when |
| --- | --- |
| `New(msg, opts...)` | Plain error with a static message |
| `Newf(format, args...)` | Formatted message; `%w` is honoured |
| `Wrap(err, msg, opts...)` | Add a layer to an existing error |
| `Wrapf(err, format, args...)` | Same, with a formatted message |

`Wrap` / `Wrapf` are nil-safe: `Wrap(nil, "...")` returns `nil`, so you can
call them unconditionally.

## `New` — static message

```go
err := ewrap.New("user not found")
```

`New` returns a `*Error` with the message stored verbatim. The metadata map
is **not** allocated until you call `WithMetadata` — most errors never need
one and pay no cost for it.

## `Newf` — formatted, `%w`-aware

```go
err := ewrap.Newf("user %d not found", id)            // simple format
err := ewrap.Newf("query %q failed: %w", q, ioErr)    // wraps ioErr
```

When `format` contains `%w`, `Newf` extracts the wrapped argument as the
cause so `errors.Is(err, ioErr)` returns true. The full formatted text
becomes the error's `.Error()` output (matching `fmt.Errorf` semantics).

If `format` contains multiple `%w` verbs, the first wrapped error becomes
the cause; the others appear in the rendered message.

## `Wrap` — layer on an existing error

```go
if err := db.Ping(); err != nil {
    return ewrap.Wrap(err, "syncing replicas")
}
```

Every `Wrap` captures **its own** stack frames, so deep chains carry the
full call history rather than just the innermost site. When the inner error
is itself a `*Error`, the wrapper inherits its metadata, error context,
recovery suggestion, retry info, observer, and logger.

## `Wrapf` — formatted wrap

```go
return ewrap.Wrapf(err, "loading row %d for tenant %s", rowID, tenantID)
```

## Options at construction time

`New`, `Wrap`, and `WrapSkip` accept variadic `Option`s:

```go
err := ewrap.New("payment authorization rejected",
    ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityError),
    ewrap.WithHTTPStatus(http.StatusBadGateway),
    ewrap.WithRetryable(true),
    ewrap.WithSafeMessage("payment authorization rejected"),
    ewrap.WithRecoverySuggestion(&ewrap.RecoverySuggestion{
        Message:       "Inspect provider's queue and retry after backoff.",
        Documentation: "https://runbooks.example.com/payments/timeout",
    }),
    ewrap.WithRetry(3, 5*time.Second),
    ewrap.WithLogger(logger),
    ewrap.WithObserver(observer),
    ewrap.WithStackDepth(16), // override default capture depth
)
```

A full list lives in [API Reference → Options](../api/options.md).

## Chained metadata

`(*Error).WithMetadata` returns the same instance so calls can chain after
construction. The map is allocated lazily on the first call.

```go
err := ewrap.New("operation failed").
    WithMetadata("query", "SELECT * FROM users").
    WithMetadata("retry_count", 3).
    WithMetadata("connection_pool_size", 10)
```

For typed reads, use the generic accessor:

```go
count, ok := ewrap.GetMetadataValue[int](err, "retry_count")
```

## Domain-specific factories

Wrap construction in your own factories to enforce conventions:

```go
func ErrUnderage(ctx context.Context, age int) *ewrap.Error {
    return ewrap.NewSkip(1, "user is underage",
        ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError),
        ewrap.WithHTTPStatus(http.StatusUnprocessableEntity),
    ).
        WithMetadata("minimum_age", 18).
        WithMetadata("provided_age", age)
}
```

`NewSkip(skip, ...)` advances the captured stack by `skip` frames, so the
trace starts at the caller of your factory rather than inside it. The
companion `WrapSkip(skip, err, ...)` does the same for wraps.

## Thread safety

All constructors and `*Error` accessors are safe for concurrent use. The
metadata map is guarded by an `RWMutex`; everything else is set once at
construction and never mutated, so reads (including cached `Error()` and
`Stack()`) are lock-free after the first call.

## Performance notes

- The metadata map is allocated on first write; errors with no metadata pay
  for one allocation (the `*Error` struct) plus the stack PCs slice.
- `Error()` and `Stack()` cache their formatted output via `sync.Once`.
- `(*Error).Format` and `LogValue` reuse those caches — `fmt.Printf("%v", err)`
  and `slog.Error(..., "err", err)` are both cheap after the first format.
