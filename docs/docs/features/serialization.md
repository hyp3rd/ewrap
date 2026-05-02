# Serialization

ewrap can serialize a single `*Error` or a whole `*ErrorGroup` to JSON or
YAML. The output schema is stable and walks the entire cause chain —
including non-`*Error` wrappers via `errors.Unwrap` — so transport
consumers don't lose context across module boundaries.

## Single error

```go
err := ewrap.New("payment failed",
    ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityError),
    ewrap.WithRecoverySuggestion(&ewrap.RecoverySuggestion{
        Message: "Retry after backoff.",
    })).
    WithMetadata("provider", "stripe")

jsonStr, _ := err.ToJSON(
    ewrap.WithTimestampFormat(time.RFC3339),
    ewrap.WithStackTrace(true),
)

yamlStr, _ := err.ToYAML(ewrap.WithStackTrace(false))
```

### Schema

```json
{
  "message": "payment failed",
  "timestamp": "2026-05-02T10:11:12Z",
  "type": "external",
  "severity": "error",
  "stack": "/repo/pay.go:42 example.com/pay.charge\n...",
  "context": {
    "request_id": "req-123",
    "user": "u-1",
    "component": "billing",
    "operation": "charge",
    "file": "/repo/pay.go",
    "line": 42,
    "environment": "prod"
  },
  "metadata": {
    "provider": "stripe"
  },
  "recovery": {
    "message": "Retry after backoff.",
    "actions": [],
    "documentation": ""
  },
  "cause": null
}
```

The `cause` field nests the same shape recursively for chained errors.

### Format options

| Option | Effect |
| --- | --- |
| `WithTimestampFormat(layout)` | Reformats the `timestamp` field (parses RFC3339 in, emits the supplied layout). Empty layout = leave unchanged. |
| `WithStackTrace(false)` | Removes the `stack` field from the output. |

Use both together for compact, dashboard-friendly output:

```go
jsonStr, _ := err.ToJSON(
    ewrap.WithTimestampFormat(time.DateTime),
    ewrap.WithStackTrace(false),
)
```

## Error groups

```go
eg := pool.Get()
defer eg.Release()

eg.Add(httpErr)
eg.Add(dbErr)

groupJSON, _ := eg.ToJSON()
groupYAML, _ := eg.ToYAML()
```

`ErrorGroup` also implements `json.Marshaler` and `yaml.Marshaler` directly,
so encoders that consume them via `json.Marshal` / `yaml.Marshal` work with
zero ceremony.

### Group schema

```json
{
  "error_count": 2,
  "timestamp": "2026-05-02T10:11:12Z",
  "errors": [
    {
      "message": "fetching user: net/http: Bad Gateway",
      "type": "ewrap",
      "stack_trace": [
        {"function": "...", "file": "...", "line": 42, "pc": 12345}
      ],
      "metadata": {"http_status": 502},
      "cause": {
        "message": "net/http: Bad Gateway",
        "type": "standard"
      }
    },
    {
      "message": "EOF",
      "type": "standard"
    }
  ]
}
```

`type` is `"ewrap"` for `*Error` members and `"standard"` for everything
else. `stack_trace` and `metadata` are emitted only for `*Error` members.

## Cause chain across boundaries

The serializer walks both `*Error` chains and standard wrapped chains:

```go
inner := ewrap.New("DB unreachable",
    ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical))
mid := fmt.Errorf("loading user: %w", inner)            // standard wrapper
outer := ewrap.Wrap(mid, "handling /users/{id}")        // ewrap on top

eg := pool.Get()
eg.Add(outer)

// outer -> mid (via errors.Unwrap) -> inner (*Error)
// All three layers appear in the serialized cause chain.
```

This works because `toSerializableError` falls through to `errors.Unwrap`
for non-`*Error` causes — you don't have to convert everything to ewrap
upfront.

## Performance

| Benchmark | ns/op | allocs |
| --- | ---: | ---: |
| `Error.ToJSON` (with context, two metadata keys) | ~17,000 | ~14 |
| `Error.ToYAML` (same) | ~250,000 | ~115 |
| `ErrorGroup.ToJSON` (10 entries) | ~10 µs | ~30 |

JSON uses `github.com/goccy/go-json`, ~2.5× faster than stdlib
`encoding/json` on this payload shape with about half the allocations.

YAML uses `gopkg.in/yaml.v3`. It's significantly slower than JSON; if
serialization is hot, prefer JSON.

## Tips

- For machine consumption, **prefer JSON** — both faster and more widely
  supported in observability sinks.
- **Strip stacks** for high-volume sinks (`WithStackTrace(false)`) and
  attach them in dev/debug paths only.
- **Use `RFC3339`** as the timestamp format unless you have a strong
  reason to deviate; it parses cleanly in every common log pipeline.
- **Set `ErrorContext`** on at least one layer so `type` and `severity`
  carry signal. Without it both default to `"unknown"` / `"error"`.
