# Interfaces

The ewrap package is built around several key interfaces that provide flexibility and extensibility. Understanding these interfaces is crucial for effectively integrating ewrap into your application and extending its functionality to meet your specific needs.

## Core Interfaces

### Logger Interface

The Logger interface is fundamental to ewrap's logging capabilities. It provides a standardized way to log error information at different severity levels:

```go
type Logger interface {
    Error(msg string, keysAndValues ...interface{})
    Debug(msg string, keysAndValues ...interface{})
    Info(msg string, keysAndValues ...interface{})
}
```

This interface is intentionally simple yet powerful. Let's explore how to implement and use it effectively:

```go
// Example implementation using standard log package
type StandardLogger struct {
    logger *log.Logger
}

func (l *StandardLogger) Error(msg string, keysAndValues ...interface{}) {
    l.logger.Printf("ERROR: %s %v", msg, formatKeyValues(keysAndValues...))
}

func (l *StandardLogger) Debug(msg string, keysAndValues ...interface{}) {
    l.logger.Printf("DEBUG: %s %v", msg, formatKeyValues(keysAndValues...))
}

func (l *StandardLogger) Info(msg string, keysAndValues ...interface{}) {
    l.logger.Printf("INFO: %s %v", msg, formatKeyValues(keysAndValues...))
}

// Helper function to format key-value pairs
func formatKeyValues(keysAndValues ...interface{}) string {
    var pairs []string
    for i := 0; i < len(keysAndValues); i += 2 {
        if i+1 < len(keysAndValues) {
            pairs = append(pairs, fmt.Sprintf("%v=%v",
                keysAndValues[i], keysAndValues[i+1]))
        }
    }
    return strings.Join(pairs, " ")
}
```

The Logger interface is designed to:

- Support structured logging through key-value pairs
- Provide different log levels for appropriate error handling
- Be easily implemented by existing logging frameworks
- Allow for context-aware logging

### Error Interface

While ewrap's Error type implements Go's standard error interface, it extends it with additional capabilities:

```go
type error interface {
    Error() string
}

// ewrap.Error implements these additional methods:
type Error struct {
    // Cause returns the underlying cause of the error
    Cause() error

    // Stack returns the error's stack trace
    Stack() string

    // GetMetadata retrieves metadata associated with the error
    GetMetadata(key string) (interface{}, bool)

    // WithMetadata adds metadata to the error
    WithMetadata(key string, value interface{}) *Error

    // Is reports whether target matches err in the error chain
    Is(target error) bool

    // Unwrap provides compatibility with Go 1.13 error chains
    Unwrap() error
}
```

Understanding these interfaces helps when working with errors:

```go
func processError(err error) {
    // Type assert to access ewrap.Error functionality
    if wrappedErr, ok := err.(*ewrap.Error); ok {
        // Access stack trace
        fmt.Printf("Stack Trace:\n%s\n", wrappedErr.Stack())

        // Access metadata
        if requestID, ok := wrappedErr.GetMetadata("request_id"); ok {
            fmt.Printf("Request ID: %v\n", requestID)
        }

        // Access cause chain
        for cause := wrappedErr; cause != nil; cause = cause.Unwrap() {
            fmt.Printf("Error: %s\n", cause.Error())
        }
    }
}
```

## Interface Integration

### Implementing Custom Loggers

Creating custom loggers for different environments or logging systems:

```go
// Production logger with structured output
type ProductionLogger struct {
    output io.Writer
}

func (l *ProductionLogger) Error(msg string, keysAndValues ...interface{}) {
    entry := LogEntry{
        Level:     "ERROR",
        Message:   msg,
        Timestamp: time.Now().UTC(),
        Data:      makeMap(keysAndValues...),
    }
    json.NewEncoder(l.output).Encode(entry)
}

// Test logger for capturing logs in tests
type TestLogger struct {
    Logs []LogEntry
    mu   sync.Mutex
}

func (l *TestLogger) Error(msg string, keysAndValues ...interface{}) {
    l.mu.Lock()
    defer l.mu.Unlock()

    l.Logs = append(l.Logs, LogEntry{
        Level:     "ERROR",
        Message:   msg,
        Timestamp: time.Now(),
        Data:      makeMap(keysAndValues...),
    })
}
```

### Interface Composition

Combining interfaces for enhanced functionality:

```go
// MetricsLogger combines logging with metrics collection
type MetricsLogger struct {
    logger  Logger
    metrics MetricsCollector
}

func (l *MetricsLogger) Error(msg string, keysAndValues ...interface{}) {
    // Log the error
    l.logger.Error(msg, keysAndValues...)

    // Collect metrics
    l.metrics.IncrementCounter("errors_total", 1)
    l.metrics.Record("error_occurred", keysAndValues...)
}
```

## Best Practices

### Logger Implementation

When implementing the Logger interface, consider these guidelines:

```go
// Structured logger with configurable output
type StructuredLogger struct {
    output    io.Writer
    minLevel  LogLevel
    formatter LogFormatter
    mu        sync.Mutex
}

func (l *StructuredLogger) Error(msg string, keysAndValues ...interface{}) {
    if l.minLevel <= ErrorLevel {
        l.log(ErrorLevel, msg, keysAndValues...)
    }
}

func (l *StructuredLogger) log(level LogLevel, msg string, keysAndValues ...interface{}) {
    l.mu.Lock()
    defer l.mu.Unlock()

    entry := l.formatter.Format(level, msg, keysAndValues...)
    l.output.Write(entry)
}
```

### Interface Testing

Writing tests for interface implementations:

```go
func TestLogger(t *testing.T) {
    // Create a buffer to capture log output
    var buf bytes.Buffer
    logger := NewStructuredLogger(&buf)

    // Test error logging
    logger.Error("test error",
        "key1", "value1",
        "key2", 42)

    // Verify log output
    output := buf.String()
    if !strings.Contains(output, "test error") {
        t.Error("Log message not found in output")
    }

    // Verify structured data
    var logEntry LogEntry
    if err := json.NewDecoder(&buf).Decode(&logEntry); err != nil {
        t.Fatal(err)
    }

    if logEntry.Level != "ERROR" {
        t.Errorf("Expected level ERROR, got %s", logEntry.Level)
    }
}
```

### Interface Extensions

Creating specialized interfaces for specific use cases:

```go
// ContextualLogger adds context awareness to the basic Logger interface
type ContextualLogger interface {
    Logger
    WithContext(ctx context.Context) Logger
    WithFields(fields map[string]interface{}) Logger
}

// Implementation example
type contextualLogger struct {
    Logger
    ctx    context.Context
    fields map[string]interface{}
}

func (l *contextualLogger) WithContext(ctx context.Context) Logger {
    return &contextualLogger{
        Logger: l.Logger,
        ctx:    ctx,
        fields: l.fields,
    }
}

func (l *contextualLogger) Error(msg string, keysAndValues ...interface{}) {
    // Combine context values, fields, and provided key-values
    allKeyValues := l.mergeContextAndFields(keysAndValues...)
    l.Logger.Error(msg, allKeyValues...)
}
```
