# Quick Start Guide

This guide will help you get started with ewrap quickly. We'll cover the basic concepts and show you how to use the main features of the package.

## Basic Usage

### Creating Errors

The simplest way to create an error with ewrap is using the `New` function:

```go
err := ewrap.New("something went wrong")
```

### Adding Context

You can add context to your errors using various options:

```go
err := ewrap.New("database connection failed",
    ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical),
    ewrap.WithLogger(logger))
```

### Wrapping Errors

When you want to add context to an existing error:

```go
if err != nil {
    return ewrap.Wrap(err, "failed to process request")
}
```

### Using Error Groups

Error groups help you collect and manage multiple errors:

```go
// Create an error group pool
pool := ewrap.NewErrorGroupPool(4)

// Get an error group from the pool
eg := pool.Get()
defer eg.Release()  // Don't forget to release it back to the pool

// Add errors as needed
eg.Add(err1)
eg.Add(err2)

if eg.HasErrors() {
    return eg.Error()
}
```

### Implementing Circuit Breaker

Protect your system from cascading failures:

```go
cb := ewrap.NewCircuitBreaker("database", 3, time.Minute)

if cb.CanExecute() {
    err := performOperation()
    if err != nil {
        cb.RecordFailure()
        return err
    }
    cb.RecordSuccess()
}
```

## Next Steps

Now that you understand the basics, you can:

1. Learn about [Error Types](../features/error-types.md)
2. Explore [Logging Integration](../features/logging.md)
3. Study [Advanced Usage](../advanced/performance.md)
4. Check out complete [Examples](../examples/basic.md)

## Best Practices

Here are some best practices to follow when using ewrap:

1. Always provide meaningful error messages
2. Use appropriate error types and severity levels
3. Release error groups back to their pools
4. Configure circuit breakers based on your system's characteristics
5. Implement proper logging integration
6. Use metadata to add relevant debugging information

## Common Patterns

Here are some common patterns you might find useful:

```go
func processItem(ctx context.Context, item string) error {
    // Create error group from pool
    pool := ewrap.NewErrorGroupPool(4)
    eg := pool.Get()
    defer eg.Release()

    // Validate input
    if err := validate(item); err != nil {
        eg.Add(ewrap.Wrap(err, "validation failed",
            ewrap.WithContext(ctx),
            ewrap.WithErrorType(ewrap.ErrorTypeValidation)))
    }

    // Process if no validation errors
    if !eg.HasErrors() {
        if err := process(item); err != nil {
            return ewrap.Wrap(err, "processing failed",
                ewrap.WithContext(ctx),
                ewrap.WithErrorType(ewrap.ErrorTypeInternal))
        }
    }

    return eg.Error()
}
```

This is just a starting point. For more detailed information about specific features, check out the relevant sections in the documentation.
