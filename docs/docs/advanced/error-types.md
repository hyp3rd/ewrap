# Error Types in Practice

The [`ErrorType`](../api/error-types.md) enum is small on purpose. This
page covers how to use the existing values consistently and how to extend
classification when the built-ins aren't enough.

## When to use each built-in

| `ErrorType` | Trigger | Maps cleanly to |
| --- | --- | --- |
| `Validation` | Caller-supplied input failed checks | HTTP 400/422, gRPC `InvalidArgument` |
| `NotFound` | Resource lookup returned no rows | HTTP 404, gRPC `NotFound` |
| `Permission` | AuthN / AuthZ failures | HTTP 401/403, gRPC `PermissionDenied`/`Unauthenticated` |
| `Database` | Storage layer failure | HTTP 500/503, gRPC `Internal` |
| `Network` | Connectivity, DNS, TLS handshake | HTTP 502/504, gRPC `Unavailable` |
| `Configuration` | Misconfiguration at startup or runtime | HTTP 500, gRPC `FailedPrecondition` |
| `Internal` | Bug or invariant violation | HTTP 500, gRPC `Internal` |
| `External` | Third-party service rejected the request | HTTP 502/503, gRPC `Unavailable`/`FailedPrecondition` |
| `Unknown` | Default — avoid in production | HTTP 500 |

`Validation` is the only type with bespoke library behaviour: the default
retry predicate refuses to retry validation errors.

## Pair with `WithHTTPStatus`

`ErrorType` is for classification; `WithHTTPStatus` is for transport.
Set both — one for routing/branching, one for the wire:

```go
ewrap.New("invalid email",
    ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityWarning),
    ewrap.WithHTTPStatus(http.StatusUnprocessableEntity))
```

This way, `ec.Type == ErrorTypeValidation` covers internal logic and
`ewrap.HTTPStatus(err) == 422` covers handler responses, with no risk of
the two drifting.

## Severity vs HTTP status

`Severity` answers *how loud should we be* — pager versus dashboard
versus log line. Don't conflate it with HTTP status:

| | Severity | HTTP status |
| --- | --- | --- |
| 401 Unauthorized | `Warning` | `401` |
| 422 Validation | `Warning` | `422` |
| 500 Internal | `Error` | `500` |
| 503 Out of capacity | `Critical` | `503` |
| 503 Breaker open | `Warning` | `503` |

The severity drives alerting; the status drives the response.

## Extending classification

If the built-in types don't fit your domain, two approaches:

### Custom metadata key

```go
ewrap.New("rate limit exceeded",
    ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityWarning)).
    WithMetadata("ewrap.subtype", "rate_limit").
    WithMetadata("retry_after_s", 30)
```

Read with `ewrap.GetMetadataValue[string](err, "ewrap.subtype")`.
Cheap, keeps you on the existing enum, surfaces in JSON / slog
automatically.

### Extra error type in your own package

For a closed domain enum, define your own and use it alongside ewrap's:

```go
package billing

type Code int

const (
    CodeUnknown Code = iota
    CodeCardDeclined
    CodeRateLimited
    CodeFraudDetected
)

func New(code Code, msg string, opts ...ewrap.Option) *ewrap.Error {
    return ewrap.NewSkip(1, msg, append(opts,
        ewrap.WithContext(context.Background(), ewrap.ErrorTypeExternal, ewrap.SeverityWarning),
    )...).WithMetadata("billing_code", code)
}
```

Callers branch on the metadata:

```go
if c, ok := ewrap.GetMetadataValue[billing.Code](err, "billing_code"); ok {
    switch c {
    case billing.CodeCardDeclined:
        // ...
    }
}
```

This keeps ewrap's enum closed (so we don't ship breaking-change new
values), while letting your domain pile on as much specificity as you
need.

## Avoid the "unknown" default

A value of `ErrorTypeUnknown` (the zero value) usually means somebody
forgot `WithContext`. Two ways to keep it out of production:

1. **CI lint:** grep for new `ewrap.New(...)` / `ewrap.Wrap(...)` calls
   that don't include `WithContext`. Easy to pair with a CODEOWNERS rule.
2. **Default in handlers:** when serializing to a response, treat
   `ErrorTypeUnknown` as a 500 with a generic message — never echo back
   the raw text — and emit a metric so the gap is visible.
