# Performance Optimization

Understanding how to optimize error handling is crucial for maintaining high-performance applications. While error handling is essential, it shouldn't become a bottleneck in your application. Let's explore how ewrap helps you achieve efficient error handling and learn about optimization strategies.

## Understanding Error Handling Performance

When we talk about performance in error handling, we need to consider several aspects:

1. Memory allocation and garbage collection impact
2. CPU overhead from stack trace capture
3. Concurrency and contention in high-throughput scenarios
4. The cost of error formatting and logging
5. The impact of error wrapping chains

Let's explore how ewrap addresses each of these concerns and how you can optimize your error handling.

## Memory Management

One of the most significant performance impacts in error handling comes from memory allocations. ewrap uses several strategies to minimize this impact:

### Object Pooling

The Error Group pool is a prime example of how we can reduce memory pressure:

```go
// Create a pool with an appropriate size for your use case
pool := ewrap.NewErrorGroupPool(4)

func processItems(items []Item) error {
    // Get an error group from the pool
    eg := pool.Get()
    defer eg.Release()  // Return to pool when done

    for _, item := range items {
        if err := processItem(item); err != nil {
            eg.Add(err)
        }
    }

    return eg.Error()
}
```

This approach is particularly effective because:

1. It reduces garbage collection pressure
2. It minimizes memory fragmentation
3. It provides predictable memory usage patterns

### Pre-allocation Strategies

When dealing with metadata or formatting, pre-allocation can significantly improve performance:

```go
// Pre-allocate slices with expected capacity
func buildErrorContext(err error, expectedFields int) map[string]any {
    // Allocate map with expected size to avoid resizing
    context := make(map[string]any, expectedFields)

    if wrappedErr, ok := err.(*ewrap.Error); ok {
        // Pre-allocate string builder with reasonable capacity
        var builder strings.Builder
        builder.Grow(256)  // Reserve space for typical error message

        // Build context efficiently
        builder.WriteString("Error occurred in ")
        builder.WriteString(wrappedErr.Operation())
        context["message"] = builder.String()

        // Add other fields...
    }

    return context
}
```

## Stack Trace Optimization

Stack traces are expensive to capture, so ewrap implements several optimizations:

### Lazy Stack Capture

```go
type lazyStack struct {
    pcs    []uintptr
    frames runtime.Frames
    once   sync.Once
}

func (ls *lazyStack) Frames() runtime.Frames {
    ls.once.Do(func() {
        if ls.frames == nil {
            ls.frames = runtime.CallersFrames(ls.pcs)
        }
    })
    return ls.frames
}
```

### Stack Filtering

We filter out unnecessary frames to reduce memory usage and improve readability:

```go
func filterStack(stack []runtime.Frame) []runtime.Frame {
    filtered := make([]runtime.Frame, 0, len(stack))
    for _, frame := range stack {
        if shouldIncludeFrame(frame) {
            filtered = append(filtered, frame)
        }
    }
    return filtered
}

func shouldIncludeFrame(frame runtime.Frame) bool {
    // Skip runtime frames
    if strings.Contains(frame.File, "runtime/") {
        return false
    }
    // Skip ewrap internal frames
    if strings.Contains(frame.File, "ewrap/errors.go") {
        return false
    }
    return true
}
```

## Concurrency Optimization

In high-concurrency scenarios, efficient error handling becomes even more critical:

### Lock-Free Operations

Where possible, ewrap uses atomic operations instead of locks:

```go
type AtomicCounter struct {
    value int64
}

func (c *AtomicCounter) Increment() {
    atomic.AddInt64(&c.value, 1)
}

func (c *AtomicCounter) Get() int64 {
    return atomic.LoadInt64(&c.value)
}
```

### Minimizing Lock Contention

When locks are necessary, we minimize their scope:

```go
func (eg *ErrorGroup) Add(err error) {
    if err == nil {
        return
    }

    // Prepare the error outside the lock
    wrappedErr := prepareError(err)

    // Minimize critical section
    eg.mu.Lock()
    eg.errors = append(eg.errors, wrappedErr)
    eg.mu.Unlock()
}
```

## Formatting Performance

Error formatting can be expensive, especially for JSON/YAML conversion. Here's how to optimize it:

### Cached Formatting

For frequently accessed formats:

```go
type CachedError struct {
    err          *ewrap.Error
    jsonCache    atomic.Value
    yamlCache    atomic.Value
    cacheTimeout time.Duration
}

func (ce *CachedError) ToJSON() (string, error) {
    if cached := ce.jsonCache.Load(); cached != nil {
        cacheEntry := cached.(*formatCacheEntry)
        if !cacheEntry.isExpired() {
            return cacheEntry.data, nil
        }
    }

    // Format and cache the result
    json, err := ce.err.ToJSON()
    if err != nil {
        return "", err
    }

    ce.jsonCache.Store(&formatCacheEntry{
        data:    json,
        expires: time.Now().Add(ce.cacheTimeout),
    })

    return json, nil
}
```

### Efficient Buffer Usage

When formatting errors:

```go
// Pool of buffers for formatting
var bufferPool = sync.Pool{
    New: func() any {
        return new(bytes.Buffer)
    },
}

func formatError(err *ewrap.Error) string {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()

    // Use buffer for formatting
    buf.WriteString("Error: ")
    buf.WriteString(err.Error())
    buf.WriteString("\nStack:\n")
    buf.WriteString(err.Stack())

    return buf.String()
}
```

## Performance Monitoring

To ensure your error handling remains efficient, implement monitoring:

```go
type ErrorMetrics struct {
    creationTime   metrics.Histogram
    wrappingTime   metrics.Histogram
    stackDepth     metrics.Histogram
    allocationSize metrics.Histogram
}

func TrackErrorMetrics(err error, metrics *ErrorMetrics) {
    if wrappedErr, ok := err.(*ewrap.Error); ok {
        metrics.stackDepth.Observe(float64(len(wrappedErr.Stack())))
        // Track other metrics...
    }
}
```

## Best Practices for Performance

### 1. Pool Appropriately

Choose pool sizes based on your application's characteristics:

```go
func initializePools(config Config) {
    // Size pools based on expected concurrent operations
    errorGroupPool := ewrap.NewErrorGroupPool(config.MaxConcurrentOperations)

    // Size buffer pools based on expected error volume
    bufferPool := sync.Pool{
        New: func() any {
            return bytes.NewBuffer(make([]byte, 0, config.AverageErrorSize))
        },
    }
}
```

### 2. Minimize Allocations

Be mindful of unnecessary allocations:

```go
// Good - reuse error types for common cases
var (
    ErrNotFound = ewrap.New("resource not found",
        ewrap.WithErrorType(ErrorTypeNotFound))
    ErrUnauthorized = ewrap.New("unauthorized access",
        ewrap.WithErrorType(ErrorTypePermission))
)

// Avoid - creating new errors for common cases
if !exists {
    return ewrap.New("resource not found")  // Creates new error each time
}
```

### 3. Profile and Monitor

Regularly profile your error handling:

```go
func TestErrorPerformance(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping performance test in short mode")
    }

    // Profile creation
    b.Run("ErrorCreation", func(b *testing.B) {
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            _ = ewrap.New("test error")
        }
    })

    // Profile wrapping
    b.Run("ErrorWrapping", func(b *testing.B) {
        baseErr := errors.New("base error")
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            _ = ewrap.Wrap(baseErr, "wrapped error")
        }
    })
}
```
