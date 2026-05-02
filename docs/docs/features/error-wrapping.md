# Error Wrapping

`Wrap` and `Wrapf` add a layer of context to an existing error while
preserving the cause chain. Each wrap captures its own stack frames, and
inherited metadata stays attached so log records remain useful all the way
out.

## Signature

```go
func Wrap(err error, msg string, opts ...Option) *Error
func Wrapf(err error, format string, args ...any) *Error
```

Both return `nil` if `err` is `nil`, so you can call them unconditionally:

```go
return ewrap.Wrap(maybeErr, "syncing replicas") // nil-safe
```

## Basic usage

```go
if err := validateInput(data); err != nil {
    return ewrap.Wrap(err, "input validation failed")
}
```

The returned `*Error`:

- Has its own `Error()` text: `"input validation failed: <inner.Error()>"`.
- Holds the inner error as its `Cause()`, so `errors.Unwrap`, `errors.Is`,
  and `errors.As` walk through it.
- Captures a fresh stack at the wrap site.

## Stack semantics

```go
root := db.Ping()                    // io error from net/http
inner := ewrap.Wrap(root, "ping db") // captures wrap site A
outer := ewrap.Wrap(inner, "boot")   // captures wrap site B

inner.Stack() // shows where Wrap was called for `inner`
outer.Stack() // shows where Wrap was called for `outer`
```

`outer.Stack()` and `inner.Stack()` are independent. To see the full
chain in one shot, use the verbose verb:

```go
fmt.Printf("%+v\n", outer)
// outer message
// stack of outer
```

If you need the inner's frames too, walk the chain:

```go
for cur := error(outer); cur != nil; cur = errors.Unwrap(cur) {
    var ec *ewrap.Error
    if errors.As(cur, &ec) {
        fmt.Println(ec.Stack())
    }
}
```

## Wrapping inherits typed fields

When the inner error is a `*Error`, the wrapper inherits:

- `metadata` (cloned via `maps.Clone` so wrapper writes don't mutate the inner)
- `errorContext`, `recovery`, `retry`
- `observer`, `logger`
- `httpStatus`, `retryable`

You can override any of these by passing the corresponding option to `Wrap`.

## Wrapping standard errors

```go
ewrap.Wrap(io.EOF, "reading body")
ewrap.Wrap(sql.ErrNoRows, "loading user")
```

These work like any other wrap; `errors.Is(err, io.EOF)` returns `true`
and serializers walk the cause chain via `errors.Unwrap`.

## `Wrapf` — formatted

```go
return ewrap.Wrapf(err, "loading row %d for tenant %s", id, tenantID)
```

If you need the wrapped error to participate in `%w` semantics, pass it
through `Newf` instead, or wrap explicitly:

```go
ewrap.Newf("loading row %d: %w", id, dbErr)
```

## Adding context while wrapping

```go
return ewrap.Wrap(err, "payment processing failed",
    ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityCritical),
    ewrap.WithHTTPStatus(http.StatusBadGateway),
    ewrap.WithRetryable(true),
    ewrap.WithLogger(logger),
).
    WithMetadata("amount", amount).
    WithMetadata("currency", "USD").
    WithMetadata("processor", "stripe")
```

## Conditional wrapping

```go
err := db.Query(...)
switch {
case errors.Is(err, sql.ErrNoRows):
    return ewrap.Wrap(err, "record not found",
        ewrap.WithContext(ctx, ewrap.ErrorTypeNotFound, ewrap.SeverityWarning),
        ewrap.WithHTTPStatus(http.StatusNotFound))
case errors.Is(err, sql.ErrConnDone):
    return ewrap.Wrap(err, "database connection lost",
        ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical),
        ewrap.WithRetryable(true))
case err != nil:
    return ewrap.Wrap(err, "database operation failed",
        ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError))
}
```

## Wrapping inside helpers — `WrapSkip`

If you wrap inside a helper, the captured stack starts in the helper rather
than at the call site:

```go
func wrapDB(err error, msg string) *ewrap.Error {
    // BAD: stack starts here
    return ewrap.Wrap(err, msg, ewrap.WithContext(...))
}
```

Use `WrapSkip(skip, ...)` to advance past the helper frames:

```go
func wrapDB(err error, msg string) *ewrap.Error {
    return ewrap.WrapSkip(1, err, msg, ewrap.WithContext(...))
}
```

The same pattern works for `New` via `NewSkip`.

## Best practices

- **One wrap per layer.** Don't wrap the same error twice in the same
  function; that just doubles the message.
- **Add information, not noise.** A useful wrap message points at *what
  this layer was doing*, not just that it failed.
- **Use `errors.Is/As` for branching, not string matching** on the rendered
  message.
- **Don't wrap simple validation errors** in tight loops if you don't add
  context — return the inner error directly.

## Performance

- A single `Wrap` allocates the `*Error` struct plus the stack PCs slice
  (~2 allocations).
- Inherited metadata is cloned shallowly via `maps.Clone`.
- `(*Error).Error()` and `Stack()` are cached on first call; subsequent
  reads on the wrapped error are lock-free.
