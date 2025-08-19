# ewrap

[![Go](https://github.com/hyp3rd/ewrap/actions/workflows/go.yml/badge.svg)](https://github.com/hyp3rd/ewrap/actions/workflows/go.yml) [![Docs](https://img.shields.io/badge/docs-passing-brightgreen)](https://hyp3rd.github.io/ewrap/) [![Go Report Card](https://goreportcard.com/badge/github.com/hyp3rd/ewrap)](https://goreportcard.com/report/github.com/hyp3rd/ewrap) [![Go Reference](https://pkg.go.dev/badge/github.com/hyp3rd/ewrap.svg)](https://pkg.go.dev/github.com/hyp3rd/ewrap) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) ![GitHub Sponsors](https://img.shields.io/github/sponsors/hyp3rd)

A sophisticated, modern error handling library for Go applications that provides comprehensive error management with advanced features, observability hooks, and seamless integration with Go 1.25+ features.

## Core Features

### Error Management & Context

- **Advanced Stack Traces**: Programmatic stack frame inspection with iterators and structured access
- **Smart Error Wrapping**: Maintains error chains with unified context handling and metadata preservation
- **Rich Metadata**: Type-safe metadata attachment with optional generics support
- **Context Integration**: Unified context handling preventing divergence between error context and metadata

### Logging & Observability

- **Modern Logging**: Support for slog (Go 1.21+), logrus, zap, zerolog with structured output
- **Observability Hooks**: Built-in metrics and tracing for error frequencies and circuit-breaker states
- **Recovery Guidance**: Integrated recovery suggestions in error output and logging

### Performance & Efficiency

- **Go 1.25+ Optimizations**: Uses `maps.Clone` and `slices.Clone` for efficient copying operations
- **Pool-based Error Groups**: Memory-efficient error aggregation with `errors.Join` compatibility
- **Thread-Safe Operations**: Zero-allocation hot paths with minimal contention
- **Structured Serialization**: JSON/YAML export with full error group serialization

### Advanced Features

- **Circuit Breaker Pattern**: Protect systems from cascading failures with state transition monitoring
- **Custom Retry Logic**: Configurable per-error retry strategies with `RetryInfo` extension
- **Error Categorization**: Built-in types, severity levels, and optional generic type constraints
- **Timestamp Formatting**: Proper timestamp formatting with customizable formats

## Installation

```bash
go get github.com/hyp3rd/ewrap
```

## Documentation

`ewrap` provides comprehensive documentation covering all features and advanced usage patterns. Visit the [complete documentation](https://hyp3rd.github.io/ewrap/) for detailed guides, examples, and API reference.

## Usage Examples

### Basic Error Handling

Create and wrap errors with context:

```go
// Create a new error
err := ewrap.New("database connection failed")

// Wrap an existing error with context
if err != nil {
    return ewrap.Wrap(err, "failed to process request")
}

err = ewrap.Newf("failed to process request id: %v", requestID)
```

### Advanced Error Context with Unified Handling

Add rich context and metadata with the new unified context system:

```go
err := ewrap.New("operation failed",
    ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical),
    ewrap.WithLogger(logger),
    ewrap.WithRecoverySuggestion("Check database connection and retry")).
    WithMetadata("query", "SELECT * FROM users").
    WithMetadata("retry_count", 3).
    WithMetadata("connection_pool_size", 10)

// Log the error with all context and recovery suggestions
err.Log()
```

### Modern Error Groups with errors.Join Integration

Use error groups efficiently with Go 1.25+ features:

```go
// Create an error group pool with initial capacity
pool := ewrap.NewErrorGroupPool(4)

// Get an error group from the pool
eg := pool.Get()
defer eg.Release()  // Return to pool when done

// Add errors as needed
eg.Add(err1)
eg.Add(err2)

// Use errors.Join compatibility for standard library integration
if err := eg.Join(); err != nil {
    return err
}

// Or serialize the entire error group
jsonOutput, _ := eg.ToJSON(ewrap.WithTimestampFormat(time.RFC3339))
```

### Stack Frame Inspection and Iteration

Programmatically inspect stack traces:

```go
if wrappedErr, ok := err.(*ewrap.Error); ok {
    // Get a stack iterator for programmatic access
    iterator := wrappedErr.GetStackIterator()

    for iterator.HasNext() {
        frame := iterator.Next()
        fmt.Printf("Function: %s\n", frame.Function)
        fmt.Printf("File: %s:%d\n", frame.File, frame.Line)
        fmt.Printf("PC: %x\n", frame.PC)
    }

    // Or get all frames at once
    frames := wrappedErr.GetStackFrames()
    for _, frame := range frames {
        // Process each frame...
    }
}
```

### Custom Retry Logic with Extended RetryInfo

Configure per-error retry strategies:

```go
// Define custom retry logic
shouldRetry := func(err error, attempt int) bool {
    if attempt >= 5 {
        return false
    }

    // Custom logic based on error type
    if wrappedErr, ok := err.(*ewrap.Error); ok {
        return wrappedErr.ErrorType() == ewrap.ErrorTypeNetwork
    }
    return false
}

// Create error with custom retry configuration
err := ewrap.New("network timeout",
    ewrap.WithRetryInfo(3, time.Second*2, shouldRetry))

// Use the retry information
if retryInfo := err.GetRetryInfo(); retryInfo != nil {
    if retryInfo.ShouldRetry(err, currentAttempt) {
        // Perform retry logic
    }
}
```

### Observability Hooks and Monitoring

Monitor error patterns and circuit breaker states:

```go
// Set up observability hooks
observer := &MyObserver{
    metricsClient: metricsClient,
    tracer:       tracer,
}

// Create circuit breaker with observability
cb := ewrap.NewCircuitBreaker("payment-service", 5, time.Minute*2,
    ewrap.WithObserver(observer))

// The observer will receive notifications for:
// - Error frequency changes
// - Circuit breaker state transitions
// - Recovery suggestions triggered
```

### Circuit Breaker Pattern

Protect your system from cascading failures:

```go
// Create a circuit breaker for database operations
cb := ewrap.NewCircuitBreaker("database", 3, time.Minute)

if cb.CanExecute() {
    if err := performDatabaseOperation(); err != nil {
        cb.RecordFailure()
        return ewrap.Wrap(err, "database operation failed",
            ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical))
    }
    cb.RecordSuccess()
}
```

### Complete Example

Here's a comprehensive example combining multiple features:

```go
func processOrder(ctx context.Context, orderID string) error {
    // Get an error group from the pool
    pool := ewrap.NewErrorGroupPool(4)
    eg := pool.Get()
    defer eg.Release()

    // Create a circuit breaker for database operations
    cb := ewrap.NewCircuitBreaker("database", 3, time.Minute)

    // Validate order
    if err := validateOrderID(orderID); err != nil {
        eg.Add(ewrap.Wrap(err, "invalid order ID",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError)))
    }

    if !eg.HasErrors() && cb.CanExecute() {
        if err := saveToDatabase(orderID); err != nil {
            cb.RecordFailure()
            return ewrap.Wrap(err, "database operation failed",
                ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical))
        }
        cb.RecordSuccess()
    }

    return eg.Error()
}
```

## Error Types and Severity

The package provides pre-defined error types and severity levels:

```go
// Error Types
ErrorTypeValidation    // Input validation failures
ErrorTypeNotFound      // Resource not found
ErrorTypePermission    // Authorization/authentication failures
ErrorTypeDatabase      // Database operation failures
ErrorTypeNetwork       // Network-related failures
ErrorTypeConfiguration // Configuration issues
ErrorTypeInternal      // Internal system errors
ErrorTypeExternal      // External service errors

// Severity Levels
SeverityInfo      // Informational messages
SeverityWarning   // Warning conditions
SeverityError     // Error conditions
SeverityCritical  // Critical failures
```

## Logging Integration

Implement the Logger interface to integrate with your logging system:

```go
type Logger interface {
    Error(msg string, keysAndValues ...any)
    Debug(msg string, keysAndValues ...any)
    Info(msg string, keysAndValues ...any)
}
```

Built-in adapters are provided for popular logging frameworks including modern slog support:

```go
// Slog logger (Go 1.21+) - Recommended for new projects
slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
err := ewrap.New("error occurred",
    ewrap.WithLogger(adapters.NewSlogAdapter(slogLogger)))

// Zap logger
zapLogger, _ := zap.NewProduction()
err := ewrap.New("error occurred",
    ewrap.WithLogger(adapters.NewZapAdapter(zapLogger)))

// Logrus logger
logrusLogger := logrus.New()
err := ewrap.New("error occurred",
    ewrap.WithLogger(adapters.NewLogrusAdapter(logrusLogger)))

// Zerolog logger
zerologLogger := zerolog.New(os.Stdout)
err := ewrap.New("error occurred",
    ewrap.WithLogger(adapters.NewZerologAdapter(zerologLogger)))
```

### Recovery Suggestions in Logging

Recovery suggestions are now automatically included in log output:

```go
err := ewrap.New("database connection failed",
    ewrap.WithRecoverySuggestion("Check database connectivity and connection pool settings"))

// When logged, includes recovery guidance for operations teams
err.Log() // Outputs recovery suggestion in structured format
```

## Error Formatting

Convert errors to structured formats with proper timestamp formatting:

```go
// Convert to JSON with proper timestamp formatting
jsonStr, _ := err.ToJSON(
    ewrap.WithTimestampFormat(time.RFC3339),
    ewrap.WithStackTrace(true),
    ewrap.WithRecoverySuggestion(true))

// Convert to YAML with custom formatting
yamlStr, _ := err.ToYAML(
    ewrap.WithTimestampFormat("2006-01-02T15:04:05Z07:00"),
    ewrap.WithStackTrace(true))

// Serialize entire error groups
pool := ewrap.NewErrorGroupPool(4)
eg := pool.Get()
eg.Add(err1)
eg.Add(err2)

// Export all errors in the group
groupJSON, _ := eg.ToJSON(ewrap.WithTimestampFormat(time.RFC3339))
```

### Modern Go Features Integration

Leverage Go 1.25+ features for efficient operations:

```go
// Efficient metadata copying using maps.Clone
originalErr := ewrap.New("base error").WithMetadata("key1", "value1")
clonedErr := originalErr.Clone() // Uses maps.Clone internally

// Error group integration with errors.Join
eg := pool.Get()
eg.Add(err1, err2, err3)
standardErr := eg.Join() // Returns standard errors.Join result

// Use with standard library error handling
if errors.Is(standardErr, expectedErr) {
    // Handle specific error type
}
```

## Performance Considerations

The package is designed with performance in mind and leverages modern Go features:

### Go 1.25+ Optimizations

- Uses `maps.Clone` and `slices.Clone` for efficient copying operations
- Zero-allocation paths for error creation and wrapping in hot paths
- Optimized stack trace capture with intelligent filtering

### Memory Management

- Error groups use `sync.Pool` for efficient memory reuse
- Stack frame iterators provide lazy evaluation
- Minimal allocations during error metadata operations

### Concurrency & Safety

- Thread-safe operations with low lock contention
- Atomic operations for circuit breaker state management
- Lock-free observability hook notifications

### Structured Operations

- Pre-allocated buffers for JSON/YAML serialization
- Efficient stack trace capture and filtering
- Optimized metadata storage and retrieval

## Observability Features

### Built-in Monitoring

- Error frequency tracking and reporting
- Circuit breaker state transition monitoring
- Recovery suggestion effectiveness metrics

### Integration Points

```go
// Implement the Observer interface for custom monitoring
type Observer interface {
    OnErrorCreated(err *Error, context ErrorContext)
    OnCircuitBreakerStateChange(name string, from, to CircuitState)
    OnRecoverySuggestionTriggered(suggestion string, context ErrorContext)
}

// Register observers for monitoring
ewrap.RegisterGlobalObserver(myObserver)
```

## Development Setup

1. Clone this repository:

   ```bash
   git clone https://github.com/hyp3rd/ewrap.git
   ```

2. Install VS Code Extensions Recommended (optional):

   ```json
   {
     "recommendations": [
       "github.vscode-github-actions",
       "golang.go",
       "ms-vscode.makefile-tools",
       "esbenp.prettier-vscode",
       "pbkit.vscode-pbkit",
       "trunk.io",
       "streetsidesoftware.code-spell-checker",
       "ms-azuretools.vscode-docker",
       "eamodio.gitlens"
     ]
   }
   ```

   1. Install [**Golang**](https://go.dev/dl).
   2. Install [**GitVersion**](https://github.com/GitTools/GitVersion).
   3. Install [**Make**](https://www.gnu.org/software/make/), follow the procedure for your OS.
   4. **Set up the toolchain:**

      ```bash
      make prepare-toolchain
      ```

   5. Initialize `pre-commit` (strongly recommended to create a virtual env, using for instance [PyEnv](https://github.com/pyenv/pyenv)) and its hooks:

   ```bash
      pip install pre-commit
      pre-commit install
      pre-commit install-hooks
   ```

## Project Structure

```txt
├── internal/ # Private code
│   └── logger/ # Application specific code
├── pkg/ # Public libraries)
├── scripts/ # Scripts for development
├── test/ # Additional test files
└── docs/ # Documentation
```

## Best Practices

- Follow the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- Run `golangci-lint` before committing code
- Ensure the pre-commit hooks pass
- Write tests for new functionality
- Keep packages small and focused
- Use meaningful package names
- Document exported functions and types

## Available Make Commands

- `make test`: Run tests.
- `make benchmark`: Run benchmark tests.
- `make update-deps`: Update all dependencies in the project.
- `make prepare-toolchain`: Install all tools required to build the project.
- `make lint`: Run the staticcheck and golangci-lint static analysis tools on all packages in the project.
- `make run`: Build and run the application in Docker.

## License

[MIT License](LICENSE)

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

Refer to [CONTRIBUTING](CONTRIBUTING.md) for more information.

## Author

I'm a surfer, and a software architect with 15 years of experience designing highly available distributed production systems and developing cloud-native apps in public and private clouds. Feel free to connect with me on LinkedIn.

[![LinkedIn](https://img.shields.io/badge/LinkedIn-0077B5?style=for-the-badge&logo=linkedin&logoColor=white)](https://www.linkedin.com/in/francesco-cosentino/)
