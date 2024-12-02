# ewrap

A sophisticated, configurable error wrapper for Go applications that provides:

- **Stack Traces**: Automatically captures stack traces when errors are created
- **Error Wrapping**: Maintains error chains while adding context
- **Metadata**: Attach arbitrary key-value pairs to errors
- **Logging Integration**: Flexible logger interface for custom logging implementations
- **Go 1.13+ Compatible**: Works with `errors.Is`, `errors.As`, and error chains
- **Clean Stack Traces**: Filters out runtime frames for cleaner output
- **Thread-Safe**: Safe for concurrent use
- **Context Preservation**: Maintains error context through the chain

Common logging frameworks like logrus, zap, or zerolog can be easily adapted to this interface.

## Installation

```bash
go get github.com/hyp3rd/errors-wrapper
```

## Usage

Basic error creation:

```go
err := ewrap.New("something went wrong")
```

With logger:

```go
logger := myapp.NewLogger() // implements ewrap.Logger
err := ewrap.New("something went wrong", ewrap.WithLogger(logger))
```

Wrapping an error with context:

```go
if err != nil {
    return ewrap.Wrap(err, "failed to process request")
}
```

Adding metadata:

```go
err := ewrap.New("database error", ewrap.WithLogger(logger)).
    WithMetadata("query", "SELECT * FROM users").
    WithMetadata("retry_count", 3)

// Log the error with all metadata
err.Log()
```

### Example

```go
func ProcessOrder(ctx context.Context, orderID string) error {
    if err := validateOrder(orderID); err != nil {
        return ewrap.Wrap(err, "invalid order",
            ewrap.WithContext(ctx),
            ewrap.WithErrorType(ewrap.ErrorTypeValidation),
            ewrap.WithLogger(logger))
    }

    err := saveToDatabase(orderID)
    if err != nil {
        return ewrap.Wrap(err, "failed to save order",
            ewrap.WithContext(ctx),
            ewrap.WithErrorType(ewrap.ErrorTypeDatabase),
            ewrap.WithRetry(3, time.Second*5),
            ewrap.WithLogger(logger))
    }

    return nil
}

// with circuit breaker and error grouping
func processOrder(ctx context.Context, orderID string) error {
    // Create a circuit breaker for database operations
    cb := ewrap.NewCircuitBreaker("database", 3, time.Minute)

    // Create an error group for collecting validation errors
    eg := ewrap.NewErrorGroup()

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

## Logger Interface

Implement the `Logger` interface to integrate with your logging system:

```go
type Logger interface {
    Error(msg string, keysAndValues ...interface{})
    Debug(msg string, keysAndValues ...interface{})
    Info(msg string, keysAndValues ...interface{})
}
```

## Development Setup

1. Clone this repository:

   ```bash
   git clone [<https://github.com/hyp3rd/ewrap.git> your-new-project](https://github.com/hyp3rd/ewrap.git)
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
