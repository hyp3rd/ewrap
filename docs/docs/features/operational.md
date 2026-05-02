# Operational Features

Three small, orthogonal features for production use:

- **HTTP status** — attach and walk a status code along the cause chain.
- **Retryable / Temporary** — classify whether retrying makes sense.
- **Safe message** — emit a redacted variant for logs that may leave the
  trust boundary.

Each is set with an option at construction (or inherited via `Wrap`) and
read either via a method on `*Error` or a top-level walker function.

## HTTP status

```go
err := ewrap.New("upstream rejected request",
    ewrap.WithHTTPStatus(http.StatusBadGateway))

ewrap.HTTPStatus(err)            // 502
ewrap.HTTPStatus(io.EOF)         // 0 — no ewrap layer set one
ewrap.HTTPStatus(nil)            // 0
```

`HTTPStatus(err)` walks the chain via `errors.As` and returns the **first
non-zero** status it finds. Wrapping a tagged error keeps the status:

```go
inner := ewrap.New("rejected", ewrap.WithHTTPStatus(http.StatusBadGateway))
outer := ewrap.Wrap(inner, "fetching invoice")

ewrap.HTTPStatus(outer) // 502, inherited from inner
```

A standard `fmt.Errorf("...: %w", inner)` wrapper also works — `HTTPStatus`
walks past it via `errors.Unwrap`.

### Typical use in an HTTP handler

```go
func handle(w http.ResponseWriter, r *http.Request) {
    if err := process(r); err != nil {
        status := ewrap.HTTPStatus(err)
        if status == 0 {
            status = http.StatusInternalServerError
        }
        http.Error(w, err.Error(), status)
        return
    }
}
```

## Retryable / Temporary

```go
err := ewrap.New("rate limited", ewrap.WithRetryable(true))

ewrap.IsRetryable(err) // true
```

The classification is **explicit and three-state**:

```go
err.Retryable() // (value, set bool)
// set == false → not classified
// set == true  → value is the explicit classification
```

`ewrap.IsRetryable(err)` walks the chain. If no ewrap layer set the
flag, it falls through to the stdlib `interface{ Temporary() bool }`,
which `net.OpError`, `*net.DNSError`, and friends already implement:

```go
ewrap.IsRetryable(myNetErr) // honours net.OpError.Temporary()
```

### Typical use in a retry loop

```go
for attempt := 1; attempt <= max; attempt++ {
    err := callUpstream(req)
    if err == nil {
        return nil
    }

    if !ewrap.IsRetryable(err) {
        return err
    }

    time.Sleep(backoff(attempt))
}
```

### Combining with `WithRetry`

`WithRetry` carries a per-error retry **policy** (max attempts, delay,
predicate). `WithRetryable` is a simple **classification** flag. Use
both together when you want a self-describing retryable error:

```go
err := ewrap.New("upstream timeout",
    ewrap.WithRetryable(true),
    ewrap.WithRetry(3, 5*time.Second))
```

## Safe (PII-redacted) messages

`SafeError()` returns a redacted variant of the error chain suitable for
external sinks (third-party logs, customer-visible responses, public
metrics):

```go
err := ewrap.New("user 'alice@example.com' rejected",
    ewrap.WithSafeMessage("user [redacted] rejected"))

err.Error()     // "user 'alice@example.com' rejected"
err.SafeError() // "user [redacted] rejected"
```

`SafeError` walks the chain. Each layer contributes either its
`WithSafeMessage` value (if set) or its raw `msg`. Standard wrapped
errors without a `SafeError` method are included verbatim — wrap them in
an `ewrap.Error` with `WithSafeMessage` if they may contain PII:

```go
root := ewrap.New("token=secret123", ewrap.WithSafeMessage("token=[redacted]"))
outer := ewrap.Wrap(root, "auth failed for user@example.com",
    ewrap.WithSafeMessage("auth failed for [redacted]"))

outer.Error()     // "auth failed for user@example.com: token=secret123"
outer.SafeError() // "auth failed for [redacted]: token=[redacted]"
```

### Typical use in dual-sink logging

```go
logger.Error("internal", "err", err.Error())     // full detail to private sink
external.Error("public", "err", err.SafeError()) // redacted to public sink
```

## Inheritance through `Wrap`

All three classifications are inherited when wrapping an `ewrap.Error`:

```go
inner := ewrap.New("boom",
    ewrap.WithHTTPStatus(http.StatusBadGateway),
    ewrap.WithRetryable(true))

outer := ewrap.Wrap(inner, "in handler")

ewrap.HTTPStatus(outer) // 502
ewrap.IsRetryable(outer) // true
```

Pass an option to `Wrap` to override the inherited value at the new
layer.

## What's intentionally not here

- **gRPC status codes** — would require pulling in `google.golang.org/grpc`.
  Use `WithHTTPStatus` and translate at the boundary, or implement a tiny
  gRPC subpackage in your own repo.
- **Message templates / i18n** — out of scope. Build your own helper that
  calls `WithSafeMessage` with the localized string.
- **Automatic PII detection** — too domain-specific. `WithSafeMessage` is
  the explicit hook; reach for it where the original message can leak.
