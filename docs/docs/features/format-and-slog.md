# `fmt.Formatter` and `slog.LogValuer`

`*Error` implements both stdlib hooks, so it formats nicely with
`fmt.Printf` and emits structured fields directly into `log/slog`.

## `fmt.Formatter`

```go
err := ewrap.New("boom").WithMetadata("k", "v")

fmt.Printf("%s\n", err)   // boom
fmt.Printf("%v\n", err)   // boom
fmt.Printf("%q\n", err)   // "boom"
fmt.Printf("%+v\n", err)  // boom\n<filtered stack trace>\n
```

| Verb | Output |
| --- | --- |
| `%s` | `Error()` text |
| `%v` | `Error()` text |
| `%q` | quoted `Error()` text |
| `%+v` | `Error()` + newline + `Stack()` |

The `%+v` variant is the canonical pkg/errors-style "pretty-print with
stack" for log/dev output. Both `Error()` and `Stack()` are cached, so
formatting the same error multiple times is essentially free after the
first call.

### Implementation sketch

```go
func (e *Error) Format(state fmt.State, verb rune) {
    switch verb {
    case 'v':
        if state.Flag('+') {
            fmt.Fprintf(state, "%s\n%s", e.Error(), e.Stack())
            return
        }
        fmt.Fprint(state, e.Error())
    case 'q':
        fmt.Fprintf(state, "%q", e.Error())
    default:
        fmt.Fprint(state, e.Error())
    }
}
```

## `slog.LogValuer`

```go
slog.Error("payment failed", "err", err)
```

Without `LogValuer`, `slog` would render `err` as an opaque string. With
it, the handler receives a structured group:

```text
level=ERROR msg="payment failed" err.message="boom"
  err.type=external err.severity=error err.component=billing
  err.request_id=req-123 err.cause="net/http: Bad Gateway"
  err.recovery="Retry after backoff." err.k=v
```

The emitted attribute set:

| Key | When |
| --- | --- |
| `message` | always — `e.Error()` |
| `type` | if `WithContext` was used |
| `severity` | if `WithContext` was used |
| `component` | if `ErrorContext.Component` is non-empty |
| `operation` | if `ErrorContext.Operation` is non-empty |
| `request_id` | if `ErrorContext.RequestID` is non-empty |
| `recovery` | if `WithRecoverySuggestion` was used |
| `cause` | if the error has a cause — `cause.Error()` |
| _user metadata_ | one attribute per metadata key |

The whole payload is wrapped in an `slog.GroupValue`, so it appears under
the attribute key you used at the call site (`err` in the example).

### When you only have `slog`, you don't need an adapter

`*Error` satisfies `slog.LogValuer` directly, so any `*slog.Logger` will
render it correctly:

```go
slog.New(slog.NewJSONHandler(os.Stdout, nil)).
    With("service", "billing").
    Error("payment failed", "err", err)
```

If you want the inverse (use `*slog.Logger` as an `ewrap.Logger`), import
the [`ewrap/slog`](slog-adapter.md) subpackage.

## Performance

Both `Format` and `LogValue` reuse the cached `Error()` and `Stack()`
strings. After the first call:

- `fmt.Sprintf("%v", err)` — one cached string read, no extra allocations
- `fmt.Sprintf("%+v", err)` — one cached message read + one cached stack
  read, joined into a single output buffer
- `slog.Error("...", "err", err)` — `LogValue` builds the attribute slice
  fresh per call (since metadata is mutable), but the per-attribute
  strings come from the cached values

For high-volume hot paths where you log the same error many times, prefer
`%v` (no stack) over `%+v` (with stack) — the latter writes the full
trace each time even though it's read from cache.
