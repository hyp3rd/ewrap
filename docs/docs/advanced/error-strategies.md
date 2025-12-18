# Error Handling Strategies

Understanding how to effectively handle errors is crucial for building robust applications. This guide explores various error handling strategies using ewrap and explains when to use each approach.

## Understanding Error Context

Error context is more than just an error message - it's the complete picture of what happened when an error occurred. With ewrap, you can capture rich context that helps with debugging and error resolution.

### Basic Context

At its simplest, context includes:

```go
err := ewrap.New("user not found",
    ewrap.WithContext(ctx, ewrap.ErrorTypeNotFound, ewrap.SeverityError))
```

This tells us:

- What happened ("user not found")
- The type of error (NotFound)
- How severe the error is (Error level)

### Enhanced Context

For more complex scenarios, you can add detailed context:

```go
err := ewrap.New("database query failed",
    ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical),
    ewrap.WithLogger(logger)).
    WithMetadata("query", query).
    WithMetadata("table", "users").
    WithMetadata("affected_rows", 0).
    WithMetadata("latency_ms", queryTime.Milliseconds())
```

This provides a complete picture:

- The operation that failed
- Where it failed
- Related technical details
- Performance metrics

## Error Group Strategies

Error groups are powerful tools for handling multiple potential errors in a single operation. Here's how to use them effectively:

### Validation Scenarios

When validating multiple fields or conditions:

```go
func validateUser(user User) error {
    pool := ewrap.NewErrorGroupPool(4)
    eg := pool.Get()
    defer eg.Release()

    // Validate email
    if !isValidEmail(user.Email) {
        eg.Add(ewrap.New("invalid email format",
            ewrap.WithErrorType(ewrap.ErrorTypeValidation)))
    }

    // Validate age
    if user.Age < 18 {
        eg.Add(ewrap.New("user must be 18 or older",
            ewrap.WithErrorType(ewrap.ErrorTypeValidation)))
    }

    // Validate username
    if len(user.Username) < 3 {
        eg.Add(ewrap.New("username too short",
            ewrap.WithErrorType(ewrap.ErrorTypeValidation)))
    }

    return eg.Error()
}
```

### Parallel Operations

When handling concurrent operations:

```go
func processItems(ctx context.Context, items []Item) error {
    pool := ewrap.NewErrorGroupPool(len(items))
    eg := pool.Get()
    defer eg.Release()

    var wg sync.WaitGroup
    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()
            if err := processItem(ctx, item); err != nil {
                eg.Add(ewrap.Wrap(err, fmt.Sprintf("failed to process item %d", item.ID)))
            }
        }(item)
    }

    wg.Wait()
    return eg.Error()
}
```

## Circuit Breaker Patterns

Circuit breakers help prevent system overload by failing fast when problems are detected. Here are some patterns for using them effectively:

### Basic Circuit Breaker

For simple protection:

```go
cb := ewrap.NewCircuitBreaker("database", 5, time.Minute)

func queryDatabase() error {
    if !cb.CanExecute() {
        return ewrap.New("circuit breaker open",
            ewrap.WithErrorType(ewrap.ErrorTypeDatabase),
            ewrap.WithMetadata("breaker", "database"))
    }

    if err := performQuery(); err != nil {
        cb.RecordFailure()
        return err
    }

    cb.RecordSuccess()
    return nil
}
```

### Cascading Circuit Breakers

For systems with dependencies:

```go
type Service struct {
    dbBreaker    *ewrap.CircuitBreaker
    cacheBreaker *ewrap.CircuitBreaker
    apiBreaker   *ewrap.CircuitBreaker
}

func (s *Service) getData(ctx context.Context, id string) (Data, error) {
    // Try cache first
    if s.cacheBreaker.CanExecute() {
        data, err := queryCache(id)
        if err == nil {
            s.cacheBreaker.RecordSuccess()
            return data, nil
        }
        s.cacheBreaker.RecordFailure()
    }

    // Fall back to database
    if s.dbBreaker.CanExecute() {
        data, err := queryDatabase(id)
        if err == nil {
            s.dbBreaker.RecordSuccess()
            return data, nil
        }
        s.dbBreaker.RecordFailure()
    }

    // Last resort: external API
    if s.apiBreaker.CanExecute() {
        data, err := queryAPI(id)
        if err == nil {
            s.apiBreaker.RecordSuccess()
            return data, nil
        }
        s.apiBreaker.RecordFailure()
    }

    return Data{}, ewrap.New("all data sources failed",
        ewrap.WithErrorType(ewrap.ErrorTypeInternal),
        ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityCritical))
}
```

## Best Practices for Error Recovery

When handling errors, consider implementing recovery strategies:

1. **Graceful Degradation**:

    ```go
    func getProductDetails(ctx context.Context, id string) (Product, error) {
        var product Product

        // Get core product data
        data, err := getProductData(id)
        if err != nil {
            // Log the error but continue with partial data
            logger.Error("failed to get full product data", "error", err)
            product.Status = "partial"
        } else {
            product.Status = "complete"
        }

        // Get non-critical enrichment data
        reviews, err := getProductReviews(id)
        if err != nil {
            // Add metadata about missing data
            err = ewrap.Wrap(err, "failed to get reviews",
                ewrap.WithMetadata("missing_component", "reviews"),
                ewrap.WithErrorType(ewrap.ErrorTypePartial))
            logger.Warn("serving product without reviews", "error", err)
        }

        return product, nil
    }
    ```

1. **Retry Patterns**:

```go
func withRetry(operation func() error, maxAttempts int, delay time.Duration) error {
    var lastErr error

    for attempt := 1; attempt <= maxAttempts; attempt++ {
        err := operation()
        if err == nil {
            return nil
        }

        lastErr = ewrap.Wrap(err, "operation failed",
            ewrap.WithMetadata("attempt", attempt),
            ewrap.WithMetadata("max_attempts", maxAttempts))

        if attempt < maxAttempts {
            time.Sleep(delay * time.Duration(attempt))
        }
    }

    return lastErr
}
```

The key to effective error handling is choosing the right strategy for each situation. Consider:

- The criticality of the operation
- Performance requirements
- User experience implications
- System resources
- Dependencies and their reliability

By understanding these factors and using ewrap's features appropriately, you can build robust and maintainable error handling systems.
