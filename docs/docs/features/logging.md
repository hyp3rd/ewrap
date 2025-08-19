# Logging Integration

When errors occur in your application, having detailed, structured logs is crucial for understanding what went wrong and why. The ewrap package provides a flexible logging system that integrates seamlessly with popular logging frameworks while maintaining a clean, consistent interface for error logging.

## Understanding Logging in ewrap

The logging system in ewrap is built around a simple yet powerful interface that can adapt to any logging framework. When an error occurs, ewrap can automatically log not just the error message, but also the stack trace, metadata, and contextual information that helps tell the complete story of what happened.

## The Logger Interface

Let's start by understanding the core logging interface:

```go
type Logger interface {
    Error(msg string, keysAndValues ...any)
    Debug(msg string, keysAndValues ...any)
    Info(msg string, keysAndValues ...any)
}
```

This interface is intentionally simple to make it easy to adapt any logging framework to work with ewrap. The variadic `keysAndValues` parameter allows for structured logging where key-value pairs provide additional context.

## Built-in Logging Adapters

ewrap provides adapters for popular logging frameworks. Here's how to use them:

### Zap Logger Integration

```go
import (
    "go.uber.org/zap"
    "github.com/hyp3rd/ewrap/adapters"
)

func setupZapLogger() error {
    // Create a production-ready Zap logger
    zapLogger, err := zap.NewProduction()
    if err != nil {
        return err
    }

    // Create the adapter
    logger := adapters.NewZapAdapter(zapLogger)

    // Create an error with logging enabled
    err = ewrap.New("operation failed",
        ewrap.WithLogger(logger),
        ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityError)).
        WithMetadata("operation", "user_update").
        WithMetadata("user_id", userID)

    // The error will be automatically logged with all context
    return err
}
```

### Logrus Integration

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/hyp3rd/ewrap/adapters"
)

func setupLogrusLogger() error {
    // Configure Logrus
    logrusLogger := logrus.New()
    logrusLogger.SetFormatter(&logrus.JSONFormatter{})

    // Create the adapter
    logger := adapters.NewLogrusAdapter(logrusLogger)

    // Use the logger with ewrap
    return ewrap.New("database connection failed",
        ewrap.WithLogger(logger),
        ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical))
}
```

### Zerolog Integration

```go
import (
    "github.com/rs/zerolog"
    "github.com/hyp3rd/ewrap/adapters"
)

func setupZerolog() error {
    // Configure Zerolog
    zerologLogger := zerolog.New(os.Stdout).With().Timestamp().Logger()

    // Create the adapter
    logger := adapters.NewZerologAdapter(zerologLogger)

    // Use with ewrap
    return ewrap.New("request validation failed",
        ewrap.WithLogger(logger),
        ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityWarning))
}
```

### Slog Integration (Go 1.21+)

```go
import (
    "log/slog"
    "os"
    "github.com/hyp3rd/ewrap/adapters"
)

func setupSlogLogger() error {
    slogLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    logger := adapters.NewSlogAdapter(slogLogger)

    return ewrap.New("operation failed",
        ewrap.WithLogger(logger))
}
```

## Advanced Logging Patterns

### Contextual Logging

Here's how to create rich, contextual logs that capture the full story of an error:

```go
func processOrder(ctx context.Context, order Order) error {
    logger := getContextLogger(ctx)

    // Create an operation logger that will track the entire process
    opLogger := &OperationLogger{
        Logger:    logger,
        Operation: "process_order",
        StartTime: time.Now(),
        Context:   map[string]any{
            "order_id": order.ID,
            "user_id":  order.UserID,
        },
    }

    // Log operation start
    opLogger.Info("starting order processing")

    if err := validateOrder(order); err != nil {
        return ewrap.Wrap(err, "order validation failed",
            ewrap.WithLogger(opLogger),
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError)).
            WithMetadata("validation_time", time.Since(opLogger.StartTime))
    }

    if err := processPayment(order); err != nil {
        return ewrap.Wrap(err, "payment processing failed",
            ewrap.WithLogger(opLogger),
            ewrap.WithContext(ctx, ewrap.ErrorTypePayment, ewrap.SeverityCritical)).
            WithMetadata("processing_time", time.Since(opLogger.StartTime))
    }

    // Log successful completion
    opLogger.Info("order processed successfully",
        "duration", time.Since(opLogger.StartTime))

    return nil
}
```

### Creating a Custom Logger

You might want to create a custom logger that adds specific functionality:

```go
type CustomLogger struct {
    logger    Logger
    component string
    env       string
}

func NewCustomLogger(baseLogger Logger, component string) *CustomLogger {
    return &CustomLogger{
        logger:    baseLogger,
        component: component,
        env:       os.Getenv("APP_ENV"),
    }
}

func (l *CustomLogger) Error(msg string, keysAndValues ...any) {
    // Add standard context to all error logs
    enrichedKV := append([]any{
        "component", l.component,
        "environment", l.env,
        "timestamp", time.Now().UTC(),
    }, keysAndValues...)

    l.logger.Error(msg, enrichedKV...)
}

func (l *CustomLogger) Debug(msg string, keysAndValues ...any) {
    enrichedKV := append([]any{
        "component", l.component,
        "environment", l.env,
    }, keysAndValues...)

    l.logger.Debug(msg, enrichedKV...)
}

func (l *CustomLogger) Info(msg string, keysAndValues ...any) {
    enrichedKV := append([]any{
        "component", l.component,
        "environment", l.env,
    }, keysAndValues...)

    l.logger.Info(msg, enrichedKV...)
}
```

### Logging with Circuit Breakers

Combining logging with circuit breakers provides insight into system health:

```go
type MonitoredCircuitBreaker struct {
    *ewrap.CircuitBreaker
    logger Logger
    name   string
}

func NewMonitoredCircuitBreaker(name string, maxFailures int, timeout time.Duration, logger Logger) *MonitoredCircuitBreaker {
    cb := ewrap.NewCircuitBreaker(name, maxFailures, timeout)
    return &MonitoredCircuitBreaker{
        CircuitBreaker: cb,
        logger:        logger,
        name:         name,
    }
}

func (m *MonitoredCircuitBreaker) RecordFailure() {
    m.CircuitBreaker.RecordFailure()
    m.logger.Error("circuit breaker failure recorded",
        "breaker_name", m.name,
        "current_state", "open",
        "timestamp", time.Now())
}

func (m *MonitoredCircuitBreaker) RecordSuccess() {
    m.CircuitBreaker.RecordSuccess()
    m.logger.Info("circuit breaker success recorded",
        "breaker_name", m.name,
        "current_state", "closed",
        "timestamp", time.Now())
}
```

## Best Practices

### 1. Structured Logging

Always use structured logging for better searchability:

```go
// Good - structured logging
logger.Error("database query failed",
    "query", queryString,
    "duration_ms", duration.Milliseconds(),
    "affected_rows", 0)

// Avoid - unstructured logging
logger.Error(fmt.Sprintf("database query failed: %s (took %v)",
    queryString, duration))
```

### 2. Consistent Log Levels

Use appropriate log levels consistently:

```go
// Error - for actual errors
logger.Error("failed to process payment",
    "error", err,
    "user_id", userID)

// Debug - for detailed troubleshooting information
logger.Debug("attempting payment processing",
    "payment_provider", provider,
    "amount", amount)

// Info - for tracking normal operations
logger.Info("payment processed successfully",
    "transaction_id", txID,
    "amount", amount)
```

### 3. Context Preservation

Ensure context is preserved through the logging chain:

```go
func processWithContext(ctx context.Context) error {
    logger := getContextLogger(ctx)

    // Add request-specific context
    requestLogger := enrichLoggerWithContext(logger, ctx)

    err := performOperation()
    if err != nil {
        return ewrap.Wrap(err, "operation failed",
            ewrap.WithLogger(requestLogger),
            ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityError))
    }

    return nil
}

func enrichLoggerWithContext(logger Logger, ctx context.Context) Logger {
    // Extract common context values
    requestID := ctx.Value("request_id")
    userID := ctx.Value("user_id")

    return &ContextLogger{
        base:      logger,
        requestID: requestID.(string),
        userID:    userID.(string),
    }
}
```
