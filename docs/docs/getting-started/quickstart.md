# Quick Start

This guide walks through the core surface of ewrap in five minutes.

## Create errors

```go
import "github.com/hyp3rd/ewrap"

err := ewrap.New("database connection failed") // captures stack at the call site
```

The returned `*Error` implements the `error` interface. You can return it
anywhere a regular `error` is expected.

## Format with arguments — `Newf` is `%w`-aware

```go
err := ewrap.Newf("query %q failed: %w", q, ioErr)

errors.Is(err, ioErr) // true — %w preserves the cause chain
err.Error()           // "query \"...\" failed: <ioErr.Error()>"
```

If `format` doesn't contain `%w`, `Newf` behaves like `fmt.Sprintf` plus a
stack capture.

## Wrap existing errors

```go
if err := db.Ping(); err != nil {
    return ewrap.Wrap(err, "syncing replicas")
}
```

`Wrap` captures its own stack frames, so deep chains carry the full call
history rather than just the innermost site. `Wrap(nil, ...)` returns nil
so you can call it unconditionally if you prefer.

`Wrapf` is the formatted variant:

```go
return ewrap.Wrapf(err, "loading row %d for tenant %s", id, tenantID)
```

## Add structured context

```go
err := ewrap.New("payment authorization rejected",
    ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityError),
    ewrap.WithHTTPStatus(http.StatusBadGateway),
    ewrap.WithRetryable(true),
    ewrap.WithSafeMessage("payment authorization rejected"), // omits PII
    ewrap.WithRecoverySuggestion(&ewrap.RecoverySuggestion{
        Message:       "Inspect upstream provider's queue and retry after backoff.",
        Documentation: "https://runbooks.example.com/payments/timeout",
    }),
).
    WithMetadata("provider", "stripe").
    WithMetadata("attempt", 2)
```

Reserved fields (`ErrorContext`, `RecoverySuggestion`, `RetryInfo`) live in
typed fields, not the user metadata map — they have dedicated accessors and
can't be silently overwritten by a stray `WithMetadata` key.

## Read the structured fields back

```go
err.GetErrorContext()         // *ErrorContext (or nil)
err.Recovery()                // *RecoverySuggestion (or nil)
err.Retry()                   // *RetryInfo (or nil)
err.GetMetadata("attempt")    // (any, bool) for user metadata

ewrap.GetMetadataValue[int](err, "attempt") // generic, type-checked accessor
```

## Walk and classify the chain

```go
errors.Is(err, ioErr)
errors.As(err, &netErr)
errors.Unwrap(err)

ewrap.HTTPStatus(err)   // walks chain; 0 if no layer set one
ewrap.IsRetryable(err)  // true if any layer set Retryable, or stdlib Temporary()
err.SafeError()         // redacted variant for external sinks
```

## Format and log

```go
fmt.Printf("%+v\n", err)         // message + filtered stack (fmt.Formatter)
fmt.Printf("%v\n", err)          // message only
fmt.Printf("%q\n", err)          // quoted

slog.Error("payment failed", "err", err) // *Error implements slog.LogValuer
```

When you've attached a `Logger`, `(*Error).Log` emits a single structured
record with message, cause, stack, recovery, and all metadata:

```go
err := ewrap.New("boom", ewrap.WithLogger(logger))
err.Log()
```

## Aggregate with `ErrorGroup`

```go
pool := ewrap.NewErrorGroupPool(4)
eg := pool.Get()
defer eg.Release()

eg.Add(validate(req))
eg.Add(persist(req))

if err := eg.Join(); err != nil { // errors.Join semantics
    return err
}
```

`(*ErrorGroup).ToJSON()` and `ToYAML()` walk both `*Error` and standard
wrapped chains.

## Add a circuit breaker (opt-in)

```go
import "github.com/hyp3rd/ewrap/breaker"

cb := breaker.New("payments", 5, 30*time.Second)

if !cb.CanExecute() {
    return ewrap.New("payments breaker open", ewrap.WithRetryable(true))
}

if err := charge(req); err != nil {
    cb.RecordFailure()

    return ewrap.Wrap(err, "charging customer",
        ewrap.WithHTTPStatus(http.StatusBadGateway))
}

cb.RecordSuccess()
```

The breaker is in a sibling subpackage, so importing only `ewrap` doesn't
bring it into your binary.

## Where to go next

- [Error Creation](../features/error-creation.md) — `New`, `Newf`, options
- [Error Wrapping](../features/error-wrapping.md) — `Wrap`, `Wrapf`, chain semantics
- [Stack Traces](../features/stack-traces.md) — capture, filter, depth, caller skip
- [Operational Features](../features/operational.md) — HTTP / retry / safe message
- [`fmt.Formatter` & `slog`](../features/format-and-slog.md)
- [Circuit Breaker](../features/circuit-breaker.md)
- [API Reference](../api/overview.md)
