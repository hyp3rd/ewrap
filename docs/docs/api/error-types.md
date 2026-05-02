# Error Types and Severity

ewrap ships small enums for classifying errors. Both have a `String()`
method whose output is what shows up in `ErrorOutput.Type`/`Severity`,
JSON, YAML, and `slog` records — pin yourself to these values rather than
re-stringifying.

## `ErrorType`

```go
type ErrorType int

const (
    ErrorTypeUnknown        // -> "unknown"
    ErrorTypeValidation     // -> "validation"
    ErrorTypeNotFound       // -> "not_found"
    ErrorTypePermission     // -> "permission"
    ErrorTypeDatabase       // -> "database"
    ErrorTypeNetwork        // -> "network"
    ErrorTypeConfiguration  // -> "configuration"
    ErrorTypeInternal       // -> "internal"
    ErrorTypeExternal       // -> "external"
)
```

| Type | When to use |
| --- | --- |
| `Unknown` | Unclassified — default if you don't supply `WithContext`. |
| `Validation` | Caller-supplied input failed checks. Not retryable by default. |
| `NotFound` | Resource doesn't exist. Maps cleanly to HTTP 404. |
| `Permission` | Authentication / authorisation failures. Maps to 401/403. |
| `Database` | Storage layer failure (driver, connection pool, query). |
| `Network` | Connectivity, DNS, TLS handshake failures. |
| `Configuration` | Misconfiguration detected at startup or runtime. |
| `Internal` | Bug or invariant violation in your own code. |
| `External` | Third-party service failure beyond `Network` (e.g. rate-limit, 5xx). |

### Setting

```go
ewrap.New("invalid email",
    ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityWarning))
```

### Reading

```go
if ec := err.GetErrorContext(); ec != nil {
    switch ec.Type {
    case ewrap.ErrorTypeValidation:
        // 422 / 400
    case ewrap.ErrorTypeNotFound:
        // 404
    }
}
```

You can also branch on the canonical string form (`ec.Type.String()`),
useful in dashboards or log filters.

### Default retry semantics

The default `WithRetry` predicate (`defaultShouldRetry`) treats
`ErrorTypeValidation` as **non-retryable** and everything else as
retryable. Override with `WithRetryShould` for finer control.

## `Severity`

```go
type Severity int

const (
    SeverityInfo      // -> "info"
    SeverityWarning   // -> "warning"
    SeverityError     // -> "error"
    SeverityCritical  // -> "critical"
)
```

| Severity | Typical use |
| --- | --- |
| `Info` | Notable but not failed (e.g. degraded mode). |
| `Warning` | Recovered automatically; likely worth investigating. |
| `Error` | Operation failed; user-facing or actionable. |
| `Critical` | System-impacting; page someone. |

### Setting severity

```go
ewrap.New("DB unreachable",
    ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical))
```

### Reading severity

```go
if ec := err.GetErrorContext(); ec != nil && ec.Severity >= ewrap.SeverityError {
    pageOncall(err)
}
```

## `RecoverySuggestion`

```go
type RecoverySuggestion struct {
    Message       string   `json:"message"       yaml:"message"`
    Actions       []string `json:"actions"       yaml:"actions"`
    Documentation string   `json:"documentation" yaml:"documentation"`
}
```

A typed payload describing what an operator should do about the error.
Attached via `WithRecoverySuggestion`, read via `(*Error).Recovery()`,
and emitted by `(*Error).Log` as `recovery_message`, `recovery_actions`,
`recovery_documentation` fields.

```go
ewrap.New("payment provider timeout",
    ewrap.WithRecoverySuggestion(&ewrap.RecoverySuggestion{
        Message:       "Inspect provider's queue and retry after backoff.",
        Actions:       []string{"check status page", "retry with backoff"},
        Documentation: "https://runbooks.example.com/payments/timeout",
    }))
```

## `ErrorContext`

```go
type ErrorContext struct {
    Timestamp   time.Time
    Type        ErrorType
    Severity    Severity
    Operation   string
    Component   string
    RequestID   string
    User        string
    Environment string
    Version     string
    File        string
    Line        int
    Data        map[string]any
}
```

Built by the `WithContext(ctx, type, severity)` option, which reads
`request_id`, `user`, `operation`, and `component` out of the
`context.Context` if present. `File`/`Line` are captured automatically
via `runtime.Caller`. `Environment` falls back to `APP_ENV` or
`"development"`.

You can also build one explicitly and pass it via the method form:

```go
err.WithContext(&ewrap.ErrorContext{
    Type:      ewrap.ErrorTypeNetwork,
    Severity:  ewrap.SeverityError,
    RequestID: "req-123",
    Component: "billing",
})
```

## `RetryInfo`

```go
type RetryInfo struct {
    MaxAttempts    int
    CurrentAttempt int
    Delay          time.Duration
    LastAttempt    time.Time
    ShouldRetry    func(error) bool
}
```

Attached via `WithRetry`, read via `(*Error).Retry()`, driven via
`(*Error).CanRetry()` and `(*Error).IncrementRetry()`.

The `ShouldRetry` predicate is consulted by `CanRetry` along with
`CurrentAttempt < MaxAttempts`. The default is
`defaultShouldRetry` (refuses validation errors); override with
`WithRetryShould`.

## `StackFrame`

```go
type StackFrame struct {
    Function string  `json:"function" yaml:"function"`
    File     string  `json:"file"     yaml:"file"`
    Line     int     `json:"line"     yaml:"line"`
    PC       uintptr `json:"pc"       yaml:"pc"`
}
```

A single decoded stack frame. Returned in slices by
`(*Error).GetStackFrames()` and via `*StackIterator` from
`GetStackIterator()`.

## `StackTrace`

```go
type StackTrace []StackFrame
```

A slice alias, mostly used for documentation purposes.

## `ErrorOutput`

The schema for `(*Error).ToJSON` / `ToYAML`. See
[Serialization](../features/serialization.md) for the full layout.

## `SerializableError` and `ErrorGroupSerialization`

The schemas for `(*ErrorGroup).ToJSON` / `ToYAML`. Same source.

## `breaker.State`

```go
type State int

const (
    Closed   // "closed"
    Open     // "open"
    HalfOpen // "half-open"
)
```

See [Circuit Breaker](../features/circuit-breaker.md) for behaviour.
