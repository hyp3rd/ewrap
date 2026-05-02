# ewrap

[![Go](https://github.com/hyp3rd/ewrap/actions/workflows/go.yml/badge.svg)](https://github.com/hyp3rd/ewrap/actions/workflows/go.yml)
[![Docs](https://img.shields.io/badge/docs-passing-brightgreen)](https://hyp3rd.github.io/ewrap/)
[![Go Report Card](https://goreportcard.com/badge/github.com/hyp3rd/ewrap)](https://goreportcard.com/report/github.com/hyp3rd/ewrap)
[![Go Reference](https://pkg.go.dev/badge/github.com/hyp3rd/ewrap.svg)](https://pkg.go.dev/github.com/hyp3rd/ewrap)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
![GitHub Sponsors](https://img.shields.io/github/sponsors/hyp3rd/sponsors)

A lightweight, modern Go error library: rich context, stack traces, structured
serialization, `slog`/`fmt.Formatter` integration, HTTP/retry classification,
PII-safe logging, and an opt-in circuit breaker — all in a tight dependency
footprint (yaml + a fast JSON encoder, nothing else).

## Highlights

- **Stdlib-first.** Two direct deps in the core module:
  [`gopkg.in/yaml.v3`][yaml] for YAML,
  [`github.com/goccy/go-json`][goccy] for the serialization hot path (~2.5× faster than `encoding/json`).
- **Correct by default.** `errors.Is` / `errors.As` work via `Unwrap()`; `Newf` honors `%w`;
  every wrap captures its own stack frames.
- **Lazy & cached hot paths.** Lazy metadata map; `Error()` and `Stack()` cached via `sync.Once`.
  After the first call, `Stack()` is ~1.7 ns/op, zero allocations.
- **Modern Go integrations.** `(*Error).Format` for `%+v`; `(*Error).LogValue` for `slog`;
  `errors.Join`-aware `ErrorGroup`.
- **Operational features.** HTTP status, retryable / `Temporary()` classification, safe
  (PII-redacted) messages, recovery suggestions, structured `ErrorContext`.
- **Opt-in subpackages.** Circuit breaker lives in [`ewrap/breaker`](breaker); `slog` adapter
  in [`ewrap/slog`](slog). Importing `ewrap` alone pulls in only the core.

[yaml]: https://pkg.go.dev/gopkg.in/yaml.v3
[goccy]: https://pkg.go.dev/github.com/goccy/go-json

## Install

```bash
go get github.com/hyp3rd/ewrap
```

Requires Go 1.25+ (uses `maps.Clone`, `slices.Clone`, range-over-int, `b.Loop`).

## Quick tour

```go
import "github.com/hyp3rd/ewrap"

// Plain error with stack trace
err := ewrap.New("database connection failed")

// %w-aware formatted constructor
err := ewrap.Newf("query %q failed: %w", q, ioErr) // errors.Is(err, ioErr) == true

// Wrap preserves the inner cause AND captures the wrap site
err := ewrap.Wrap(ioErr, "syncing replicas")

// Nil-safe
ewrap.Wrap(nil, "ignored") == nil
ewrap.Wrapf(nil, "ignored %d", 42) == nil
```

### Rich context

```go
err := ewrap.New("payment authorization rejected",
    ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityError),
    ewrap.WithHTTPStatus(http.StatusBadGateway),
    ewrap.WithRetryable(true),
    ewrap.WithSafeMessage("payment authorization rejected"), // omits PII
    ewrap.WithRecoverySuggestion(&ewrap.RecoverySuggestion{
        Message: "Inspect upstream provider's queue and retry after backoff.",
        Documentation: "https://runbooks.example.com/payments/timeout",
    }),
).
    WithMetadata("provider", "stripe").
    WithMetadata("attempt", 2)

err.Log() // emits structured fields via the configured Logger
```

### Stack traces

```go
fmt.Printf("%+v\n", err) // message + filtered stack via fmt.Formatter

// Or inspect frames programmatically
for it := err.GetStackIterator(); it.HasNext(); {
    f := it.Next()
    fmt.Printf("%s:%d %s\n", f.File, f.Line, f.Function)
}
```

`Stack()` is computed once and cached. `WithStackDepth(n)` tunes capture; pass
`0` to disable. `NewSkip` / `WrapSkip` add caller-skip when wrapping `New`/`Wrap`
in helpers.

### Standard library compatibility

```go
errors.Is(err, ioErr)  // walks the cause chain via Unwrap()
errors.As(err, &netErr)
errors.Unwrap(err)
fmt.Errorf("layered: %w", err) // also walks correctly
```

### Operational classification

```go
ewrap.HTTPStatus(err)   // walks chain; 0 if unset
ewrap.IsRetryable(err)  // checks ewrap classification, then stdlib Temporary()
err.SafeError()         // redacted variant for external sinks
err.Recovery()          // typed accessor for the recovery suggestion
err.Retry()             // typed accessor for retry metadata
err.GetErrorContext()   // typed ErrorContext or nil
```

### `slog` integration

`*Error` implements `slog.LogValuer`, so `slog.Error("boom", "err", err)`
emits the message, type, severity, component, request_id, metadata and
cause as structured fields.

For drivers that want an `ewrap.Logger`, the `slog` subpackage provides a
3-line adapter:

```go
import (
    stdslog "log/slog"
    ewrapslog "github.com/hyp3rd/ewrap/slog"
)

logger := ewrapslog.New(stdslog.New(stdslog.NewJSONHandler(os.Stdout, nil)))
err := ewrap.New("boom", ewrap.WithLogger(logger))
```

For zap, zerolog, logrus, glog, etc. — write a 5-line adapter against the
`ewrap.Logger` interface (3 methods: `Error`, `Debug`, `Info`).

### Error groups

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

`(*ErrorGroup).ToJSON()` / `ToYAML()` recursively serialize the whole group,
walking both `*Error` and standard wrapped chains so transport consumers
keep full context.

### Circuit breaker

The breaker is a sibling subpackage so consumers who only want errors don't
pay for it.

```go
import "github.com/hyp3rd/ewrap/breaker"

cb := breaker.New("payments", 5, 30*time.Second)

if !cb.CanExecute() {
    return ewrap.New("payments breaker open",
        ewrap.WithRetryable(true))
}

if err := charge(req); err != nil {
    cb.RecordFailure()

    return ewrap.Wrap(err, "charging customer",
        ewrap.WithHTTPStatus(http.StatusBadGateway))
}

cb.RecordSuccess()
```

Observers receive transitions synchronously after the lock is released:

```go
type metrics struct{ /* ... */ }

func (m *metrics) RecordTransition(name string, from, to breaker.State) {
    m.gauge.WithLabelValues(name, to.String()).Inc()
}

cb := breaker.NewWithObserver("payments", 5, 30*time.Second, &metrics{})
```

## Error types and severity

Pre-defined enums for categorisation. Their `String()` form is what shows up
in `ErrorOutput.Type` / `Severity`, JSON, and `slog` fields.

```go
ErrorTypeUnknown        // -> "unknown"
ErrorTypeValidation     // -> "validation"
ErrorTypeNotFound       // -> "not_found"
ErrorTypePermission     // -> "permission"
ErrorTypeDatabase       // -> "database"
ErrorTypeNetwork        // -> "network"
ErrorTypeConfiguration  // -> "configuration"
ErrorTypeInternal       // -> "internal"
ErrorTypeExternal       // -> "external"

SeverityInfo            // -> "info"
SeverityWarning         // -> "warning"
SeverityError           // -> "error"
SeverityCritical        // -> "critical"
```

## Logger interface

```go
type Logger interface {
    Error(msg string, keysAndValues ...any)
    Debug(msg string, keysAndValues ...any)
    Info(msg string, keysAndValues ...any)
}
```

Three methods, key-value pairs after the message. Implementations stay
goroutine-safe; `(*Error).Log` calls them synchronously.

## Performance

Snapshot from `go test -bench=. -benchmem ./test/...` on Apple Silicon (Go 1.25+):

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkNew/Simple` | 1622 | 496 | 2 |
| `BenchmarkWrap/NestedWraps` | 11433 | 1512 | 9 |
| `BenchmarkFormatting/ToJSON` | 16947 | 2941 | 14 |
| `BenchmarkStackTrace/CaptureStack` | 858 | 256 | 1 |
| `BenchmarkStackTrace/FormatStack` (cached) | **1.71** | 0 | **0** |
| `BenchmarkCircuitBreaker/RecordFailure` | 33 | 0 | 0 |
| `BenchmarkMetadataOperations/GetMetadata` | 9 | 0 | 0 |

Notable design choices behind the numbers:

- **Lazy metadata map** — only allocated on the first `WithMetadata` call.
- **Cached `Error()` / `Stack()`** — `sync.Once` guards a one-shot computation;
  subsequent reads are lock-free.
- **goccy/go-json** for the serialization hot path: ~2.5× faster than
  stdlib `encoding/json` with ~half the allocations.
- **`runtime.Callers`** captures up to 32 PCs by default, configurable via
  `WithStackDepth(n)`. The frame filter is function-prefix based, so the
  output starts at user code.
- **Breaker** is allocation-free in steady state; observer/callback dispatch
  happens outside the lock to avoid holding it across user code.

## Project layout

```text
.
├── attributes.go              # WithHTTPStatus, WithRetryable, WithSafeMessage
├── context.go                 # ErrorContext, WithContext option
├── errors.go                  # Error type, New/Wrap/Newf/Wrapf, lazy paths
├── error_group.go             # ErrorGroup, pool, serialization
├── format.go                  # ErrorOutput, ToJSON/ToYAML
├── format_verb.go             # fmt.Formatter, slog.LogValuer
├── logger.go                  # Logger interface
├── observability.go           # Observer interface (errors only)
├── retry.go                   # RetryInfo, WithRetry
├── stack.go                   # StackFrame, StackIterator
├── types.go                   # ErrorType, Severity, RecoverySuggestion
├── breaker/                   # opt-in circuit breaker
└── slog/                      # opt-in slog adapter
```

## Development

```bash
git clone https://github.com/hyp3rd/ewrap.git
cd ewrap
make prepare-toolchain    # one-time: golangci-lint, gofumpt, govulncheck, gosec
make test                 # go test -v -timeout 5m -cover ./...
make test-race            # go test -race ./...
make benchmark            # go test -bench=. -benchmem ./test/...
make lint                 # gci + gofumpt + staticcheck + golangci-lint
make sec                  # govulncheck + gosec
```

## License

[MIT License](LICENSE)

## Contributing

See [CONTRIBUTING](CONTRIBUTING.md). PRs welcome — please run `make lint` and
`make test-race` before opening one.

## Author

I'm a surfer, and a software architect with 15 years of experience designing highly available distributed production systems and developing cloud-native apps in public and private clouds. Feel free to connect with me on LinkedIn.

[![LinkedIn](https://img.shields.io/badge/LinkedIn-0077B5?style=for-the-badge&logo=linkedin&logoColor=white)](https://www.linkedin.com/in/francesco-cosentino/)
