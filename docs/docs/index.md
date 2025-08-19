# ewrap Documentation

Welcome to the documentation for `ewrap`, a sophisticated, modern error handling library for Go applications that provides comprehensive error management with advanced features, observability hooks, and seamless integration with Go 1.25+ features.

## Overview

ewrap is designed to make error handling in Go applications more robust, informative, and maintainable. It provides a rich set of features while maintaining excellent performance characteristics through careful optimization, efficient memory management, and modern Go language features.

### Key Features

- **Advanced Stack Traces**: Programmatic stack frame inspection with iterators and structured access
- **Smart Error Wrapping**: Maintains error chains with unified context handling and metadata preservation
- **Modern Logging Integration**: Support for slog (Go 1.21+), logrus, zap, zerolog with structured output
- **Observability Hooks**: Built-in metrics and tracing for error frequencies and circuit-breaker states
- **Go 1.25+ Optimizations**: Uses `maps.Clone` and `slices.Clone` for efficient copying operations
- **Pool-based Error Groups**: Memory-efficient error aggregation with `errors.Join` compatibility
- **Circuit Breaker Pattern**: Protect systems from cascading failures with state transition monitoring
- **Custom Retry Logic**: Configurable per-error retry strategies with `RetryInfo` extension
- **Recovery Guidance**: Integrated recovery suggestions in error output and logging
- **Structured Serialization**: JSON/YAML export with full error group serialization
- **Thread-Safe Operations**: Zero-allocation hot paths with minimal contention
- **Type-Safe Metadata**: Optional generics support for strongly typed error contexts

## Quick Example

Here's a comprehensive example showcasing the modern features of ewrap:

```go
func processOrder(ctx context.Context, orderID string) error {
    // Set up observability
    observer := &MyObserver{metricsClient: metrics, tracer: trace}

    // Get an error group from the pool with errors.Join support
    pool := ewrap.NewErrorGroupPool(4)
    eg := pool.Get()
    defer eg.Release()

    // Create a circuit breaker with observability hooks
    cb := ewrap.NewCircuitBreaker("payment-service", 5, time.Minute*2,
        ewrap.WithObserver(observer))

    // Add validation errors with recovery suggestions
    if err := validateOrder(orderID); err != nil {
        eg.Add(ewrap.Wrap(err, "invalid order",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError),
            ewrap.WithRecoverySuggestion("Validate order format and required fields"),
            ewrap.WithLogger(slogLogger)))
    }

    // Handle database operations with custom retry logic
    shouldRetry := func(err error, attempt int) bool {
        return attempt < 3 && ewrap.IsType(err, ewrap.ErrorTypeNetwork)
    }

    if !eg.HasErrors() && cb.CanExecute() {
        if err := saveToDatabase(orderID); err != nil {
            cb.RecordFailure()
            dbErr := ewrap.Wrap(err, "database operation failed",
                ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical),
                ewrap.WithRetryInfo(3, time.Second*5, shouldRetry),
                ewrap.WithRecoverySuggestion("Check database connectivity and connection pool"))

            // Inspect stack frames programmatically
            if iterator := dbErr.GetStackIterator(); iterator.HasNext() {
                frame := iterator.Next()
                // Custom handling based on stack frame information
                handleCriticalFrame(frame)
            }

            return dbErr
        }
        cb.RecordSuccess()
    }

    // Use errors.Join compatibility for standard library integration
    if err := eg.Join(); err != nil {
        // Serialize the entire error group for structured logging
        if jsonOutput, serErr := eg.ToJSON(ewrap.WithTimestampFormat(time.RFC3339)); serErr == nil {
            structuredLogger.Error("order processing failed", "errors", jsonOutput)
        }
        return err
    }

    return nil
}
```

## Getting Started

To start using ewrap in your project, visit the [Installation](getting-started/installation.md) guide, followed by the [Quick Start](getting-started/quickstart.md) tutorial.

## Why ewrap?

ewrap was created to address common challenges in modern Go error handling:

### Traditional Challenges Solved

1. **Context Loss**: Traditional error handling often loses important context during error propagation
2. **Performance Overhead**: Many error handling libraries introduce significant memory and CPU overhead
3. **Memory Management**: Poor memory management in error handling leads to increased GC pressure
4. **Inconsistent Logging**: Different parts of applications handle error logging differently
5. **Missing Stack Traces**: Getting meaningful, filterable stack traces is challenging
6. **Circuit Breaking**: Protecting systems from cascading failures requires complex implementation

### Modern Go Challenges Addressed

1. **Go 1.25+ Feature Integration**: Lack of libraries leveraging modern Go performance features
2. **Observability Gaps**: Missing built-in support for metrics and tracing in error handling
3. **Recovery Guidance**: Errors without actionable remediation suggestions
4. **Type Safety**: Metadata handling without compile-time guarantees
5. **Standard Library Integration**: Poor integration with `errors.Join` and modern error patterns
6. **Serialization Complexity**: Difficulty in structured error export for monitoring systems

ewrap provides solutions to all these challenges while maintaining backward compatibility and excellent performance characteristics.

ewrap solves these challenges while maintaining excellent performance characteristics and providing a clean, intuitive API.
