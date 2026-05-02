# Basic Examples

Drop-in patterns showing the core surface. Each example compiles against
the current API.

## Simple error

```go
package main

import (
    "fmt"

    "github.com/hyp3rd/ewrap"
)

func main() {
    err := ewrap.New("file not found")
    fmt.Println(err.Error())          // file not found
    fmt.Printf("%+v\n", err)          // file not found + stack
}
```

## Wrapping with context

```go
import (
    "context"
    "net/http"

    "github.com/hyp3rd/ewrap"
)

func loadUser(ctx context.Context, id string) (*User, error) {
    u, err := db.Get(ctx, id)
    if err != nil {
        return nil, ewrap.Wrap(err, "loading user",
            ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError),
            ewrap.WithHTTPStatus(http.StatusInternalServerError)).
            WithMetadata("user_id", id)
    }
    return u, nil
}
```

## `%w` in `Newf`

```go
import (
    "errors"
    "io"

    "github.com/hyp3rd/ewrap"
)

func read() error {
    err := ewrap.Newf("reading config: %w", io.EOF)

    fmt.Println(err.Error())              // reading config: EOF
    fmt.Println(errors.Is(err, io.EOF))   // true
    return err
}
```

## Validation accumulator

```go
func validate(o Order) error {
    eg := ewrap.NewErrorGroup()

    if o.Customer == "" {
        eg.Add(ewrap.New("customer is required",
            ewrap.WithContext(nil, ewrap.ErrorTypeValidation, ewrap.SeverityError)).
            WithMetadata("field", "customer"))
    }
    if o.Total <= 0 {
        eg.Add(ewrap.New("total must be positive",
            ewrap.WithContext(nil, ewrap.ErrorTypeValidation, ewrap.SeverityError)).
            WithMetadata("field", "total"))
    }

    return eg.ErrorOrNil()
}
```

## Pooled error group

```go
var pool = ewrap.NewErrorGroupPool(8)

func process(items []Item) error {
    eg := pool.Get()
    defer eg.Release()

    for _, it := range items {
        eg.Add(handle(it))
    }
    return eg.Join()
}
```

## Inspecting metadata

```go
err := ewrap.New("payment failed").
    WithMetadata("provider", "stripe").
    WithMetadata("attempt", 2)

if v, ok := err.GetMetadata("provider"); ok {
    fmt.Println(v) // stripe
}

if attempt, ok := ewrap.GetMetadataValue[int](err, "attempt"); ok {
    fmt.Println(attempt) // 2
}
```

## Walking the cause chain

```go
import "errors"

err := ewrap.Wrap(io.EOF, "reading body")

errors.Is(err, io.EOF)     // true
errors.Unwrap(err)         // io.EOF
err.Cause()                // io.EOF
```

## Logging with `slog` directly

`*Error` implements `slog.LogValuer`, so you can pass it as a structured
field with no adapter:

```go
import (
    "log/slog"
    "os"
)

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
err := ewrap.New("payment failed",
    ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityError))

logger.Error("payment failed", "err", err)
// {"level":"ERROR","msg":"payment failed","err":{"message":"payment failed",
//  "type":"external","severity":"error",...}}
```

## Logging via `(*Error).Log`

For loggers attached as `ewrap.Logger`:

```go
import ewrapslog "github.com/hyp3rd/ewrap/slog"

logger := ewrapslog.New(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

err := ewrap.New("payment failed",
    ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityError),
    ewrap.WithLogger(logger))
err.Log()
```

## Pretty-printing for humans

```go
fmt.Printf("%+v\n", err)
// payment failed
// /path/to/repo/billing.go:42 - example.com/repo/billing.charge
// /path/to/repo/handlers.go:71 - example.com/repo/handlers.Pay
// ...
```

## Circuit breaker basics

```go
import "github.com/hyp3rd/ewrap/breaker"

cb := breaker.New("payments", 3, time.Minute)

if !cb.CanExecute() {
    return ewrap.New("payments breaker open",
        ewrap.WithRetryable(true))
}

if err := charge(); err != nil {
    cb.RecordFailure()
    return ewrap.Wrap(err, "charging customer")
}

cb.RecordSuccess()
```

## Classifying for the wire

```go
func toResponse(w http.ResponseWriter, err error) {
    status := ewrap.HTTPStatus(err)
    if status == 0 {
        status = http.StatusInternalServerError
    }

    msg := err.Error()
    if e, ok := err.(*ewrap.Error); ok {
        msg = e.SafeError() // PII-redacted variant
    }

    http.Error(w, msg, status)
}
```

## Retry with classification

```go
func withRetry(ctx context.Context, op func() error) error {
    for attempt := 1; attempt <= 5; attempt++ {
        err := op()
        if err == nil {
            return nil
        }
        if !ewrap.IsRetryable(err) {
            return err
        }
        select {
        case <-time.After(backoff(attempt)):
        case <-ctx.Done():
            return ewrap.Wrap(ctx.Err(), "retry budget exhausted")
        }
    }
    return op()
}
```
