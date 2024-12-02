# Context Integration

Understanding how to effectively integrate error handling with Go's context package is crucial for building robust, context-aware applications. Context integration allows us to carry request-scoped data, handle timeouts gracefully, and maintain traceability throughout our application's error handling flow.

## Understanding Context in Error Handling

Go's context package serves multiple purposes in error handling:

- Carrying request-scoped values (like request IDs or user information)
- Managing timeouts and cancellation
- Ensuring proper resource cleanup
- Maintaining traceability across service boundaries

Let's explore how ewrap integrates with context to enhance error handling capabilities.

## Basic Context Integration

The most straightforward way to integrate context with error handling is through the WithContext option:

```go
func processUserRequest(ctx context.Context, userID string) error {
    // Create an error with context
    if err := validateUser(userID); err != nil {
        return ewrap.Wrap(err, "user validation failed",
            ewrap.WithContext(ctx, ErrorTypeValidation, SeverityError))
    }

    return nil
}
```

When you add context to an error, ewrap automatically extracts and preserves important context values such as:

- Request IDs for tracing
- User information for auditing
- Operation metadata for monitoring
- Timing information for performance tracking

## Advanced Context Usage

Let's look at more sophisticated ways to use context in error handling:

```go
// Define context keys for common values
type contextKey string

const (
    requestIDKey contextKey = "request_id"
    userIDKey    contextKey = "user_id"
    traceIDKey   contextKey = "trace_id"
)

// RequestContext enriches context with standard fields
func RequestContext(ctx context.Context, requestID, userID string) context.Context {
    ctx = context.WithValue(ctx, requestIDKey, requestID)
    ctx = context.WithValue(ctx, userIDKey, userID)
    ctx = context.WithValue(ctx, traceIDKey, generateTraceID())
    return ctx
}

// ContextualOperation shows how to use context throughout an operation
func ContextualOperation(ctx context.Context) error {
    // Extract context values
    requestID := ctx.Value(requestIDKey).(string)
    userID := ctx.Value(userIDKey).(string)

    // Create an operation-specific context
    opCtx := ewrap.WithOperationContext(ctx, "user_update")

    // Start a timed operation
    timer := time.Now()

    // Perform the operation with timeout
    if err := performTimedOperation(opCtx); err != nil {
        return ewrap.Wrap(err, "operation failed",
            ewrap.WithContext(ctx, ErrorTypeInternal, SeverityError)).
            WithMetadata("request_id", requestID).
            WithMetadata("user_id", userID).
            WithMetadata("duration_ms", time.Since(timer).Milliseconds())
    }

    return nil
}
```

## Context-Aware Error Groups

Error groups can be made context-aware to handle cancellation and timeouts:

```go
// ContextualErrorGroup manages errors with context awareness
type ContextualErrorGroup struct {
    *ewrap.ErrorGroup
    ctx context.Context
}

// NewContextualErrorGroup creates a context-aware error group
func NewContextualErrorGroup(ctx context.Context, pool *ewrap.ErrorGroupPool) *ContextualErrorGroup {
    return &ContextualErrorGroup{
        ErrorGroup: pool.Get(),
        ctx:       ctx,
    }
}

// ProcessWithContext demonstrates context-aware parallel processing
func ProcessWithContext(ctx context.Context, items []Item) error {
    pool := ewrap.NewErrorGroupPool(len(items))
    group := NewContextualErrorGroup(ctx, pool)
    defer group.Release()

    var wg sync.WaitGroup
    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()

            // Check context cancellation
            select {
            case <-ctx.Done():
                group.Add(ewrap.New("operation cancelled",
                    ewrap.WithContext(ctx, ErrorTypeInternal, SeverityWarning)))
                return
            default:
                if err := processItem(ctx, item); err != nil {
                    group.Add(err)
                }
            }
        }(item)
    }

    wg.Wait()
    return group.Error()
}
```

## Timeout and Cancellation Handling

Proper context integration includes handling timeouts and cancellation gracefully:

```go
// TimeoutAwareOperation shows how to handle context timeouts
func TimeoutAwareOperation(ctx context.Context) error {
    // Create a timeout context
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    // Channel for operation result
    resultCh := make(chan error, 1)

    // Start the operation
    go func() {
        resultCh <- performLongOperation(ctx)
    }()

    // Wait for result or timeout
    select {
    case err := <-resultCh:
        if err != nil {
            return ewrap.Wrap(err, "operation failed",
                ewrap.WithContext(ctx, ErrorTypeInternal, SeverityError))
        }
        return nil
    case <-ctx.Done():
        return ewrap.New("operation timed out",
            ewrap.WithContext(ctx, ErrorTypeTimeout, SeverityCritical)).
            WithMetadata("timeout", 5*time.Second)
    }
}
```

## Context Propagation in Middleware

Context integration is particularly useful in middleware chains:

```go
// ErrorHandlingMiddleware demonstrates context propagation
func ErrorHandlingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Create request context with tracing
        ctx := r.Context()
        requestID := generateRequestID()
        ctx = context.WithValue(ctx, requestIDKey, requestID)

        // Create error group for request
        pool := ewrap.NewErrorGroupPool(4)
        eg := pool.Get()
        defer eg.Release()

        // Wrap handler execution
        err := func() error {
            // Add request timing
            timer := time.Now()
            defer func() {
                if err := recover(); err != nil {
                    eg.Add(ewrap.New("panic in handler",
                        ewrap.WithContext(ctx, ErrorTypeInternal, SeverityCritical)).
                        WithMetadata("panic_value", err).
                        WithMetadata("stack", debug.Stack()))
                }
            }()

            // Execute handler
            next.ServeHTTP(w, r.WithContext(ctx))

            // Record timing
            duration := time.Since(timer)
            if duration > time.Second {
                eg.Add(ewrap.New("slow request",
                    ewrap.WithContext(ctx, ErrorTypePerformance, SeverityWarning)).
                    WithMetadata("duration_ms", duration.Milliseconds()))
            }

            return nil
        }()

        if err != nil {
            eg.Add(err)
        }

        // Handle any collected errors
        if eg.HasErrors() {
            handleRequestErrors(w, eg.Error())
        }
    })
}
```

## Best Practices

### 1. Consistent Context Propagation

Maintain consistent context handling throughout your application:

```go
// ContextualService demonstrates consistent context handling
type ContextualService struct {
    db  *Database
    log Logger
}

func (s *ContextualService) ProcessRequest(ctx context.Context, req Request) error {
    // Enrich context with request information
    ctx = enrichContext(ctx, req)

    // Use context in all operations
    if err := s.validateRequest(ctx, req); err != nil {
        return ewrap.Wrap(err, "validation failed",
            ewrap.WithContext(ctx, ErrorTypeValidation, SeverityError))
    }

    if err := s.processData(ctx, req.Data); err != nil {
        return ewrap.Wrap(err, "processing failed",
            ewrap.WithContext(ctx, ErrorTypeInternal, SeverityError))
    }

    return nil
}
```

### 2. Context Value Management

Be careful with context values and provide type-safe accessors:

```go
// RequestInfo holds request-specific context values
type RequestInfo struct {
    RequestID string
    UserID    string
    TraceID   string
    StartTime time.Time
}

// GetRequestInfo safely extracts request information from context
func GetRequestInfo(ctx context.Context) (RequestInfo, bool) {
    info, ok := ctx.Value(requestInfoKey).(RequestInfo)
    return info, ok
}

// WithRequestInfo adds request information to context
func WithRequestInfo(ctx context.Context, info RequestInfo) context.Context {
    return context.WithValue(ctx, requestInfoKey, info)
}
```

### 3. Error Context Enrichment

Systematically enrich errors with context information:

```go
// EnrichError adds standard context information to errors
func EnrichError(ctx context.Context, err error) error {
    if err == nil {
        return nil
    }

    info, ok := GetRequestInfo(ctx)
    if !ok {
        return err
    }

    return ewrap.Wrap(err, "operation failed",
        ewrap.WithContext(ctx, getErrorType(err), getSeverity(err))).
        WithMetadata("request_id", info.RequestID).
        WithMetadata("user_id", info.UserID).
        WithMetadata("trace_id", info.TraceID).
        WithMetadata("duration_ms", time.Since(info.StartTime).Milliseconds())
}
```

The context integration in ewrap provides a robust foundation for error tracking and debugging. By consistently using these features, you can build applications that are easier to monitor, debug, and maintain.
