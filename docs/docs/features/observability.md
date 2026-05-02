# Observability

ewrap exposes a single, deliberately small `Observer` interface for
errors. The matching observer for the circuit breaker lives in the
[`breaker`](circuit-breaker.md) subpackage. Both are plain interfaces — wire
them to whatever metrics, tracing, or alerting backend you use.

## `ewrap.Observer`

```go
type Observer interface {
    RecordError(message string)
}
```

A single method. Implementations must be goroutine-safe because
`(*Error).Log` calls them synchronously from the calling goroutine.

## Attaching an observer

```go
type metricsObserver struct {
    counter *prometheus.CounterVec
}

func (m *metricsObserver) RecordError(message string) {
    m.counter.WithLabelValues(message).Inc()
}

err := ewrap.New("payment failed", ewrap.WithObserver(&metricsObserver{counter: errCounter}))
err.Log() // observer.RecordError("payment failed")
```

The observer reference is inherited by `Wrap` when the inner error is a
`*Error`, so attaching once at the root applies to every layer that's
later wrapped.

## Pairing with a logger

`Observer` and `Logger` are independent — you can attach either, both, or
neither. A common setup:

```go
err := ewrap.New("payment failed",
    ewrap.WithLogger(logger),       // structured log output
    ewrap.WithObserver(metrics),    // metric increment
)
err.Log()
```

`(*Error).Log` first calls `Observer.RecordError`, then writes the
structured log record. Either is a no-op if not configured.

## Tracing integration

You can wire OpenTelemetry, Datadog, or any other tracer through the same
interface:

```go
type otelObserver struct{ tracer trace.Tracer }

func (o *otelObserver) RecordError(message string) {
    span := trace.SpanFromContext(context.Background())
    if span.IsRecording() {
        span.RecordError(errors.New(message))
    }
}
```

A richer integration would attach a `context.Context` to the error via
`WithContext` and pull the active span out of it inside the observer.

## Why so minimal?

The interface deliberately carries only `message`. Anything richer would
mean ewrap dictating a particular metric label set, tracing API, or sample
rate. Keeping it tight means:

- Zero dependencies for observers.
- You can record whatever's relevant to *your* stack inside the
  implementation (`(*Error).Log` runs synchronously in the caller's
  goroutine, so you have access to its `context.Context` etc.).
- Substitution is trivial — wrap an existing observer to add sampling,
  filtering, or rate-limiting without touching ewrap.

If you want to observe the full structured payload, attach a `Logger`
instead. The logger receives the message, cause, stack, metadata, and
recovery fields all in one record.

## Circuit-breaker observability

The breaker subpackage has its own observer interface:

```go
import "github.com/hyp3rd/ewrap/breaker"

type breakerMetrics struct {
    state *prometheus.GaugeVec
}

func (m *breakerMetrics) RecordTransition(name string, from, to breaker.State) {
    m.state.WithLabelValues(name).Set(float64(to))
}

cb := breaker.NewWithObserver("payments", 5, 30*time.Second, &breakerMetrics{state: stateGauge})
```

See [Circuit Breaker](circuit-breaker.md) for details on transition
semantics. Importantly, transition callbacks fire **synchronously** after
the breaker lock is released, so they must not invoke the breaker
recursively.
