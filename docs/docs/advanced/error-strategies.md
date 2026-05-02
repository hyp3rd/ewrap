# Error Handling Strategies

Patterns we've seen work well in production code that uses ewrap. None of
them are mandatory; pick the ones that fit your shape.

## Sentinel errors

For small, well-known failure modes, package-level sentinels remain the
right tool — even with ewrap. Use `errors.New` for the sentinel and `Wrap`
when you need to add layered context:

```go
var ErrNotFound = errors.New("not found")

func GetUser(ctx context.Context, id string) (*User, error) {
    u, err := store.Lookup(ctx, id)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ewrap.Wrap(ErrNotFound, "user lookup",
            ewrap.WithContext(ctx, ewrap.ErrorTypeNotFound, ewrap.SeverityWarning),
            ewrap.WithHTTPStatus(http.StatusNotFound)).
            WithMetadata("user_id", id)
    }
    return u, err
}
```

Callers branch on identity:

```go
if errors.Is(err, GetUser.ErrNotFound) {
    // 404 path
}
```

## Typed error structs

Use a typed struct when callers need to inspect structured fields, not
just identity. Embed `*ewrap.Error` or compose with it:

```go
type ValidationError struct {
    Field string
    Rule  string
    *ewrap.Error
}

func (e *ValidationError) Error() string { return e.Error.Error() }

func NewValidation(field, rule string) *ValidationError {
    return &ValidationError{
        Field: field,
        Rule:  rule,
        Error: ewrap.NewSkip(1, fmt.Sprintf("%s: %s", field, rule),
            ewrap.WithContext(nil, ewrap.ErrorTypeValidation, ewrap.SeverityWarning),
            ewrap.WithHTTPStatus(http.StatusUnprocessableEntity)),
    }
}
```

Callers extract via `errors.As`:

```go
var ve *ValidationError
if errors.As(err, &ve) {
    fmt.Println(ve.Field, ve.Rule)
}
```

## Top-of-handler classification

In an HTTP handler, the most useful place to classify is at the very top
of the error path. Pull `HTTPStatus` and pick a fallback:

```go
func toHTTPResponse(w http.ResponseWriter, err error) {
    status := ewrap.HTTPStatus(err)
    if status == 0 {
        status = http.StatusInternalServerError
    }

    msg := err.Error()
    if e, ok := err.(*ewrap.Error); ok {
        msg = e.SafeError() // PII-redacted variant for the wire
    }

    http.Error(w, msg, status)
}
```

## Retry with classification

```go
func withRetry(ctx context.Context, op func(context.Context) error) error {
    delay := 100 * time.Millisecond
    for attempt := 1; attempt <= 5; attempt++ {
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
    return op(ctx) // last try, return as-is
}
```

The retry loop is fully decoupled from the error itself — `IsRetryable`
walks the chain and consults `Temporary()` as a fallback, so it works
whether the error is yours or stdlib.

## Validation accumulator

When validating a request, you usually want to surface every problem at
once, not just the first:

```go
func validateOrder(ctx context.Context, o Order) error {
    eg := pool.Get()
    defer eg.Release()

    if o.Customer == "" {
        eg.Add(ewrap.New("customer is required",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError)).
            WithMetadata("field", "customer"))
    }
    if o.Total <= 0 {
        eg.Add(ewrap.New("total must be positive",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError)).
            WithMetadata("field", "total"))
    }

    return eg.ErrorOrNil()
}
```

The handler can then serialize the whole group via `(*ErrorGroup).ToJSON`
for a structured 422 response.

## Layered messages

Each layer's message answers "what was *this* layer doing":

```go
// db.go
func (s *Store) GetUser(ctx context.Context, id string) (*User, error) {
    row, err := s.db.QueryRowContext(ctx, "SELECT ...", id)
    if err != nil {
        return nil, ewrap.Wrap(err, "loading user row")
    }
    // ...
}

// service.go
func (s *Service) Profile(ctx context.Context, id string) (*Profile, error) {
    u, err := s.store.GetUser(ctx, id)
    if err != nil {
        return nil, ewrap.Wrap(err, "building profile",
            ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityError))
    }
    // ...
}

// handler.go
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
    p, err := h.service.Profile(r.Context(), chi.URLParam(r, "id"))
    if err != nil {
        toHTTPResponse(w, ewrap.Wrap(err, "GET /profile"))
        return
    }
    // ...
}
```

The final `Error()` reads top-down: `"GET /profile: building profile:
loading user row: <db driver error>"`.

## Don't wrap to "log and rethrow"

Wrapping just to log is an anti-pattern — you log the same error twice
when the eventual handler logs it. Either log **or** wrap, not both:

```go
// BAD
if err != nil {
    log.Printf("failed: %v", err)
    return ewrap.Wrap(err, "failed")
}

// GOOD
if err != nil {
    return ewrap.Wrap(err, "failed")
}
```

The exception is when wrap-and-log adds genuine signal (e.g. logging at a
boundary you control with metadata the caller can't see).

## Observer for metrics

Wire a metrics observer near the top so all errors flowing through your
service get counted, with labels derived from the error context:

```go
type counterObserver struct {
    counter *prometheus.CounterVec
}

func (o *counterObserver) RecordError(message string) {
    o.counter.WithLabelValues(message).Inc()
}

baseOpts := []ewrap.Option{
    ewrap.WithLogger(logger),
    ewrap.WithObserver(observer),
}

err := ewrap.New("checkout failed", baseOpts...)
err.Log() // observer counts + logger writes
```

For label cardinality control, derive the label from `ErrorContext.Type`
inside a richer observer:

```go
type typedCounterObserver struct {
    counter *prometheus.CounterVec
    err     *ewrap.Error // captured at construction
}

func (o *typedCounterObserver) RecordError(string) {
    if ec := o.err.GetErrorContext(); ec != nil {
        o.counter.WithLabelValues(ec.Type.String(), ec.Severity.String()).Inc()
    }
}
```
