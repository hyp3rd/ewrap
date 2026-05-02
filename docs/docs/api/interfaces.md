# Interfaces

ewrap exposes only two interfaces in the root package, both deliberately
small. The breaker subpackage adds one more.

## `Logger`

```go
type Logger interface {
    Error(msg string, keysAndValues ...any)
    Debug(msg string, keysAndValues ...any)
    Info(msg string, keysAndValues ...any)
}
```

The contract:

- `keysAndValues` is the standard alternating key/value convention.
- Implementations must be **goroutine-safe**. `(*Error).Log` calls them
  synchronously from the calling goroutine, but multiple goroutines can
  hold a reference to the same logger via `WithLogger`.
- Returning errors is intentionally not part of the interface; logger
  failures are typically fatal-or-ignore decisions.

Used via `WithLogger(logger)`. See [Logging](../features/logging.md) for
adapter patterns.

## `Observer`

```go
type Observer interface {
    RecordError(message string)
}
```

Single method. `(*Error).Log` calls it synchronously before writing the
structured log record. Implementations must be goroutine-safe.

The interface is deliberately minimal — anything richer would mean ewrap
dictating a particular metric / tracing API. If you want the full
structured payload, attach a `Logger` instead.

Used via `WithObserver(obs)`. See [Observability](../features/observability.md).

## `breaker.Observer`

```go
package breaker

type Observer interface {
    RecordTransition(name string, from, to State)
}
```

Receives circuit-breaker transitions. Fired synchronously after the
breaker lock is released, so observer code runs without holding the
breaker mutex. Implementations must not invoke the breaker recursively.

Used via `breaker.NewWithObserver(...)` or `(*Breaker).SetObserver(obs)`.
See [Circuit Breaker](../features/circuit-breaker.md).

## Optional interfaces ewrap consumes

ewrap also recognises a few well-known interfaces from the stdlib and
ecosystem.

### `interface{ Unwrap() error }`

Standard error chain walker (Go 1.13+). `(*Error)` implements it; ewrap
helpers that walk chains (`HTTPStatus`, `IsRetryable`, serializer) call
`errors.Unwrap` so they cross both `*Error` and `fmt.Errorf("...:%w", ...)`
boundaries.

### `interface{ Unwrap() []error }`

Multi-cause variant introduced in Go 1.20 for `errors.Join`. ewrap's
`Newf` recognises it when extracting the cause from a `%w` format
containing multiple wrapped values; it stores the first cause as the
single `cause` field. `(*Error)` itself does not implement this — for
genuine multi-cause aggregation use `ErrorGroup`.

### `interface{ Temporary() bool }`

Stdlib transient-error marker (`net.Error`, `*net.OpError`, etc.).
`ewrap.IsRetryable(err)` falls through to this when no ewrap layer set
`WithRetryable`.

### `interface{ SafeError() string }`

Implemented by `(*Error)`. `(*Error).SafeError()` walks the chain and
defers to a cause's `SafeError` method when present, so PII redaction
composes cleanly.

### `fmt.Formatter`

Implemented by `(*Error)`. Supports `%s`, `%v`, `%q`, and `%+v`.

### `slog.LogValuer`

Implemented by `(*Error)`. Returns an `slog.GroupValue` containing
message, type, severity, request_id, cause, recovery, and metadata.
