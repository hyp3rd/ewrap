# ewrap Documentation

`ewrap` is a lightweight, modern Go error library: rich context, stack traces,
structured serialization, `slog` and `fmt.Formatter` integration, HTTP / retry
classification, PII-safe logging, and an opt-in circuit breaker — all in a
tight dependency footprint (yaml + a fast JSON encoder, nothing else).

## Highlights

- **Stdlib-first.** Two direct deps in the core module: `gopkg.in/yaml.v3` for
  YAML, `github.com/goccy/go-json` for the serialization hot path.
- **Correct by default.** `errors.Is` / `errors.As` work via `Unwrap()`; `Newf`
  honors `%w`; every wrap captures its own stack frames.
- **Lazy & cached hot paths.** Lazy metadata map; `Error()` and `Stack()`
  cached via `sync.Once`. After the first call, `Stack()` is ~1.7 ns/op,
  zero allocations.
- **Modern Go integrations.** `(*Error).Format` for `%+v`; `(*Error).LogValue`
  for `slog`; `errors.Join`-aware `ErrorGroup`.
- **Operational features.** HTTP status, retryable / `Temporary()`
  classification, safe (PII-redacted) messages, recovery suggestions,
  structured `ErrorContext`.
- **Opt-in subpackages.** Circuit breaker lives in `ewrap/breaker`; `slog`
  adapter in `ewrap/slog`. Importing `ewrap` alone pulls in only the core.

## Quick example

```go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "os"
    "time"

    "github.com/hyp3rd/ewrap"
    "github.com/hyp3rd/ewrap/breaker"
    ewrapslog "github.com/hyp3rd/ewrap/slog"
)

func processOrder(ctx context.Context, orderID string) error {
    logger := ewrapslog.New(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

    cb := breaker.New("payments", 5, 30*time.Second)
    if !cb.CanExecute() {
        return ewrap.New("payments breaker open",
            ewrap.WithRetryable(true),
            ewrap.WithLogger(logger))
    }

    if err := charge(ctx, orderID); err != nil {
        cb.RecordFailure()

        return ewrap.Wrap(err, "charging customer",
            ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityError),
            ewrap.WithHTTPStatus(http.StatusBadGateway),
            ewrap.WithRetryable(true),
            ewrap.WithSafeMessage("charge failed"),
            ewrap.WithLogger(logger),
        ).
            WithMetadata("order_id", orderID)
    }

    cb.RecordSuccess()

    return nil
}
```

This example uses every subsystem: structured logging, the `breaker`
subpackage, classification (`HTTPStatus`, `Retryable`), PII redaction
(`SafeMessage`), and metadata.

## Why ewrap

| Need | What ewrap provides |
| --- | --- |
| Wrap with stack traces | `New`, `Wrap`, `Newf` (`%w`-aware), `Wrapf` |
| Structured cause chain | `Unwrap()`; `errors.Is/As` work as you'd expect |
| Pretty stack output | `fmt.Printf("%+v", err)` via `fmt.Formatter` |
| Structured logging | `slog.LogValuer` exposes typed fields automatically |
| Multi-error aggregation | `ErrorGroup` (pooled) + `errors.Join` |
| HTTP / retry policy | `WithHTTPStatus` / `WithRetryable` + walkers |
| PII-safe logs | `WithSafeMessage` / `(*Error).SafeError()` |
| Cascade protection | `breaker` subpackage (no extra deps for non-users) |
| Transport | `ToJSON`, `ToYAML` walk both `*Error` and standard chains |

## Next steps

- [Installation](getting-started/installation.md)
- [Quick Start](getting-started/quickstart.md)
- [Error Creation](features/error-creation.md) and [Wrapping](features/error-wrapping.md)
- [HTTP / Retry / Safe operational features](features/operational.md)
- [Circuit breaker subpackage](features/circuit-breaker.md)
- [API Reference](api/overview.md)
