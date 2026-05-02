# Metadata

ewrap separates **user metadata** (a string-keyed map you control) from
**reserved typed fields** (`ErrorContext`, `RecoverySuggestion`,
`RetryInfo`). Each lives in its own slot so a stray `WithMetadata` key
can't silently overwrite the structured fields.

## User metadata

Use `WithMetadata` to attach arbitrary key/value data:

```go
err := ewrap.New("checkout failed").
    WithMetadata("order_id", orderID).
    WithMetadata("attempt", 2).
    WithMetadata("provider", "stripe")
```

Read it back with `GetMetadata`:

```go
val, ok := err.GetMetadata("order_id")
```

Or with the generic, type-checked variant:

```go
attempt, ok := ewrap.GetMetadataValue[int](err, "attempt")
provider, ok := ewrap.GetMetadataValue[string](err, "provider")
```

`GetMetadataValue` returns the zero value of `T` and `false` if the key is
missing or the stored value isn't of type `T`.

### Lazy allocation

The metadata map is **not allocated until the first write**. An error that
never gets metadata pays nothing for the field beyond the nil slice header.

### Concurrent reads and writes

`WithMetadata`, `GetMetadata`, and `GetMetadataValue` are protected by a
`sync.RWMutex`, so concurrent use across goroutines is safe.

## Reserved typed fields

These slots have dedicated options and accessors. They never appear in the
user metadata map.

### `ErrorContext`

Captured via `WithContext(ctx, type, severity)`:

```go
err := ewrap.New("payment failed",
    ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityError))

ec := err.GetErrorContext()
// ec.Type, ec.Severity, ec.RequestID, ec.User, ec.Operation, ec.Component,
// ec.Environment, ec.Timestamp, ec.File, ec.Line, ec.Data
```

`WithContext` reads `request_id`, `user`, `operation`, and `component`
out of the supplied `context.Context` if those keys are present.

You can also attach a pre-built `ErrorContext` after construction:

```go
err.WithContext(&ewrap.ErrorContext{Type: ewrap.ErrorTypeNetwork})
```

### `RecoverySuggestion`

```go
err := ewrap.New("DB unreachable",
    ewrap.WithRecoverySuggestion(&ewrap.RecoverySuggestion{
        Message:       "Check connectivity and pool sizing.",
        Actions:       []string{"reset pool", "verify network"},
        Documentation: "https://runbooks.example.com/db",
    }))

rs := err.Recovery()
```

When the error is logged via `(*Error).Log`, the recovery suggestion is
emitted as `recovery_message`, `recovery_actions`, and
`recovery_documentation` fields.

### `RetryInfo`

```go
err := ewrap.New("upstream timeout",
    ewrap.WithRetry(3, 5*time.Second))

ri := err.Retry() // *RetryInfo, or nil if not set
err.CanRetry()    // checks attempts vs ShouldRetry predicate
err.IncrementRetry()
```

Customise the retry predicate:

```go
err := ewrap.New("rate limited",
    ewrap.WithRetry(5, 2*time.Second,
        ewrap.WithRetryShould(func(e error) bool {
            return ewrap.IsRetryable(e)
        })))
```

The default predicate returns `true` unless the error's `ErrorContext.Type`
is `ErrorTypeValidation`.

## Why typed fields?

The previous design stored these under reserved string keys
(`"error_context"`, `"recovery_suggestion"`, `"retry_info"`) in the same
map as user metadata. That made it possible — and easy — to silently
corrupt them with a stray `WithMetadata("error_context", ...)` call.

Lifting them to typed fields:

- Eliminates that footgun.
- Makes the API self-documenting — the type system shows exactly what a
  recovery suggestion looks like.
- Avoids the runtime cost of a type assertion on every read.

## Inheritance through `Wrap`

When `Wrap` is given a `*Error`, the wrapper inherits **all** typed fields
plus a clone of the metadata map:

```go
inner := ewrap.New("DB error",
    ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical),
    ewrap.WithRetryable(true)).
    WithMetadata("query", q)

outer := ewrap.Wrap(inner, "loading user")

outer.GetErrorContext()           // inherited
outer.Retryable()                 // inherited
outer.GetMetadata("query")        // inherited via maps.Clone
```

Pass an option to `Wrap` to override:

```go
outer := ewrap.Wrap(inner, "loading user",
    ewrap.WithContext(ctx, ewrap.ErrorTypeNotFound, ewrap.SeverityWarning))
```

## Cheat sheet

| Concept | Set with | Read with |
| --- | --- | --- |
| User metadata (untyped) | `WithMetadata(key, value)` | `GetMetadata(key)` / `GetMetadataValue[T]` |
| Error context | `WithContext(ctx, type, sev)` option / `(*Error).WithContext(ec)` method | `GetErrorContext()` |
| Recovery guidance | `WithRecoverySuggestion(rs)` | `Recovery()` |
| Retry info | `WithRetry(max, delay, opts...)` | `Retry()` / `CanRetry()` / `IncrementRetry()` |
| HTTP status | `WithHTTPStatus(code)` | `ewrap.HTTPStatus(err)` |
| Retryable flag | `WithRetryable(bool)` | `(*Error).Retryable()` / `ewrap.IsRetryable(err)` |
| Safe message | `WithSafeMessage(s)` | `(*Error).SafeError()` |
