# Options

Options in ewrap provide a flexible way to configure error behavior. Using the functional options pattern, you can customize how errors are created, logged, and handled while maintaining clean and extensible code.

## Understanding Options

Options are functions that modify error behavior. They follow Go's functional options pattern, which allows for flexible and readable configuration. Each option function takes an error pointer and modifies its properties:

```go
type Option func(*Error)
```

## Built-in Options

### WithContext

The `WithContext` option enriches errors with contextual information, including error type, severity, and relevant request data:

```go
func processUser(ctx context.Context, userID string) error {
    err := ewrap.New("user processing failed",
        ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityError))

    // The error now includes:
    // - Error type and severity
    // - Stack trace location
    // - Request ID (if present in context)
    // - User information (if present in context)
    // - Operation name (if present in context)
    // - Component name (if present in context)
    // - Environment information
    return err
}
```

The context option automatically extracts common values from the provided context:

- `request_id` for request tracing
- `user` for user identification
- `operation` for operation naming
- `component` for system component identification

### WithLogger

The `WithLogger` option attaches a logger to the error, enabling automatic logging of error events:

```go
// Create a logger (implementing the Logger interface)
logger := NewZapLogger()

// Attach logger to error
err := ewrap.New("database connection failed",
    ewrap.WithLogger(logger),
    ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical))

// The error will automatically log:
// - Error creation
// - Context addition
// - Metadata changes
// - Stack trace information
```

### WithRetry

The `WithRetry` option configures retry behavior for recoverable errors:

```go
err := ewrap.New("temporary network failure",
    ewrap.WithContext(ctx, ewrap.ErrorTypeNetwork, ewrap.SeverityError),
    ewrap.WithRetry(3, time.Second*5))

// The error now includes retry information:
// - Maximum retry attempts (3)
// - Delay between attempts (5 seconds)
// - Retry strategy configuration
```

## Combining Options

Options can be combined to create rich error configurations:

```go
func processOrder(ctx context.Context, orderID string) error {
    // Create an error with multiple options
    return ewrap.New("order processing failed",
        // Add context information
        ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityError),
        // Attach logger
        ewrap.WithLogger(logger),
        // Configure retry behavior
        ewrap.WithRetry(3, time.Second*5))
}
```

## Creating Custom Options

You can create custom options to extend error functionality:

```go
// WithCorrelationID adds a correlation ID to the error
func WithCorrelationID(correlationID string) ewrap.Option {
    return func(err *ewrap.Error) {
        err.WithMetadata("correlation_id", correlationID)
    }
}

// WithResource adds resource information to the error
func WithResource(resourceType, resourceID string) ewrap.Option {
    return func(err *ewrap.Error) {
        err.WithMetadata("resource_type", resourceType)
        err.WithMetadata("resource_id", resourceID)
    }
}

// Usage example
err := ewrap.New("resource access failed",
    ewrap.WithContext(ctx, ewrap.ErrorTypePermission, ewrap.SeverityError),
    WithCorrelationID("corr-123"),
    WithResource("document", "doc-456"))
```

## Best Practices

### Option Organization

Group related options together for better readability:

```go
// Configuration options
configOpts := []ewrap.Option{
    ewrap.WithContext(ctx, errorType, severity),
    ewrap.WithLogger(logger),
}

// Retry options
retryOpts := []ewrap.Option{
    ewrap.WithRetry(maxAttempts, delay),
}

// Combine all options
allOpts := append(configOpts, retryOpts...)

// Create error with combined options
err := ewrap.New("operation failed", allOpts...)
```

### Option Factories

Create factory functions for commonly used option combinations:

```go
// CreateHTTPErrorOptions creates standard options for HTTP handlers
func CreateHTTPErrorOptions(ctx context.Context, logger Logger) []ewrap.Option {
    return []ewrap.Option{
        ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityError),
        ewrap.WithLogger(logger),
        WithCorrelationID(GetRequestID(ctx)),
    }
}

// Usage in HTTP handlers
func handleRequest(w http.ResponseWriter, r *http.Request) {
    opts := CreateHTTPErrorOptions(r.Context(), logger)

    if err := processRequest(r); err != nil {
        err = ewrap.Wrap(err, "request processing failed", opts...)
        handleError(w, err)
        return
    }
}
```

### Option Validation

When creating custom options, include validation logic:

```go
// WithTimeout adds a timeout duration to the error
func WithTimeout(duration time.Duration) ewrap.Option {
    return func(err *ewrap.Error) {
        // Validate input
        if duration <= 0 {
            duration = time.Second * 30 // Default timeout
        }

        err.WithMetadata("timeout", duration.String())
        err.WithMetadata("deadline", time.Now().Add(duration))
    }
}
```

### Dynamic Options

Create options that adapt based on conditions:

```go
// WithEnvironmentAwareLogging adjusts logging based on environment
func WithEnvironmentAwareLogging(logger Logger) ewrap.Option {
    return func(err *ewrap.Error) {
        env := os.Getenv("APP_ENV")

        switch env {
        case "production":
            // Use production logger settings
            err.WithMetadata("log_level", "error")
            err.WithMetadata("include_stack", true)
        case "development":
            // Use development logger settings
            err.WithMetadata("log_level", "debug")
            err.WithMetadata("include_stack", true)
        default:
            // Use default settings
            err.WithMetadata("log_level", "info")
        }

        err.SetLogger(logger)
    }
}
```
