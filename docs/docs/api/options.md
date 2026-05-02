# Options

`type Option func(*Error)` — variadic configuration for `New` and
`Wrap`. Options run during construction; after `New`/`Wrap` returns the
`*Error` is effectively immutable except for chained `WithMetadata` /
`WithContext` calls and `IncrementRetry`.

This page is the canonical reference for every option exported from the
root package.

## `WithLogger(log Logger) Option`

Attach a `Logger` consulted by `(*Error).Log`. Inherited by `Wrap` when
the inner error is a `*Error`.

```go
err := ewrap.New("boom", ewrap.WithLogger(logger))
err.Log() // calls logger.Error("error occurred", ...kv)
```

Setting `WithLogger(nil)` is allowed and silently no-ops the logger
(typical pattern: pass nil in unit tests).

## `WithObserver(obs Observer) Option`

Attach an `Observer` whose `RecordError(msg string)` is called from
`(*Error).Log`. Inherited by `Wrap` when the inner error is a `*Error`.

```go
err := ewrap.New("boom", ewrap.WithObserver(metrics))
err.Log() // metrics.RecordError("boom")
```

## `WithStackDepth(depth int) Option`

Override the default stack capture depth (32). Pass `0` to disable
capture entirely.

```go
ewrap.New("boom", ewrap.WithStackDepth(8))   // shallower
ewrap.New("boom", ewrap.WithStackDepth(0))   // no stack
ewrap.New("boom", ewrap.WithStackDepth(128)) // deeper
```

## `WithContext(ctx context.Context, type ErrorType, sev Severity) Option`

Build an `ErrorContext` from the supplied `context.Context` and attach
it. The option reads `request_id`, `user`, `operation`, and `component`
keys out of `ctx` if present. The resulting `ErrorContext` includes the
file/line of the calling `New`/`Wrap` (via `runtime.Caller`).

```go
err := ewrap.New("boom",
    ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError))
```

For attaching a pre-built `ErrorContext`, use the method form:

```go
err.WithContext(&ewrap.ErrorContext{Type: ewrap.ErrorTypeNetwork})
```

## `WithRecoverySuggestion(rs *RecoverySuggestion) Option`

Attach actionable recovery guidance. Read back via `(*Error).Recovery()`,
emitted as `recovery_message`, `recovery_actions`, and
`recovery_documentation` fields by `Log`.

```go
ewrap.New("DB unreachable",
    ewrap.WithRecoverySuggestion(&ewrap.RecoverySuggestion{
        Message:       "Verify pool sizing and credentials.",
        Actions:       []string{"reset pool", "rotate creds"},
        Documentation: "https://runbooks.example.com/db",
    }))
```

## `WithRetry(maxAttempts int, delay time.Duration, opts ...RetryOption) Option`

Attach a retry **policy** (max attempts, delay, predicate). Use with
`(*Error).CanRetry()` and `(*Error).IncrementRetry()` to drive a retry
loop, or just inspect `(*Error).Retry()` for the raw `*RetryInfo`.

```go
err := ewrap.New("upstream timeout", ewrap.WithRetry(3, 5*time.Second))

for err.CanRetry() {
    if doErr := upstream(); doErr == nil {
        break
    }
    err.IncrementRetry()
    time.Sleep(err.Retry().Delay)
}
```

### `WithRetryShould(fn func(error) bool) RetryOption`

Customise the predicate consulted by `CanRetry`:

```go
ewrap.WithRetry(5, 2*time.Second,
    ewrap.WithRetryShould(func(e error) bool { return ewrap.IsRetryable(e) }))
```

The default predicate returns `true` unless `ErrorContext.Type` is
`ErrorTypeValidation`.

## `WithHTTPStatus(status int) Option`

Tag the error with an HTTP status code. Use `net/http` constants for
clarity. `ewrap.HTTPStatus(err)` walks the chain and returns the first
non-zero status.

```go
ewrap.New("upstream 502",
    ewrap.WithHTTPStatus(http.StatusBadGateway))
```

## `WithRetryable(retryable bool) Option`

Three-state retry classification (unset / true / false). Read with
`(*Error).Retryable() (value, set bool)` or `ewrap.IsRetryable(err)`.

```go
ewrap.New("rate limited", ewrap.WithRetryable(true))
ewrap.New("invalid credentials", ewrap.WithRetryable(false))
```

`IsRetryable` falls through to the stdlib `interface{ Temporary() bool }`
when no ewrap layer set the flag, so `net.OpError` and similar work
out of the box.

## `WithSafeMessage(safe string) Option`

Attach a redacted variant returned by `(*Error).SafeError()`. Each layer
contributes either its safe message (if set) or its raw `msg`; standard
wrapped errors without a `SafeError` method are included verbatim.

```go
ewrap.New("user 'alice@example.com' rejected",
    ewrap.WithSafeMessage("user [redacted] rejected"))
```

## Inheritance through `Wrap`

When the inner error is a `*Error`, `Wrap` inherits **all** option-set
state on the inner: logger, observer, stack-depth-derived stack, error
context, recovery suggestion, retry info, HTTP status, retryable flag,
and a clone of the metadata map.

Any option passed to `Wrap` overrides the inherited value:

```go
inner := ewrap.New("boom", ewrap.WithHTTPStatus(http.StatusBadGateway))
outer := ewrap.Wrap(inner, "in handler",
    ewrap.WithHTTPStatus(http.StatusInternalServerError))

ewrap.HTTPStatus(outer) // 500 — outer wins
ewrap.HTTPStatus(inner) // 502 — inner unchanged
```

## RetryOption

A separate option type for `WithRetry`'s sub-options:

```go
type RetryOption func(*RetryInfo)

func WithRetryShould(fn func(error) bool) RetryOption
```

## FormatOption

`(*Error).ToJSON` and `(*Error).ToYAML` accept their own option type:

```go
type FormatOption func(*ErrorOutput)

func WithTimestampFormat(format string) FormatOption
func WithStackTrace(include bool) FormatOption
```

See [Serialization](../features/serialization.md).
