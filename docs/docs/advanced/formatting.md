# Formatting

Three ways to render an `*Error` for output. Pick the one that matches
the consumer.

| Consumer | Use |
| --- | --- |
| Human reader (terminal, dev logs) | `fmt.Printf("%+v", err)` |
| Structured log sink (slog/zap/zerolog) | `LogValuer` (automatic) or `(*Error).Log` |
| Machine pipeline (transport, dashboard) | `(*Error).ToJSON()` / `ToYAML()` |

## `fmt.Formatter`

`*Error` implements `fmt.Formatter` and supports four verbs:

```go
err := ewrap.New("boom").WithMetadata("k", "v")

fmt.Printf("%s\n", err)   // boom
fmt.Printf("%v\n", err)   // boom
fmt.Printf("%q\n", err)   // "boom"
fmt.Printf("%+v\n", err)  // boom\n<filtered stack>\n
```

| Verb | Output |
| --- | --- |
| `%s`, `%v` | `Error()` |
| `%q` | quoted `Error()` |
| `%+v` | `Error()` + newline + `Stack()` |

Both `Error()` and `Stack()` are cached, so formatting is essentially free
after the first call.

## `slog.LogValuer`

`*Error` implements `slog.LogValuer`, so passing it as a value in any
`log/slog` call emits structured fields:

```go
slog.Error("payment failed", "err", err)
```

The handler receives an attribute group with:

- `message` (always)
- `type`, `severity` (if `WithContext` was set)
- `component`, `operation`, `request_id` (if non-empty in context)
- `recovery` (if `WithRecoverySuggestion` was set)
- `cause` (if non-nil)
- one attribute per metadata key

If you'd rather log via `(*Error).Log` (using an attached `ewrap.Logger`),
the same fields appear:

```go
err := ewrap.New("boom",
    ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical),
    ewrap.WithLogger(logger))
err.Log()
```

`Log` writes a single record at error level.

## JSON and YAML

```go
jsonStr, _ := err.ToJSON(
    ewrap.WithTimestampFormat(time.RFC3339),
    ewrap.WithStackTrace(true),
)

yamlStr, _ := err.ToYAML(ewrap.WithStackTrace(false))
```

The schema is documented in [Serialization](../features/serialization.md).
Two format options:

| Option | Effect |
| --- | --- |
| `WithTimestampFormat(layout)` | Reformat the `timestamp` field. Empty = leave unchanged. |
| `WithStackTrace(false)` | Strip the `stack` field. |

For an `ErrorGroup`, the same options apply via `(*ErrorGroup).ToJSON()`
/ `ToYAML()`. The group also implements `json.Marshaler` and
`yaml.Marshaler`, so generic encoders that consume them via
`json.Marshal` / `yaml.Marshal` work without ceremony.

## Picking the timestamp format

Default: RFC3339 (e.g. `2026-05-02T10:11:12Z`). Most log pipelines parse
that natively. Use the friendlier alternatives only when emitting
direct-to-human output:

```go
err.ToJSON(ewrap.WithTimestampFormat(time.DateTime))      // 2026-05-02 10:11:12
err.ToJSON(ewrap.WithTimestampFormat("2006-01-02"))       // 2026-05-02
```

## Stripping stacks for hot paths

A formatted stack trace adds a few hundred bytes to each serialized
record. If you serialize on a high-volume path (e.g. error metrics
shipping), strip the stack and capture it only in dev/debug paths:

```go
const includeStack = false // toggle in your config

opts := []ewrap.FormatOption{ewrap.WithStackTrace(includeStack)}
jsonStr, _ := err.ToJSON(opts...)
```

## SafeError for external sinks

When the rendered output may leave your trust boundary (third-party log
ingestion, customer-facing responses), use `SafeError()`:

```go
external.Log(err.SafeError()) // redacted variant
internal.Log(err.Error())     // full detail
```

See [Operational Features](../features/operational.md) for how to attach
safe messages.

## Performance summary

| Operation | ns/op | allocs |
| --- | ---: | ---: |
| `fmt.Sprintf("%v", err)` (cached) | ~30 | 1 (output buffer) |
| `fmt.Sprintf("%+v", err)` (cached) | ~70 | 1 |
| `(*Error).Log` (with metadata) | ~500 | a few (logger-dependent) |
| `(*Error).ToJSON` | ~17,000 | ~14 |
| `(*Error).ToYAML` | ~250,000 | ~115 |

The first call to `Error()` and `Stack()` does the formatting; subsequent
calls hit the cache. JSON output is dominated by `goccy/go-json` (already
~2.5× faster than stdlib for this payload shape). YAML is significantly
slower — prefer JSON wherever the consumer accepts it.
