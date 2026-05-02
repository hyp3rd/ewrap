# `ewrap/slog` — slog adapter subpackage

A 30-line subpackage that lets a stdlib `*log/slog.Logger` satisfy
`ewrap.Logger`. Stdlib-only — no extra deps.

## Install

```bash
go get github.com/hyp3rd/ewrap/slog
```

## Usage

```go
import (
    stdslog "log/slog"
    "os"

    "github.com/hyp3rd/ewrap"
    ewrapslog "github.com/hyp3rd/ewrap/slog"
)

handler := stdslog.NewJSONHandler(os.Stdout, &stdslog.HandlerOptions{
    Level: stdslog.LevelDebug,
})
logger := ewrapslog.New(stdslog.New(handler))

err := ewrap.New("boom", ewrap.WithLogger(logger))
err.Log()
```

## API

```go
type Adapter struct{ /* unexported */ }

func New(logger *slog.Logger) *Adapter

func (a *Adapter) Error(msg string, keysAndValues ...any)
func (a *Adapter) Debug(msg string, keysAndValues ...any)
func (a *Adapter) Info(msg string, keysAndValues ...any)
```

`Adapter` is a thin shim — each method forwards to the wrapped logger
with the same arguments. `keysAndValues` is the standard alternating
key/value convention; `*slog.Logger` accepts it natively.

## When you need this vs when you don't

You need the adapter when you're calling `(*Error).Log` (or anything else
that takes an `ewrap.Logger`):

```go
err := ewrap.New("boom", ewrap.WithLogger(ewrapslog.New(slogger)))
err.Log() // adapter forwards to slogger.Error(...)
```

You **don't** need the adapter when you're logging via slog directly.
`*Error` implements `slog.LogValuer`, so this works without any adapter:

```go
slogger.Error("payment failed", "err", err)
// emits message, type, severity, request_id, cause, metadata as
// structured fields — see fmt.Formatter & slog
```

See [fmt.Formatter & slog](format-and-slog.md) for the `LogValuer`
details.

## Why a subpackage?

`*log/slog.Logger` is in the stdlib, so the adapter has no third-party
dependencies. It's still a separate subpackage so importing only `ewrap`
doesn't pull in `log/slog` for projects that don't use it (rare but
possible — embedded targets, for example).
