# ewrap Documentation

Welcome to the documentation for `ewrap`, a sophisticated error handling package for Go applications that provides comprehensive error management capabilities with a focus on performance and developer experience.

## Overview

ewrap is designed to make error handling in Go applications more robust, informative, and maintainable. It provides a rich set of features while maintaining excellent performance characteristics through careful optimization and efficient memory management.

### Key Features

- **Stack Traces**: Automatically capture and filter stack traces for meaningful error debugging
- **Error Wrapping**: Maintain error chains while preserving context
- **Metadata Attachment**: Attach and manage arbitrary key-value pairs to errors
- **Logging Integration**: Flexible logger interface supporting major logging frameworks
- **Error Categorization**: Built-in error types and severity levels
- **Circuit Breaker Pattern**: Protect your systems from cascading failures
- **Efficient Error Grouping**: Pool-based error group management
- **Context Preservation**: Rich error context preservation
- **Thread-Safe Operations**: Safe for concurrent use
- **Format Options**: JSON and YAML output support

## Quick Example

Here's a quick example of how ewrap can be used in your application:

```go
func processOrder(ctx context.Context, orderID string) error {
    // Get an error group from the pool
    pool := ewrap.NewErrorGroupPool(4)
    eg := pool.Get()
    defer eg.Release()

    // Create a circuit breaker
    cb := ewrap.NewCircuitBreaker("database", 3, time.Minute)

    // Add validation errors to the group
    if err := validateOrder(orderID); err != nil {
        eg.Add(ewrap.Wrap(err, "invalid order",
            ewrap.WithContext(ctx),
            ewrap.WithErrorType(ewrap.ErrorTypeValidation),
            ewrap.WithLogger(logger)))
    }

    // Handle database operations with circuit breaker
    if !eg.HasErrors() && cb.CanExecute() {
        if err := saveToDatabase(orderID); err != nil {
            cb.RecordFailure()
            return ewrap.Wrap(err, "database operation failed",
                ewrap.WithContext(ctx),
                ewrap.WithErrorType(ewrap.ErrorTypeDatabase),
                ewrap.WithRetry(3, time.Second*5))
        }
        cb.RecordSuccess()
    }

    return eg.Error()
}
```

## Getting Started

To start using ewrap in your project, visit the [Installation](getting-started/installation.md) guide, followed by the [Quick Start](getting-started/quickstart.md) tutorial.

## Why ewrap?

ewrap was created to address common challenges in error handling:

1. **Lack of Context**: Traditional error handling often loses important context
2. **Performance Overhead**: Many error handling libraries introduce significant overhead
3. **Memory Management**: Poor memory management in error handling can lead to increased GC pressure
4. **Inconsistent Logging**: Different parts of an application often handle error logging differently
5. **Missing Stack Traces**: Getting meaningful stack traces can be challenging
6. **Circuit Breaking**: Protecting systems from cascading failures requires complex implementation

ewrap solves these challenges while maintaining excellent performance characteristics and providing a clean, intuitive API.
