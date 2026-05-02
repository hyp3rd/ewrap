# Logging Integration

ewrap defines a tiny three-method `Logger` interface and ships a single
adapter (for stdlib `log/slog`) in a sibling subpackage. Adapters for
zap, zerolog, logrus, glog, etc. are intentionally **not** bundled: the
interface is so small that a working adapter is well under ten lines of
your own code.

## The interface

```go
type Logger interface {
    Error(msg string, keysAndValues ...any)
    Debug(msg string, keysAndValues ...any)
    Info(msg string, keysAndValues ...any)
}
```

`keysAndValues` is the standard structured-logging convention: alternating
key/value pairs after the message. Implementations must be goroutine-safe
because `(*Error).Log` calls them synchronously from the calling
goroutine.

## Attaching a logger

```go
err := ewrap.New("payment failed", ewrap.WithLogger(logger))
err.Log() // emits an "error occurred" record with all attached fields
```

`(*Error).Log` emits a single record containing:

- `error` — the message
- `cause` — `e.cause.Error()` if the chain has one
- `stack` — formatted stack trace
- every key/value from the metadata map
- `recovery_message`, `recovery_actions`, `recovery_documentation` if
  `WithRecoverySuggestion` was used

The logger reference is also inherited by `Wrap` when the inner error is
already a `*Error`, so a single `WithLogger` near the root propagates out.

## Slog adapter

Stdlib `log/slog` is the recommended target for new projects. The adapter
is in [`ewrap/slog`](slog-adapter.md):

```go
import (
    "log/slog"
    "os"

    "github.com/hyp3rd/ewrap"
    ewrapslog "github.com/hyp3rd/ewrap/slog"
)

handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
logger := ewrapslog.New(slog.New(handler))

err := ewrap.New("payment failed", ewrap.WithLogger(logger))
err.Log()
```

If you'd rather log the error directly via `slog`, you don't need an
adapter at all — `*Error` implements `slog.LogValuer`:

```go
slog.Error("payment failed", "err", err)
// emits structured fields: message, type, severity, request_id, cause,
// metadata, recovery — all without the adapter
```

See [fmt.Formatter & slog](format-and-slog.md) for the `LogValuer` details.

## Writing an adapter for another logger

The whole adapter is three methods. Here's zap:

```go
import "go.uber.org/zap"

type ZapAdapter struct{ l *zap.SugaredLogger }

func NewZap(l *zap.Logger) *ZapAdapter         { return &ZapAdapter{l: l.Sugar()} }
func (a *ZapAdapter) Error(msg string, kv ...any) { a.l.Errorw(msg, kv...) }
func (a *ZapAdapter) Debug(msg string, kv ...any) { a.l.Debugw(msg, kv...) }
func (a *ZapAdapter) Info(msg string, kv ...any)  { a.l.Infow(msg, kv...) }
```

logrus:

```go
import "github.com/sirupsen/logrus"

type LogrusAdapter struct{ l *logrus.Logger }

func NewLogrus(l *logrus.Logger) *LogrusAdapter { return &LogrusAdapter{l: l} }

func (a *LogrusAdapter) emit(level logrus.Level, msg string, kv []any) {
    fields := logrus.Fields{}
    for i := 0; i+1 < len(kv); i += 2 {
        if k, ok := kv[i].(string); ok {
            fields[k] = kv[i+1]
        }
    }
    a.l.WithFields(fields).Log(level, msg)
}

func (a *LogrusAdapter) Error(msg string, kv ...any) { a.emit(logrus.ErrorLevel, msg, kv) }
func (a *LogrusAdapter) Debug(msg string, kv ...any) { a.emit(logrus.DebugLevel, msg, kv) }
func (a *LogrusAdapter) Info(msg string, kv ...any)  { a.emit(logrus.InfoLevel, msg, kv) }
```

zerolog:

```go
import "github.com/rs/zerolog"

type ZerologAdapter struct{ l zerolog.Logger }

func NewZerolog(l zerolog.Logger) *ZerologAdapter { return &ZerologAdapter{l: l} }

func (a *ZerologAdapter) emit(ev *zerolog.Event, msg string, kv []any) {
    for i := 0; i+1 < len(kv); i += 2 {
        if k, ok := kv[i].(string); ok {
            ev = ev.Interface(k, kv[i+1])
        }
    }
    ev.Msg(msg)
}

func (a *ZerologAdapter) Error(msg string, kv ...any) { a.emit(a.l.Error(), msg, kv) }
func (a *ZerologAdapter) Debug(msg string, kv ...any) { a.emit(a.l.Debug(), msg, kv) }
func (a *ZerologAdapter) Info(msg string, kv ...any)  { a.emit(a.l.Info(), msg, kv) }
```

Drop one of these into your codebase, pass an instance to `WithLogger`,
and you're done. ewrap stays free of those dependencies.

## Recovery suggestions in log output

When you attach a `RecoverySuggestion`, `(*Error).Log` automatically
expands it into structured fields:

```go
err := ewrap.New("DB unreachable",
    ewrap.WithLogger(logger),
    ewrap.WithRecoverySuggestion(&ewrap.RecoverySuggestion{
        Message:       "Verify pool sizing and credentials.",
        Actions:       []string{"reset pool", "rotate creds"},
        Documentation: "https://runbooks.example.com/db",
    }))

err.Log()
// Fields emitted: error, stack, recovery_message, recovery_actions,
// recovery_documentation
```

## Best practices

- **Set the logger near the root** so wraps inherit it, instead of
  threading `WithLogger` through every layer.
- **Don't log inside libraries** — return the error and let the caller
  decide. `WithLogger` is for application-layer code.
- **Use slog directly** for new projects unless you've already standardised
  on another logger. `LogValuer` gives you fully structured output with no
  adapter at all.
- **Keep adapters in a single internal package** in your own repo so all
  of your services share the same logger choice without ewrap having to
  pick one.
