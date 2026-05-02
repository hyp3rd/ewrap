# Testing Error Handling

Patterns for testing code that produces `*ewrap.Error` values.

## Asserting identity with `errors.Is`

Always test by **identity**, not by string match. ewrap respects the
stdlib chain via `Unwrap()`, so `errors.Is` works through `Wrap`:

```go
sentinel := errors.New("not found")

err := layered() // returns ewrap.Wrap(sentinel, "...")

if !errors.Is(err, sentinel) {
    t.Fatalf("expected sentinel in chain, got %v", err)
}
```

Strings are fragile and break the moment somebody adds a layer.

## Asserting type with `errors.As`

For typed errors:

```go
var ec *ewrap.Error
if !errors.As(err, &ec) {
    t.Fatal("expected an ewrap.Error in the chain")
}

if ec.GetErrorContext().Type != ewrap.ErrorTypeValidation {
    t.Errorf("expected validation type, got %v", ec.GetErrorContext().Type)
}
```

## Asserting attached state

`HTTPStatus`, `IsRetryable`, and the typed accessors are the right tools
in tests:

```go
if got := ewrap.HTTPStatus(err); got != http.StatusBadGateway {
    t.Errorf("HTTP status: got %d, want %d", got, http.StatusBadGateway)
}

if !ewrap.IsRetryable(err) {
    t.Error("expected retryable error")
}

if rs := ec.Recovery(); rs == nil || rs.Message == "" {
    t.Error("expected non-empty recovery suggestion")
}
```

## Checking metadata

```go
val, ok := ec.GetMetadata("user_id")
if !ok || val != "u-1" {
    t.Errorf("user_id metadata: got %v (ok=%v)", val, ok)
}

// Or with type checking
if id, ok := ewrap.GetMetadataValue[string](ec, "user_id"); !ok || id != "u-1" {
    t.Errorf("user_id: got %q (ok=%v)", id, ok)
}
```

## Test loggers

Don't reach for a mocking framework — implement the three-method
interface inline. The test suite in this repo uses this pattern:

```go
type recordingLogger struct {
    mu     sync.Mutex
    logs   []entry
    calls  map[string]int
}

type entry struct {
    Level string
    Msg   string
    Args  []any
}

func (l *recordingLogger) Error(msg string, kv ...any) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.logs = append(l.logs, entry{"error", msg, kv})
    l.calls["error"]++
}
// Debug, Info similarly

func TestSomethingLogsErrors(t *testing.T) {
    l := &recordingLogger{calls: map[string]int{}}
    err := callee(ewrap.WithLogger(l))
    err.Log()

    if l.calls["error"] != 1 {
        t.Errorf("expected 1 error log, got %d", l.calls["error"])
    }
}
```

## Test observers

The breaker subpackage uses an analogous pattern:

```go
type recordingObserver struct {
    mu          sync.Mutex
    transitions []transition
}

func (o *recordingObserver) RecordTransition(name string, from, to breaker.State) {
    o.mu.Lock()
    defer o.mu.Unlock()
    o.transitions = append(o.transitions, transition{name, from, to})
}
```

Same shape works for `ewrap.Observer.RecordError`.

## Concurrency tests

Run race-detected concurrent stress tests on anything that holds shared
state. ewrap's own suite includes:

```go
func TestConcurrentMetadata(t *testing.T) {
    t.Parallel()

    err := ewrap.New("test")

    var wg sync.WaitGroup
    for i := range 100 {
        wg.Go(func() { _ = err.WithMetadata(fmt.Sprintf("k%d", i), i) })
        wg.Go(func() { _, _ = err.GetMetadata(fmt.Sprintf("k%d", i)) })
    }
    wg.Wait()
}
```

Run with `go test -race ./...` to catch real races.

## Deep-chain tests

Verify your code survives long wrap chains — easy to construct:

```go
func TestDeepChain(t *testing.T) {
    var err error = errors.New("root")
    for i := range 200 {
        err = ewrap.Wrap(err, fmt.Sprintf("layer-%d", i))
    }

    if !errors.Is(err, errors.New("root")) {
        // would fail because errors.New gives unique identity each call
    }
}
```

(Use a sentinel `var` instead of inline `errors.New` for the assertion.)

## Fuzz tests

`(*Error).ToJSON` and `Newf` are good fuzz targets:

```go
func FuzzJSONRoundTrip(f *testing.F) {
    for _, seed := range []string{"", "boom", strings.Repeat("a", 1024)} {
        f.Add(seed)
    }
    f.Fuzz(func(t *testing.T, msg string) {
        err := ewrap.New(msg)
        s, jerr := err.ToJSON()
        if jerr != nil {
            t.Fatalf("ToJSON: %v", jerr)
        }
        var out ewrap.ErrorOutput
        if err := json.Unmarshal([]byte(s), &out); err != nil {
            t.Fatalf("invalid JSON: %v", err)
        }
        if out.Message != msg {
            t.Errorf("round-trip lost data: got %q, want %q", out.Message, msg)
        }
    })
}
```

## `t.Parallel()` is safe

ewrap is goroutine-safe by design — every test in this repo runs with
`t.Parallel()`. Add it to your tests too unless they mutate global state
(e.g. `runtime.MemProfileRate`, env vars).

## Testing the breaker

Pin the timeout to something tiny so you don't wait around:

```go
func TestBreakerOpens(t *testing.T) {
    t.Parallel()

    cb := breaker.New("test", 1, 10*time.Millisecond)

    cb.RecordFailure()
    if cb.State() != breaker.Open {
        t.Errorf("State: got %v, want Open", cb.State())
    }

    time.Sleep(15 * time.Millisecond)

    if !cb.CanExecute() {
        t.Error("expected breaker to allow execution after timeout")
    }
}
```

The transition observer fires synchronously, so you don't need to sleep
to wait for callbacks.

## Test fixtures

For shared sentinels and constants across a test suite, define them in a
`*_test.go` helper file:

```go
// test_helpers_test.go
package mypkg

import "errors"

const (
    msgValidation = "invalid input"
    msgNotFound   = "user not found"
)

var (
    errSentinel = errors.New("sentinel")
    errOther    = errors.New("other")
)
```

This pattern silences `goconst` and `err113` linters while keeping tests
readable.
