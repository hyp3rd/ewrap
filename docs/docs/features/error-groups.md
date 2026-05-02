# Error Groups

`ErrorGroup` aggregates multiple errors into a single `error`-implementing
value. Use it for validation passes, batch operations, fan-out fan-in
goroutine work, or anywhere you'd otherwise return the first error and
silently drop the rest.

## Quick start

```go
eg := ewrap.NewErrorGroup()
eg.Add(validate(req))
eg.Add(persist(req))
eg.Add(notify(req))

if err := eg.ErrorOrNil(); err != nil {
    return err
}
```

`Add(nil)` is a no-op, so you can call it unconditionally.

## Pooled allocation

For high-throughput paths, reuse `ErrorGroup` instances via `ErrorGroupPool`:

```go
pool := ewrap.NewErrorGroupPool(4) // initial slice capacity

eg := pool.Get()
defer eg.Release() // returns it to the pool, cleared

eg.Add(err1)
eg.Add(err2)
```

`Release()` clears the underlying slice (preserving capacity) and puts the
group back in the pool. Calling `Release()` on a non-pooled group is a no-op.

## Reading the group

```go
eg.HasErrors()              // bool
len(eg.Errors())            // count (clones the slice)
eg.Error()                  // formatted "N errors occurred:\n..." text
eg.ErrorOrNil()             // returns eg if non-empty, else nil
eg.Join()                   // errors.Join semantics — single, multi-cause error
```

`Errors()` returns a defensive copy via `slices.Clone` so callers can't
mutate the group's internal state.

## `errors.Is` / `errors.As` over a group

`Join()` returns a value compatible with `errors.Join`, so the stdlib walks
through every member:

```go
joined := eg.Join()

errors.Is(joined, sql.ErrNoRows)   // true if any member matches
errors.Is(joined, io.EOF)          // ditto
errors.As(joined, &myCustomError)  // first matching member fills target
```

## Concurrent use

`Add`, `HasErrors`, `Error`, `Errors`, `Join`, and `Clear` are all
goroutine-safe via an internal `sync.RWMutex`. Typical fan-out fan-in:

```go
eg := pool.Get()
defer eg.Release()

var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func(it Item) {
        defer wg.Done()
        eg.Add(process(it))
    }(item)
}
wg.Wait()

if err := eg.Join(); err != nil {
    return err
}
```

## Serialization

`ErrorGroup` implements `json.Marshaler` and `yaml.Marshaler`, plus explicit
`ToJSON()` / `ToYAML()` methods for callers that want the result as a string.

```go
jsonStr, _ := eg.ToJSON()
yamlStr, _ := eg.ToYAML()

bytes, _ := json.Marshal(eg) // also works
```

The serialized payload includes:

```json
{
  "error_count": 2,
  "timestamp": "2026-05-02T10:11:12Z",
  "errors": [
    {
      "message": "validation failed: missing field 'email'",
      "type": "ewrap",
      "stack_trace": [
        {"function": "...", "file": "...", "line": 42, "pc": 12345}
      ],
      "metadata": {"field": "email"},
      "cause": null
    },
    {
      "message": "EOF",
      "type": "standard"
    }
  ]
}
```

The cause chain is preserved for both `*Error` members and standard wrapped
errors (the serializer walks them via `errors.Unwrap`), so transport
consumers see the full picture.

## Patterns

### Validation pass

```go
func validateOrder(o Order) error {
    eg := pool.Get()
    defer eg.Release()

    if o.Customer == "" {
        eg.Add(ewrap.New("missing customer",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError)))
    }
    if o.Total <= 0 {
        eg.Add(ewrap.New("invalid total",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError)))
    }

    return eg.ErrorOrNil()
}
```

### Best-effort cleanup

```go
func close(resources []io.Closer) error {
    eg := pool.Get()
    defer eg.Release()

    for _, r := range resources {
        eg.Add(r.Close())
    }
    return eg.Join() // single error containing every Close failure
}
```

### Mixed wrapping

```go
eg.Add(ewrap.Wrap(httpErr, "fetching user",
    ewrap.WithHTTPStatus(http.StatusBadGateway)))
eg.Add(ewrap.Wrap(dbErr, "loading order"))
eg.Add(io.EOF) // raw stdlib error mixes fine
```

The serializer normalises all members into the same shape, so consumers
don't need to special-case ewrap vs standard errors.

## Performance

| Operation | ns/op | allocs |
| --- | ---: | ---: |
| `Add` (non-nil) | ~30 | 0 (steady state) |
| `Get` from pool | ~50 | 0 (warm pool) |
| `Error()` (formatted) | varies | 1 (builder) |
| `ToJSON` (10 entries) | ~10 µs | ~30 |

The pool eliminates the per-error allocation of the slice header in
hot paths.
