# Advanced Examples

Realistic scenarios combining multiple ewrap features. Each example is
self-contained and uses the current API.

## HTTP service with classification + retry

```go
package billing

import (
    "context"
    "net/http"
    "time"

    "github.com/hyp3rd/ewrap"
    "github.com/hyp3rd/ewrap/breaker"
)

type Service struct {
    upstream Upstream
    breaker  *breaker.Breaker
    logger   ewrap.Logger
}

func (s *Service) Charge(ctx context.Context, req ChargeRequest) (*Receipt, error) {
    if err := req.Validate(); err != nil {
        return nil, ewrap.Wrap(err, "invalid charge request",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityWarning),
            ewrap.WithHTTPStatus(http.StatusUnprocessableEntity),
            ewrap.WithSafeMessage("invalid charge request"),
            ewrap.WithLogger(s.logger))
    }

    if !s.breaker.CanExecute() {
        return nil, ewrap.New("payments breaker open",
            ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityWarning),
            ewrap.WithHTTPStatus(http.StatusServiceUnavailable),
            ewrap.WithRetryable(true),
            ewrap.WithRetry(3, 2*time.Second),
            ewrap.WithLogger(s.logger))
    }

    receipt, err := s.upstream.Charge(ctx, req)
    if err != nil {
        s.breaker.RecordFailure()

        return nil, ewrap.Wrap(err, "upstream charge",
            ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityError),
            ewrap.WithHTTPStatus(http.StatusBadGateway),
            ewrap.WithRetryable(true),
            ewrap.WithSafeMessage("payment provider error"),
            ewrap.WithRecoverySuggestion(&ewrap.RecoverySuggestion{
                Message:       "Retry after backoff; check provider status page.",
                Documentation: "https://runbooks.example.com/payments/upstream",
            }),
            ewrap.WithLogger(s.logger)).
            WithMetadata("provider", "stripe").
            WithMetadata("amount_cents", req.AmountCents)
    }

    s.breaker.RecordSuccess()
    return receipt, nil
}
```

The handler then translates uniformly:

```go
func (h *Handler) Charge(w http.ResponseWriter, r *http.Request) {
    receipt, err := h.svc.Charge(r.Context(), parseRequest(r))
    if err != nil {
        status := ewrap.HTTPStatus(err)
        if status == 0 {
            status = http.StatusInternalServerError
        }
        msg := err.Error()
        if e, ok := err.(*ewrap.Error); ok {
            msg = e.SafeError()
        }
        http.Error(w, msg, status)
        return
    }

    writeJSON(w, receipt)
}
```

## Retry middleware

```go
func WithRetry(max int, base time.Duration, op func(context.Context) error) func(context.Context) error {
    return func(ctx context.Context) error {
        delay := base

        for attempt := 1; attempt <= max; attempt++ {
            err := op(ctx)
            if err == nil {
                return nil
            }

            if !ewrap.IsRetryable(err) {
                return err
            }

            select {
            case <-time.After(delay):
                delay *= 2
            case <-ctx.Done():
                return ewrap.Wrap(ctx.Err(), "retry budget exhausted",
                    ewrap.WithRetryable(false))
            }
        }

        // last attempt — return whatever the op returns
        return op(ctx)
    }
}
```

`IsRetryable` walks the chain and consults `Temporary()` as a fallback,
so this middleware works with any error source — ewrap or stdlib.

## Validation middleware emitting structured 422

```go
func writeValidation(w http.ResponseWriter, err error) {
    eg, ok := err.(*ewrap.ErrorGroup)
    if !ok {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    body, _ := eg.ToJSON()
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusUnprocessableEntity)
    _, _ = w.Write([]byte(body))
}
```

The `ErrorGroup` JSON envelope already carries every member's `field`
metadata, so the API consumer gets a clean per-field error list with no
extra translation.

## Background worker with the breaker

```go
type Worker struct {
    queue   <-chan Job
    cb      *breaker.Breaker
    logger  ewrap.Logger
    obs     ewrap.Observer
}

func (w *Worker) Run(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case job := <-w.queue:
            if !w.cb.CanExecute() {
                w.requeue(job, time.Second)
                continue
            }

            if err := w.process(ctx, job); err != nil {
                w.cb.RecordFailure()

                wrapped := ewrap.Wrap(err, "processing job",
                    ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityError),
                    ewrap.WithLogger(w.logger),
                    ewrap.WithObserver(w.obs)).
                    WithMetadata("job_id", job.ID)
                wrapped.Log()

                if ewrap.IsRetryable(wrapped) {
                    w.requeue(job, backoff(job.Attempts))
                }
                continue
            }

            w.cb.RecordSuccess()
        }
    }
}
```

A single `WithLogger`/`WithObserver` near the construction site
propagates to wraps via inheritance.

## Fan-out fan-in with `ErrorGroup`

```go
var pool = ewrap.NewErrorGroupPool(16)

func sweep(ctx context.Context, ids []string) error {
    eg := pool.Get()
    defer eg.Release()

    var wg sync.WaitGroup
    for _, id := range ids {
        wg.Add(1)
        go func(id string) {
            defer wg.Done()

            if err := refresh(ctx, id); err != nil {
                eg.Add(ewrap.Wrap(err, "refresh failed",
                    ewrap.WithContext(ctx, ewrap.ErrorTypeNetwork, ewrap.SeverityWarning),
                    ewrap.WithRetryable(true)).
                    WithMetadata("id", id))
            }
        }(id)
    }
    wg.Wait()

    return eg.Join() // single error containing every failure
}
```

Each member error preserves its own stack trace, metadata, and HTTP
status; the aggregator returned by `Join()` is `errors.Is`-walkable.

## Custom domain factory

```go
package billing

import (
    "context"
    "net/http"
    "time"

    "github.com/hyp3rd/ewrap"
)

type Code int

const (
    CodeUnknown Code = iota
    CodeCardDeclined
    CodeRateLimited
)

// New constructs a billing-specific error. NewSkip(1, ...) advances the
// captured stack past this helper so the trace begins at the caller.
func New(ctx context.Context, code Code, msg string) *ewrap.Error {
    base := ewrap.NewSkip(1, msg,
        ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityWarning))

    switch code {
    case CodeCardDeclined:
        return base.
            WithMetadata("billing_code", code).
            WithContext(&ewrap.ErrorContext{
                Type:     ewrap.ErrorTypeExternal,
                Severity: ewrap.SeverityWarning,
            })
    case CodeRateLimited:
        return base.
            WithMetadata("billing_code", code).
            WithMetadata("retry_after_s", 30)
    }

    return base.WithMetadata("billing_code", code)
}
```

Usage:

```go
return billing.New(ctx, billing.CodeRateLimited, "rate limited at provider")
```

## OpenTelemetry observer

```go
import "go.opentelemetry.io/otel/trace"

type otelObserver struct {
    tracer trace.Tracer
    ctx    context.Context // captured on construction
}

func (o *otelObserver) RecordError(message string) {
    span := trace.SpanFromContext(o.ctx)
    if !span.IsRecording() {
        return
    }
    span.RecordError(errors.New(message))
}

err := ewrap.New("payment failed",
    ewrap.WithObserver(&otelObserver{tracer: tracer, ctx: ctx}))
err.Log() // span event recorded
```

For a richer integration, walk the metadata in the observer and attach
each as a span attribute.

## Test fixtures

```go
package billing

import (
    "errors"
    "testing"

    "github.com/hyp3rd/ewrap"
)

var errFakeUpstream = errors.New("fake upstream failure")

func TestChargeWrapsUpstreamError(t *testing.T) {
    t.Parallel()

    svc := &Service{upstream: stubUpstream{err: errFakeUpstream}}
    _, err := svc.Charge(t.Context(), validRequest())

    if !errors.Is(err, errFakeUpstream) {
        t.Fatalf("expected upstream error in chain, got %v", err)
    }

    if got := ewrap.HTTPStatus(err); got != http.StatusBadGateway {
        t.Errorf("HTTP status: got %d, want %d", got, http.StatusBadGateway)
    }

    if !ewrap.IsRetryable(err) {
        t.Error("expected upstream error to be retryable")
    }
}
```

`errors.Is`, `ewrap.HTTPStatus`, and `ewrap.IsRetryable` are the right
assertions here — none of them touch string-formatted messages.
