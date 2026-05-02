# Context Integration

`context.Context` flows through every modern Go service. ewrap weaves it
into errors via the `WithContext` option, lifts request-scoped values out
automatically, and exposes them via the typed `ErrorContext` accessor.

## What `WithContext` does

```go
err := ewrap.New("payment failed",
    ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityError))
```

This builds an `ErrorContext` and attaches it as a typed field on the
`*Error`. The option:

1. Records the supplied `ErrorType` and `Severity`.
2. Captures the file/line of the calling `New`/`Wrap` (via
   `runtime.Caller`).
3. Sets `Environment` from `APP_ENV` (or `"development"` by default).
4. Reads four well-known keys out of `ctx`:
   - `request_id` → `ErrorContext.RequestID`
   - `user`       → `ErrorContext.User`
   - `operation`  → `ErrorContext.Operation`
   - `component`  → `ErrorContext.Component`

If you store request-scoped data under those exact string keys, ewrap
picks it up for free.

## Reading the context back

```go
if ec := err.GetErrorContext(); ec != nil {
    fmt.Println(ec.RequestID, ec.Component, ec.Operation, ec.Type, ec.Severity)
}
```

The accessor returns `*ErrorContext` (or `nil`) — typed, no runtime cast.

## Wiring it into your context

If you use `context.WithValue` for request data, use **string** keys
matching the names above:

```go
ctx = context.WithValue(ctx, "request_id", reqID)
ctx = context.WithValue(ctx, "user", userID)
ctx = context.WithValue(ctx, "operation", "POST /v1/charges")
ctx = context.WithValue(ctx, "component", "billing")
```

Some teams prefer typed keys (e.g. `type ctxKey int`) for safety; in that
case, set both — the typed key for your own code, and the string key for
ewrap to consume:

```go
type ctxKey int
const reqIDKey ctxKey = iota

ctx = context.WithValue(ctx, reqIDKey, reqID) // typed key for your code
ctx = context.WithValue(ctx, "request_id", reqID) // string key for ewrap
```

## Inheritance through `Wrap`

`Wrap` carries the inherited `ErrorContext` from a wrapped `*Error`:

```go
inner := ewrap.New("DB unreachable",
    ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical))

outer := ewrap.Wrap(inner, "loading user profile")

outer.GetErrorContext() // same as inner.GetErrorContext()
```

Pass `WithContext` to `Wrap` to override at the new layer.

## Cancellation and deadlines

`WithContext` does not record `ctx.Err()` automatically. If a deadline or
cancellation is the cause, wrap that error explicitly so callers can
classify with `errors.Is`:

```go
if err := upstream.Call(ctx); err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        return ewrap.Wrap(err, "upstream call timed out",
            ewrap.WithContext(ctx, ewrap.ErrorTypeNetwork, ewrap.SeverityWarning),
            ewrap.WithRetryable(true),
            ewrap.WithHTTPStatus(http.StatusGatewayTimeout))
    }
    if errors.Is(err, context.Canceled) {
        return ewrap.Wrap(err, "upstream call cancelled",
            ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityInfo))
    }
    return ewrap.Wrap(err, "upstream call failed",
        ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityError))
}
```

## Tracing integration

`Observer.RecordError(message)` runs synchronously. Pull the active span
out of the active goroutine's context inside the observer:

```go
type otelObserver struct{}

func (o *otelObserver) RecordError(message string) {
    span := trace.SpanFromContext(activeContext()) // however your code reaches it
    if span.IsRecording() {
        span.RecordError(errors.New(message))
    }
}
```

For richer attribute propagation, store the relevant tracing data on the
error via `WithMetadata` and emit it from your `Logger` adapter — that
gives you the structured key/value pairs ewrap would otherwise hide
behind a single message.

## Why string keys for context lookup?

`context.WithValue` keys are typed `any`, so unique compile-time keys
require shared package-level types — which would create a dependency
cycle between ewrap and consumer code.

Pinning to string keys keeps ewrap independent of any particular
context-key convention. If you'd rather store everything via typed keys
in your own code, build the `ErrorContext` explicitly and use the method
form:

```go
err.WithContext(&ewrap.ErrorContext{
    RequestID: reqIDFromTypedKey(ctx),
    User:      userFromTypedKey(ctx),
    Type:      ewrap.ErrorTypeExternal,
    Severity:  ewrap.SeverityError,
})
```
