# Error Creation

Understanding how to create errors effectively is fundamental to using the ewrap package. This guide explains the different ways to create errors and when to use each approach.

## Basic Error Creation

The most straightforward way to create an error is using the `New` function. However, there's more to consider than just the error message.

```go
// Simple error creation
err := ewrap.New("user not found")

// With additional context
err := ewrap.New("user not found",
    ewrap.WithContext(ctx, ewrap.ErrorTypeNotFound, ewrap.SeverityError),
    ewrap.WithLogger(logger))
```

The `New` function captures a stack trace automatically, allowing you to trace the error's origin later. This is particularly valuable when debugging complex applications where errors might surface far from their source.

## Creating Errors with Options

The `New` function accepts variadic options that configure the error's behavior and context:

```go
err := ewrap.New("failed to process payment",
    // Add request context
    ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityCritical),
    // Configure logging
    ewrap.WithLogger(logger),
    // Add retry information
    ewrap.WithRetry(3, time.Second*5))
```

Each option serves a specific purpose:

- `WithContext`: Adds request context and error classification
- `WithLogger`: Configures error logging behavior
- `WithRetry`: Specifies retry behavior for recoverable errors

## Creating Domain-Specific Errors

For domain-specific error cases, you can combine error creation with metadata:

```go
func validateUserAge(age int) error {
    if age < 18 {
        return ewrap.New("user is underage",
            ewrap.WithContext(context.Background(), ewrap.ErrorTypeValidation, ewrap.SeverityError)).
            WithMetadata("minimum_age", 18).
            WithMetadata("provided_age", age)
    }
    return nil
}
```

## Best Practices for Error Creation

When creating errors, follow these guidelines for maximum effectiveness:

1. **Be Specific**: Error messages should clearly indicate what went wrong:

    ```go
    // Good
    err := ewrap.New("database connection timeout after 5 seconds")

    // Not as helpful
    err := ewrap.New("database error")
    ```

2. **Include Relevant Context**: Add context that helps with debugging:

    ```go
    err := ewrap.New("failed to update user profile",
        ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError)).
        WithMetadata("user_id", userID).
        WithMetadata("fields_updated", fields)
    ```

3. **Use Appropriate Error Types**: Choose error types that match the situation:

    ```go
    // For validation errors
    err := ewrap.New("invalid email format",
        ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityWarning))

    // For system errors
    err := ewrap.New("failed to connect to database",
        ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical))
    ```

4. **Consider Recovery Options**: Include information that helps with recovery:

    ```go
    err := ewrap.New("rate limit exceeded",
        ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityWarning)).
        WithMetadata("retry_after", time.Now().Add(time.Minute)).
        WithMetadata("current_rate", currentRate).
        WithMetadata("limit", rateLimit)
    ```

## Working with Stack Traces

Every error created with `New` automatically captures a stack trace:

```go
func processOrder(orderID string) error {
    err := ewrap.New("order processing failed")
    fmt.Println(err.Stack()) // Prints the stack trace
    return err
}
```

The stack trace includes function names, file names, and line numbers, making it easier to trace the error's origin.

## Error Creation in Tests

When writing tests, you might want to create errors for specific scenarios:

```go
func TestOrderProcessing(t *testing.T) {
    // Create a test error
    testErr := ewrap.New("simulated database error",
        ewrap.WithContext(context.Background(), ewrap.ErrorTypeDatabase, ewrap.SeverityError))

    // Mock database returns our test error
    mockDB := &MockDatabase{
        QueryFunc: func() error {
            return testErr
        },
    }

    err := processOrder(mockDB, "order123")

    // Verify error handling
    if !errors.Is(err, testErr) {
        t.Errorf("expected error %v, got %v", testErr, err)
    }
}
```

## Thread Safety

All error creation operations in ewrap are thread-safe. You can safely create errors from multiple goroutines:

```go
func processItems(items []string) []error {
    var wg sync.WaitGroup
    errors := make([]error, 0)
    var mu sync.Mutex

    for _, item := range items {
        wg.Add(1)
        go func(item string) {
            defer wg.Done()
            if err := process(item); err != nil {
                mu.Lock()
                errors = append(errors, ewrap.New("processing failed",
                    ewrap.WithMetadata("item", item)))
                mu.Unlock()
            }
        }(item)
    }

    wg.Wait()
    return errors
}
```

## Performance Considerations

Error creation in ewrap is optimized for both CPU and memory usage. However, consider these performance tips:

1. Reuse error types for common errors instead of creating new ones
2. Only capture stack traces when necessary
3. Be mindful of metadata quantity in high-throughput scenarios
4. Use error pools for frequent error creation scenarios

Remember: error creation should be reserved for exceptional cases. Don't use errors for normal control flow in your application.
