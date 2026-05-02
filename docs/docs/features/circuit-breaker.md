# Circuit Breaker (`ewrap/breaker` subpackage)

The classic circuit-breaker pattern, implemented in a small, self-contained
subpackage. It does **not** depend on the parent `ewrap` module — you can
use it on its own, and consumers who only want error wrapping don't pay
for it.

## States

```text
┌────────┐  failures ≥ max     ┌──────┐  timeout elapsed   ┌───────────┐
│ Closed │ ───────────────────►│ Open │ ─────────────────► │ Half-Open │
└────────┘                     └──────┘                    └─────┬─────┘
     ▲                                                           │
     │                       success                             │
     └───────────────────────────────────────────────────────────┘
                              failure
                                ↓
                              Open
```

| State | Behaviour |
| --- | --- |
| `Closed` | Calls pass through. Failures increment a counter. |
| `Open` | Calls are rejected fast. After `timeout` elapses, the next `CanExecute` flips state to `HalfOpen`. |
| `HalfOpen` | A single probe call is allowed. Success closes the breaker; failure re-opens it. |

## Quick start

```go
import "github.com/hyp3rd/ewrap/breaker"

cb := breaker.New("payments", 5, 30*time.Second)

if !cb.CanExecute() {
    return ewrap.New("payments breaker open",
        ewrap.WithRetryable(true))
}

if err := charge(req); err != nil {
    cb.RecordFailure()
    return ewrap.Wrap(err, "charging customer")
}

cb.RecordSuccess()
```

## API

```go
func New(name string, maxFailures int, timeout time.Duration) *Breaker
func NewWithObserver(name string, maxFailures int, timeout time.Duration, obs Observer) *Breaker

func (cb *Breaker) Name() string
func (cb *Breaker) State() State
func (cb *Breaker) CanExecute() bool
func (cb *Breaker) RecordFailure()
func (cb *Breaker) RecordSuccess()
func (cb *Breaker) OnStateChange(callback func(name string, from, to State))
func (cb *Breaker) SetObserver(obs Observer)
```

States and the observer interface:

```go
type State int
const (
    Closed State = iota
    Open
    HalfOpen
)

func (s State) String() string // "closed", "open", "half-open", "unknown"

type Observer interface {
    RecordTransition(name string, from, to State)
}
```

## Observability

Pass an `Observer` at construction time or with `SetObserver`:

```go
type metrics struct {
    gauge *prometheus.GaugeVec
}

func (m *metrics) RecordTransition(name string, from, to breaker.State) {
    m.gauge.WithLabelValues(name, to.String()).Set(1)
}

cb := breaker.NewWithObserver("payments", 5, 30*time.Second, &metrics{gauge: stateGauge})
```

`OnStateChange` registers a callback that fires for the same events as the
observer:

```go
cb.OnStateChange(func(name string, from, to breaker.State) {
    log.Printf("breaker %s: %s -> %s", name, from, to)
})
```

### Synchronous, lock-released dispatch

Transition events (observer + callback) fire **synchronously** after the
breaker lock is released. The relevant guarantees:

1. The breaker is never holding its own mutex when your code runs.
2. Two transitions cannot interleave — observer/callback for transition A
   completes before transition B begins.
3. Your callbacks must **not** invoke the breaker recursively (would
   deadlock).

There is no fire-and-forget goroutine — earlier versions spawned one per
transition, which would have allowed unbounded goroutine growth under load.

## Concurrency

`CanExecute`, `RecordFailure`, `RecordSuccess`, `State`, `OnStateChange`,
and `SetObserver` are all goroutine-safe. The breaker uses a single
`sync.Mutex` and the `Open → HalfOpen` transition is atomic.

A typical hot-path use:

```go
for range workers {
    go func() {
        for req := range jobs {
            if !cb.CanExecute() {
                jobs <- req // requeue / drop
                continue
            }

            if err := process(req); err != nil {
                cb.RecordFailure()
                continue
            }

            cb.RecordSuccess()
        }
    }()
}
```

## Pairing with `ewrap`

The breaker has no compile-time dependency on `ewrap`, but the two compose
naturally — a tripped breaker is the canonical place to return a
retryable, well-classified error:

```go
if !cb.CanExecute() {
    return ewrap.New("payments breaker open",
        ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityWarning),
        ewrap.WithHTTPStatus(http.StatusServiceUnavailable),
        ewrap.WithRetryable(true),
        ewrap.WithRetry(3, 5*time.Second))
}
```

Downstream callers can then use `ewrap.IsRetryable(err)` and
`ewrap.HTTPStatus(err)` to decide what to do.

## Performance

| Benchmark | ns/op | B/op | allocs |
| --- | ---: | ---: | ---: |
| `RecordFailure` | ~33 | 0 | 0 |
| `ConcurrentOperations` (parallel CanExecute / Record) | ~200 | 0 | 0 |

Steady-state operations are allocation-free. Observer / callback dispatch
allocates only the closure passed to `OnStateChange`.

## When NOT to use a circuit breaker

- **Per-call retries** — use exponential backoff (e.g.
  `cenkalti/backoff`) instead. The breaker protects shared infrastructure
  from being overwhelmed; per-call backoff smooths a single client's
  request.
- **Validation errors** — those are not transient; failing fast is
  already the right answer.
- **Tests** — pin the timeout to something tiny (10 ms) and you won't
  need to mock the clock.
