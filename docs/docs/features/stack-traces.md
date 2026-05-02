# Stack Traces

ewrap captures a stack trace on every constructor call and exposes it via
`(*Error).Stack()`, `fmt.Printf("%+v", err)`, and an iterator API. Captures
are tunable; formatted output is cached so you can read it freely.

## What gets captured

- `runtime.Callers` records up to **32** program counters by default.
- The capture skips the ewrap entry point so the first visible frame is
  your call to `New` / `Wrap` / `Newf` / `Wrapf`.
- Internal ewrap frames are filtered from the rendered output. Test files
  in the same package are allowed through so the library's own tests still
  produce useful traces.

## Reading the stack

### As a formatted string

```go
fmt.Println(err.Stack())
```

Output (one frame per line):

```text
/path/to/repo/db.go:42 - example.com/repo/db.queryUser
/path/to/repo/handlers.go:71 - example.com/repo/handlers.GetProfile
...
```

`Stack()` formats and caches the result on first call via `sync.Once`;
subsequent calls return the cached string with no allocations.

### As frames you can walk

```go
for it := err.GetStackIterator(); it.HasNext(); {
    f := it.Next()
    fmt.Printf("%s:%d %s (pc=%x)\n", f.File, f.Line, f.Function, f.PC)
}
```

`StackIterator` supports `Next`, `HasNext`, `Reset`, `Frames` (remaining
slice), and `AllFrames` (full slice).

For a one-shot snapshot:

```go
frames := err.GetStackFrames()
```

`StackFrame` is JSON/YAML-tagged so it serializes cleanly.

### Via `%+v`

```go
fmt.Printf("%+v\n", err)
// boom
// /path/to/foo.go:12 - example.com/foo.do
// ...
```

`(*Error).Format` implements `fmt.Formatter`. Other verbs:

- `%s`, `%v` — the error message only
- `%q` — quoted message
- `%+v` — message plus formatted stack

## Tuning capture depth

The default depth (32) is plenty for most stacks. Override with
`WithStackDepth`:

```go
ewrap.New("boom", ewrap.WithStackDepth(8))   // shallower
ewrap.New("boom", ewrap.WithStackDepth(0))   // disable capture entirely
ewrap.New("boom", ewrap.WithStackDepth(128)) // deeper
```

Setting depth to 0 returns a `*Error` with `len(err.stack) == 0` and an
empty `Stack()`. Useful for hot-path errors you know will never need a
trace.

## Skipping helper frames

If you call `New` or `Wrap` from a thin helper, the captured stack begins
inside the helper rather than at the caller. Use the `Skip` variants to
advance past those frames:

```go
func ErrInvalid(field string) *ewrap.Error {
    return ewrap.NewSkip(1, "invalid input").
        WithMetadata("field", field)
}

return ErrInvalid("email") // stack starts at the caller of ErrInvalid
```

`WrapSkip` is the wrap analogue:

```go
func wrapDB(err error, msg string) *ewrap.Error {
    return ewrap.WrapSkip(1, err, msg)
}
```

## How wrap chains compose

Each `Wrap` captures its own stack, so deep chains don't lose information:

```go
root  := io.EOF
inner := ewrap.Wrap(root, "ping db")  // stack A
outer := ewrap.Wrap(inner, "boot")    // stack B

outer.Stack() // shows where outer was created
inner.Stack() // shows where inner was created
```

To assemble a full multi-layer trace, walk the chain:

```go
for cur := error(outer); cur != nil; cur = errors.Unwrap(cur) {
    var ec *ewrap.Error
    if errors.As(cur, &ec) {
        fmt.Println("---")
        fmt.Println(ec.Stack())
    }
}
```

## Serialization

`(*Error).ToJSON` includes the formatted stack by default; pass
`WithStackTrace(false)` to omit it:

```go
jsonStr, _ := err.ToJSON(ewrap.WithStackTrace(false))
```

For `ErrorGroup`, each member's stack frames serialize as a typed
`[]StackFrame` slice (`stack_trace` field) — easy to render in dashboards.

## Performance

| Operation | ns/op | allocs |
| --- | ---: | ---: |
| `runtime.Callers` (depth 32) | ~860 | 1 |
| `Stack()` first call (formatting + filter) | ~2,500 | 1 |
| `Stack()` cached call | **1.7** | **0** |

Capture happens once at construction. Formatting is paid once per error.
After that, `Stack()`, `%+v`, and `LogValue` all read the cached string.

## Internal frame filter

The filter recognises a frame as ewrap-internal when:

1. The function path starts with `runtime.`, **or**
2. The function path starts with `github.com/hyp3rd/ewrap.` AND the file
   does NOT end in `_test.go`.

That second clause keeps ewrap's own tests visible in their own traces
(useful for debugging the library) while hiding the library's machinery
from end-user code.

If you fork ewrap under a different module path, update the prefix in
`isInternalFrame` (see [errors.go](https://github.com/hyp3rd/ewrap/blob/main/errors.go)).
