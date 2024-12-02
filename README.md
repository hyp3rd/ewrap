# ewrap

[![Go](https://github.com/hyp3rd/ewrap/actions/workflows/go.yml/badge.svg)](https://github.com/hyp3rd/ewrap/actions/workflows/go.yml)

A sophisticated, configurable error wrapper for Go applications that provides comprehensive error handling capabilities with a focus on performance and flexibility.

## Core Features

- **Stack Traces**: Automatically captures and filters stack traces for meaningful error debugging
- **Error Wrapping**: Maintains error chains while preserving context through the entire chain
- **Metadata Attachment**: Attach and manage arbitrary key-value pairs to errors
- **Logging Integration**: Flexible logger interface supporting major logging frameworks (logrus, zap, zerolog)
- **Error Categorization**: Built-in error types and severity levels for better error handling
- **Circuit Breaker Pattern**: Protect your systems from cascading failures
- **Efficient Error Grouping**: Pool-based error group management for high-performance scenarios
- **Context Preservation**: Rich error context including request IDs, user information, and operation details
- **Thread-Safe Operations**: Safe for concurrent use in all operations
- **Format Options**: JSON and YAML output support with customizable formatting
- **Go 1.13+ Compatible**: Full support for `errors.Is`, `errors.As`, and error chains

## Installation

```bash
go get github.com/hyp3rd/errors-wrapper
```

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
```

### Advanced Error Context

Add rich context and metadata to errors:

```go
err := ewrap.New("operation failed",
    ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical),
    ewrap.WithLogger(logger)).
    WithMetadata("query", "SELECT * FROM users").
    WithMetadata("retry_count", 3)

// Log the error with all context
err.Log()
```

### Error Groups with Pooling

Use error groups efficiently in high-throughput scenarios:

```go
// Create an error group pool with initial capacity
pool := ewrap.NewErrorGroupPool(4)

// Get an error group from the pool
eg := pool.Get()
defer eg.Release()  // Return to pool when done

// Add errors as needed
eg.Add(err1)
eg.Add(err2)

if eg.HasErrors() {
    return eg.Error()
}
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
    Error(msg string, keysAndValues ...interface{})
    Debug(msg string, keysAndValues ...interface{})
    Info(msg string, keysAndValues ...interface{})
}
```

Built-in adapters are provided for popular logging frameworks:

```go
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

## Error Formatting

Convert errors to structured formats:

```go
// Convert to JSON
jsonStr, _ := err.ToJSON(
    ewrap.WithTimestampFormat(time.RFC3339),
    ewrap.WithStackTrace(true))

// Convert to YAML
yamlStr, _ := err.ToYAML(
    ewrap.WithStackTrace(true))
```

## Performance Considerations

The package is designed with performance in mind:

- Error groups use sync.Pool for efficient memory usage
- Minimal allocations in hot paths
- Thread-safe operations with low contention
- Pre-allocated buffers for string operations
- Efficient stack trace capture and filtering

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
├── cmd/ # Main applications
│   └── app/ # Your application
│       └── main.go # Application entry point
├── internal/ # Private code
│   ├── pkg/ # Internal packages
│   └── app/ # Application specific code
├── pkg/ # Public libraries
├── api/ # API contracts (proto files, OpenAPI specs)
├── configs/ # Configuration files
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
