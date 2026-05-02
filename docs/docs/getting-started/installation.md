# Installation

## Requirements

- Go **1.25 or later**. ewrap uses `maps.Clone`, `slices.Clone`,
  range-over-int, `(*testing.B).Loop`, and `sync.WaitGroup.Go`.

## Install the core module

```bash
go get github.com/hyp3rd/ewrap
```

This pulls in two transitive deps and nothing else:

- `gopkg.in/yaml.v3` — YAML serialization
- `github.com/goccy/go-json` — fast JSON encoder used on the serialization hot path

That's the entire footprint of the core module.

## Subpackages

ewrap ships small, opt-in subpackages. Importing them is what enables their
features; if you don't import them, you don't pay for them.

### Circuit breaker

```bash
go get github.com/hyp3rd/ewrap/breaker
```

```go
import "github.com/hyp3rd/ewrap/breaker"
```

The breaker is a self-contained package with no dependency on the parent
`ewrap` module — you can use it on its own.

### `slog` adapter

```bash
go get github.com/hyp3rd/ewrap/slog
```

```go
import ewrapslog "github.com/hyp3rd/ewrap/slog"
```

A 30-line adapter that lets a stdlib `*slog.Logger` satisfy `ewrap.Logger`.
Stdlib-only — no extra deps.

## Logger adapters for other libraries

ewrap intentionally does **not** bundle adapters for zap, zerolog, logrus, or
glog. The `ewrap.Logger` interface has three methods; a working adapter is
under ten lines:

```go
type zapAdapter struct{ l *zap.Logger }

func (a *zapAdapter) Error(msg string, kv ...any) { a.l.Sugar().Errorw(msg, kv...) }
func (a *zapAdapter) Debug(msg string, kv ...any) { a.l.Sugar().Debugw(msg, kv...) }
func (a *zapAdapter) Info(msg string, kv ...any)  { a.l.Sugar().Infow(msg, kv...) }
```

Pass an instance to `ewrap.WithLogger(adapter)`.

## Verify the install

```bash
cat <<'EOF' > smoke.go
package main

import (
    "fmt"

    "github.com/hyp3rd/ewrap"
)

func main() {
    err := ewrap.New("ewrap is installed")
    fmt.Printf("%+v\n", err)
}
EOF

go run smoke.go
```

The expected output is `"ewrap is installed"` followed by the captured stack
trace (filtered to your code only).
