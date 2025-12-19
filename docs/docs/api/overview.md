# Package Overview

The ewrap package provides a sophisticated error handling solution for Go applications, combining modern error handling patterns with performance optimizations and developer-friendly features. This overview explains how the package's components work together to create a comprehensive error handling system.

## Core Philosophy

ewrap is built on several key principles:

1. **Rich Context**: Errors should carry enough information to understand what went wrong, where, and why
1. **Performance**: Error handling shouldn't become a bottleneck in your application
1. **Developer Experience**: Clear, consistent patterns that make error handling both powerful and approachable
1. **Flexibility**: Easy integration with existing systems and adaptable to different needs
1. **Production Readiness**: Built-in support for logging, monitoring, and debugging

## Component Architecture

### Error Types and Context

At the heart of ewrap is the Error type, which provides the foundation for all error handling:

```go
// Core error structure
type Error struct {
    msg      string
    cause    error
    stack    []uintptr
    metadata map[string]any
    logger   Logger
    mu       sync.RWMutex
}

// Error context structure
type ErrorContext struct {
    Timestamp   time.Time
    Type        ErrorType
    Severity    Severity
    Operation   string
    Component   string
    RequestID   string
    User        string
    Environment string
    Version     string
    File        string
    Line        int
    Data        map[string]any
}
```

This structure allows errors to carry:

- Stack traces for debugging
- Metadata for context
- Logging configuration
- Error categorization
- Severity levels
- Operation tracking

### Memory Management

The package includes sophisticated memory management through pooling:

```go
// Error group pooling
pool := ewrap.NewErrorGroupPool(4)
eg := pool.Get()
defer eg.Release()

// Usage in high-throughput scenarios
for item := range items {
    if err := processItem(item); err != nil {
        eg.Add(err)
    }
}
```

This system helps reduce garbage collection pressure in applications that handle many errors.

### Logging Integration

The logging system is designed for flexibility and integration with existing logging frameworks:

```go
// Logger interface
type Logger interface {
    Error(msg string, keysAndValues ...any)
    Debug(msg string, keysAndValues ...any)
    Info(msg string, keysAndValues ...any)
}
```

Built-in adapters support popular logging frameworks while allowing custom implementations.

### Circuit Breaker Pattern

The circuit breaker implementation provides protection against cascading failures:

```go
type CircuitBreaker struct {
    name           string
    maxFailures    int
    timeout        time.Duration
    failureCount   int
    lastFailure    time.Time
    state          CircuitState
    mu             sync.RWMutex
    onStateChange  func(name string, from, to CircuitState)
}
```

This helps build resilient systems that can gracefully handle service degradation.

## Feature Integration

### Error Creation and Wrapping

The package provides a fluent API for error creation and wrapping:

```go
// Error creation
err := ewrap.New("operation failed",
    ewrap.WithContext(ctx, ErrorTypeInternal, SeverityError),
    ewrap.WithLogger(logger))

// Error wrapping
if err != nil {
    return ewrap.Wrap(err, "processing failed",
        ewrap.WithContext(ctx, ErrorTypeInternal, SeverityError))
}
```

### Error Groups and Concurrency

Error groups handle concurrent error collection with built-in synchronization:

```go
func processItems(items []Item) error {
    pool := ewrap.NewErrorGroupPool(len(items))
    eg := pool.Get()
    defer eg.Release()

    var wg sync.WaitGroup
    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()
            if err := processItem(item); err != nil {
                eg.Add(err)
            }
        }(item)
    }
    wg.Wait()

    return eg.Error()
}
```

## Performance Considerations

The package includes several optimizations:

1. **Memory Pooling**: Reduces allocation overhead
1. **Lock-Free Operations**: Where possible, for better concurrency
1. **Efficient Stack Traces**: Captures only necessary frames
1. **Lazy Formatting**: Defers expensive string operations

## Best Practices

To get the most out of ewrap:

1. **Use Error Types Consistently**: Choose appropriate error types for better error handling:

   ```go
   ErrorTypeValidation  // For input validation
   ErrorTypeDatabase    // For database operations
   ErrorTypeNetwork     // For network operations
   ```

1. **Leverage Context**: Add relevant context to errors:

   ```go
   ewrap.WithContext(ctx, errorType, severity)
   ```

1. **Implement Proper Logging**: Use structured logging for better debugging:

   ```go
   ewrap.WithLogger(logger)
   ```

1. **Use Error Groups Efficiently**: Pool error groups for better performance:

   ```go
   pool := ewrap.NewErrorGroupPool(size)
   ```

1. **Handle Circuit Breaking**: Protect your system from cascading failures:

   ```go
   breaker := ewrap.NewCircuitBreaker(name, maxFailures, timeout)
   ```

## Common Patterns

### Request Handling

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Get error group from pool
    pool := ewrap.NewErrorGroupPool(4)
    eg := pool.Get()
    defer eg.Release()

    // Process request
    if err := processRequest(ctx, r); err != nil {
        eg.Add(ewrap.Wrap(err, "request processing failed",
            ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityError)))
    }

    // Handle any errors
    if err := eg.Error(); err != nil {
        handleError(w, err)
        return
    }

    // Send success response
    respondSuccess(w)
}
```

### Service Integration

```go
type Service struct {
    breaker *ewrap.CircuitBreaker
    logger  Logger
}

func (s *Service) CallExternalService(ctx context.Context) error {
    if !s.breaker.CanExecute() {
        return ewrap.New("service unavailable",
            ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityCritical),
            ewrap.WithLogger(s.logger))
    }

    if err := makeExternalCall(); err != nil {
        s.breaker.RecordFailure()
        return ewrap.Wrap(err, "external call failed",
            ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityError))
    }

    s.breaker.RecordSuccess()
    return nil
}
```

## Integration Points

ewrap is designed to integrate with:

- Logging frameworks (zap, logrus, zerolog)
- Monitoring systems
- Tracing solutions
- HTTP middleware
- Database layers
- External services

## Version Compatibility

The package is compatible with:

- Go 1.13+ error wrapping
- Standard library contexts
- Common logging frameworks
- Standard HTTP packages
- Database/SQL interfaces

## Further Reading

For more detailed information about specific features, refer to:

- [Error Types Documentation](error-types.md)
- [Circuit Breaker Documentation](../features/circuit-breaker.md)
- [Error Groups Documentation](../features/error-groups.md)
- [Logging Integration](../features/logging.md)
- [Context Integration](../advanced/context.md)
