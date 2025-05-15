# Installation

```bash
go get github.com/hyp3rd/ewrap
```

## Usage Examples

### Basic Error Handling

Create and wrap errors with context:

```go
import "github.com/hyp3rd/ewrap"
// ...
// Create a new error
err := ewrap.New("database connection failed")

// Wrap an existing error with context
if err != nil {
    return ewrap.Wrap(err, "failed to process request")
}

// Create a new error with context
err := ewrap.Newf("operation failed: %w", err)
    ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical),
    ewrap.WithLogger(logger))

err.Log()
```

### Advanced Error Context

Add rich context and metadata to errors:

```go
import "github.com/hyp3rd/ewrap"
// ...
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
