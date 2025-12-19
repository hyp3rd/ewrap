# Stack Traces

Stack traces are crucial for understanding where and why errors occur in your application. In ewrap, stack traces are automatically captured and enhanced to provide meaningful debugging information while maintaining performance. The latest version includes programmatic stack frame inspection through iterators and structured access.

## Understanding Stack Traces

A stack trace represents the sequence of function calls that led to an error. Think of it as a trail of breadcrumbs showing exactly how your program reached a particular point of failure. ewrap captures this information automatically while filtering out unnecessary noise.

## How ewrap Captures Stack Traces

When you create a new error using ewrap, it automatically captures the current stack trace:

```go
func processUserData(userID string) error {
    // This will capture the stack trace automatically
    if err := validateUser(userID); err != nil {
        return ewrap.New("user validation failed")
    }
    return nil
}
```

The captured stack trace includes:

- Function names
- File names
- Line numbers
- Program counter (PC) values

However, ewrap goes beyond simple capture by:

1. Filtering out runtime implementation details
1. Maintaining stack traces through error wrapping
1. Providing formatted output options
1. Offering programmatic access through iterators

## Programmatic Stack Frame Access

### Using Stack Iterators

The new stack iterator provides efficient, lazy access to stack frames:

```go
func analyzeError(err error) {
    if wrappedErr, ok := err.(*ewrap.Error); ok {
        iterator := wrappedErr.GetStackIterator()

        for iterator.HasNext() {
            frame := iterator.Next()

            fmt.Printf("Function: %s\n", frame.Function)
            fmt.Printf("File: %s:%d\n", frame.File, frame.Line)
            fmt.Printf("PC: %x\n", frame.PC)

            // Custom logic based on frame information
            if strings.Contains(frame.Function, "database") {
                handleDatabaseFrame(frame)
            }
        }
    }
}
```

### Accessing All Frames

Get all stack frames at once for batch processing:

```go
func generateErrorReport(err error) ErrorReport {
    if wrappedErr, ok := err.(*ewrap.Error); ok {
        frames := wrappedErr.GetStackFrames()

        return ErrorReport{
            Message: wrappedErr.Error(),
            StackFrames: frames,
            Timestamp: time.Now(),
        }
    }
    return ErrorReport{}
}
```

### Stack Frame Structure

Each stack frame provides detailed information:

```go
type StackFrame struct {
    Function string  `json:"function" yaml:"function"` // Fully qualified function name
    File     string  `json:"file" yaml:"file"`         // Source file path
    Line     int     `json:"line" yaml:"line"`         // Line number
    PC       uintptr `json:"pc" yaml:"pc"`             // Program counter
}
```

## Iterator Operations

### Navigation and Control

```go
iterator := wrappedErr.GetStackIterator()

// Check if more frames are available
if iterator.HasNext() {
    frame := iterator.Next()
    // Process frame
}

// Reset iterator to beginning
iterator.Reset()

// Get remaining frames from current position
remainingFrames := iterator.Frames()

// Get all frames regardless of current position
allFrames := iterator.AllFrames()
```

### Filtering and Processing

```go
func findApplicationFrames(err error) []StackFrame {
    var appFrames []StackFrame

    if wrappedErr, ok := err.(*ewrap.Error); ok {
        iterator := wrappedErr.GetStackIterator()

        for iterator.HasNext() {
            frame := iterator.Next()

            // Filter for application-specific frames
            if strings.Contains(frame.File, "/myapp/") &&
               !strings.Contains(frame.File, "/vendor/") {
                appFrames = append(appFrames, *frame)
            }
        }
    }

    return appFrames
}
```

When working with JSON output:

```go
err := ewrap.New("database connection failed")
jsonOutput, _ := err.ToJSON(ewrap.WithStackTrace(true))
fmt.Println(jsonOutput)
```

## Stack Trace Filtering

ewrap automatically filters stack traces to remove unhelpful information. Consider this example:

```go
func getUserProfile(id string) (*Profile, error) {
    profile, err := db.GetProfile(id)
    if err != nil {
        // The stack trace will exclude runtime internals
        return nil, ewrap.Wrap(err, "failed to retrieve user profile")
    }
    return profile, nil
}
```

The resulting stack trace might look like this:

```text
/app/services/user.go:25 - getUserProfile
/app/handlers/profile.go:42 - HandleProfileRequest
/app/router/routes.go:156 - ServeHTTP
```

Instead of the more verbose and less helpful unfiltered version:

```text
/app/services/user.go:25 - getUserProfile
/app/handlers/profile.go:42 - HandleProfileRequest
/app/router/routes.go:156 - ServeHTTP
/usr/local/go/src/runtime/asm_amd64.s:1571 - goexit
/usr/local/go/src/runtime/proc.go:203 - main
...
```

## Stack Traces in Error Chains

When you wrap errors, ewrap preserves the original stack trace while maintaining the error chain:

```go
func processOrder(orderID string) error {
    // Original error with its stack trace
    err := validateOrder(orderID)
    if err != nil {
        // Wraps the error while preserving the original stack trace
        return ewrap.Wrap(err, "order validation failed")
    }

    err = saveOrder(orderID)
    if err != nil {
        // Each wrap maintains the complete error context
        return ewrap.Wrap(err, "failed to save order")
    }

    return nil
}
```

## Performance Considerations

While stack traces are valuable for debugging, they do come with some overhead. ewrap optimizes this by:

1. Using efficient stack capture mechanisms
1. Implementing lazy formatting
1. Caching stack trace strings
1. Filtering irrelevant frames early

Here's how to work with stack traces efficiently:

```go
func processItems(items []Item) error {
    for _, item := range items {
        if err := processItem(item); err != nil {
            // In tight loops, consider whether you need the stack trace
            if isCriticalError(err) {
                return ewrap.Wrap(err, "critical error during item processing")
            }
            // For non-critical errors, maybe just log and continue
            log.Printf("Non-critical error: %v", err)
            continue
        }
    }
    return nil
}
```

## Using Stack Traces for Debugging

Stack traces are most valuable when combined with other error context. Here's a comprehensive example:

```go
func debugError(err error) {
    if wrappedErr, ok := err.(*ewrap.Error); ok {
        fmt.Printf("Error Message: %v\n", wrappedErr.Error())

        // Print the stack trace
        fmt.Printf("\nStack Trace:\n%s\n", wrappedErr.Stack())

        // Get any metadata
        if metadata, ok := wrappedErr.GetMetadata("request_id"); ok {
            fmt.Printf("\nRequest ID: %v\n", metadata)
        }

        // Print error chain
        fmt.Println("\nError Chain:")
        for e := wrappedErr; e != nil; e = e.Unwrap().(*ewrap.Error) {
            fmt.Printf("- %s\n", e.Error())
        }
    }
}
```

## Best Practices for Stack Traces

1. **Keep Stack Traces Meaningful**

    In service handlers, capture enough context without excessive detail:

    ```go
    func (s *Service) HandleRequest(ctx context.Context, req Request) error {
        // Capture high-level service context
        if err := s.processRequest(ctx, req); err != nil {
            return ewrap.Wrap(err, "request processing failed",
                ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityError))
        }
        return nil
    }
    ```

1. **Combine with Logging**

    Integrate stack traces with your logging system:

    ```go
    func logError(err error, logger Logger) {
        if wrappedErr, ok := err.(*ewrap.Error); ok {
            logger.Error("operation failed",
                "error", wrappedErr.Error(),
                "stack", wrappedErr.Stack(),
                "type", ewrap.GetErrorType(wrappedErr))
        }
    }
    ```

1. **Use in Development and Testing**

    Stack traces are particularly valuable during development and testing:

    ```go
    func TestComplexOperation(t *testing.T) {
        err := performComplexOperation()
        if err != nil {
            t.Errorf("Operation failed with stack trace:\n%+v", err)
        }
    }
    ```

## Common Pitfalls and Solutions

1. **Stack Trace Depth**

    If you're seeing too much or too little information:

    ```go
    // Too much information
    err := ewrap.New("operation failed")

    // Just right - wrap with context
    err := ewrap.New("operation failed",
        ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityError)).
        WithMetadata("operation", "user_update")
    ```

1. **Missing Context**

Ensure you're capturing relevant context with your stack traces:

```go
func handleRequest(ctx context.Context, req *Request) error {
    if err := validateRequest(req); err != nil {
        // Include request context with the stack trace
        return ewrap.Wrap(err, "invalid request",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError)).
            WithMetadata("request_id", req.ID).
            WithMetadata("user_id", req.UserID)
    }
    return nil
}
```
