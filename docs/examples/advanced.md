# Advanced Examples

These examples demonstrate sophisticated error handling patterns using ewrap's advanced features. We'll explore complex scenarios that combine multiple features to create robust error handling solutions.

## Microservice Error Handling

This example shows a complete microservice error handling setup, combining circuit breakers, error groups, and contextual logging:

```go
// ServiceManager handles communication with external services
type ServiceManager struct {
    // Circuit breakers for different services
    authBreaker    *ewrap.CircuitBreaker
    paymentBreaker *ewrap.CircuitBreaker
    storageBreaker *ewrap.CircuitBreaker

    // Error group pool for batch operations
    errorPool *ewrap.ErrorGroupPool

    // Contextual logger
    logger Logger
}

func NewServiceManager(logger Logger) *ServiceManager {
    return &ServiceManager{
        authBreaker:    ewrap.NewCircuitBreaker("auth", 5, time.Second*30),
        paymentBreaker: ewrap.NewCircuitBreaker("payment", 3, time.Minute),
        storageBreaker: ewrap.NewCircuitBreaker("storage", 5, time.Second*45),
        errorPool:      ewrap.NewErrorGroupPool(10),
        logger:        logger,
    }
}

func (sm *ServiceManager) ProcessOrder(ctx context.Context, order Order) error {
    // Create error group for the operation
    eg := sm.errorPool.Get()
    defer eg.Release()

    // Enrich context with operation details
    ctx = context.WithValue(ctx, "operation", "process_order")
    ctx = context.WithValue(ctx, "order_id", order.ID)

    // Step 1: Authenticate user
    if !sm.authBreaker.CanExecute() {
        return ewrap.New("authentication service unavailable",
            ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityCritical),
            ewrap.WithLogger(sm.logger))
    }

    if err := sm.authenticateUser(ctx, order.UserID); err != nil {
        sm.authBreaker.RecordFailure()
        return ewrap.Wrap(err, "authentication failed",
            ewrap.WithContext(ctx, ewrap.ErrorTypePermission, ewrap.SeverityError),
            ewrap.WithLogger(sm.logger))
    }
    sm.authBreaker.RecordSuccess()

    // Step 2: Process payment with retry mechanism
    if !sm.paymentBreaker.CanExecute() {
        return ewrap.New("payment service unavailable",
            ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityCritical),
            ewrap.WithLogger(sm.logger))
    }

    err := retry.Do(
        func() error {
            return sm.processPayment(ctx, order)
        },
        retry.Attempts(3),
        retry.Delay(time.Second),
        retry.OnRetry(func(n uint, err error) {
            sm.logger.Debug("retrying payment",
                "attempt", n+1,
                "error", err.Error())
        }),
    )

    if err != nil {
        sm.paymentBreaker.RecordFailure()
        return ewrap.Wrap(err, "payment processing failed",
            ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityError),
            ewrap.WithLogger(sm.logger))
    }
    sm.paymentBreaker.RecordSuccess()

    // Step 3: Update inventory in parallel
    var wg sync.WaitGroup
    for _, item := range order.Items {
        wg.Add(1)
        go func(item OrderItem) {
            defer wg.Done()

            if err := sm.updateInventory(ctx, item); err != nil {
                eg.Add(ewrap.Wrap(err, "inventory update failed",
                    ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError),
                    ewrap.WithLogger(sm.logger)).
                    WithMetadata("item_id", item.ID))
            }
        }(item)
    }
    wg.Wait()

    // Check if any inventory updates failed
    if eg.HasErrors() {
        return eg.Error()
    }

    return nil
}

// Error handling middleware for HTTP servers
func ErrorHandlingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Create request-specific error group
        pool := ewrap.NewErrorGroupPool(4)
        eg := pool.Get()
        defer eg.Release()

        // Enrich context with request information
        ctx := r.Context()
        ctx = context.WithValue(ctx, "request_id", uuid.New().String())
        ctx = context.WithValue(ctx, "user_agent", r.UserAgent())
        ctx = context.WithValue(ctx, "remote_addr", r.RemoteAddr)

        // Wrap handler execution with panic recovery
        func() {
            defer func() {
                if r := recover(); r != nil {
                    err := ewrap.New("panic recovered",
                        ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityCritical),
                        ewrap.WithLogger(logger)).
                        WithMetadata("panic_value", r).
                        WithMetadata("stack", string(debug.Stack()))
                    eg.Add(err)
                }
            }()

            next.ServeHTTP(w, r.WithContext(ctx))
        }()

        // Handle any collected errors
        if eg.HasErrors() {
            handleErrors(w, r, eg.Error())
            return
        }
    })
}

// Sophisticated error response handling
func handleErrors(w http.ResponseWriter, r *http.Request, err error) {
    var response ErrorResponse

    if wrappedErr, ok := err.(*ewrap.Error); ok {
        // Convert error to API response
        response = ErrorResponse{
            Code:      getErrorCode(wrappedErr),
            Message:   sanitizeErrorMessage(wrappedErr.Error()),
            RequestID: r.Context().Value("request_id").(string),
            Timestamp: time.Now().UTC(),
        }

        // Add details based on error type
        if details := getErrorDetails(wrappedErr); details != nil {
            response.Details = details
        }

        // Log error with full context
        logger.Error("request failed",
            "error", wrappedErr.Error(),
            "stack", wrappedErr.Stack(),
            "metadata", wrappedErr.GetMetadata("error_context"),
            "request_id", response.RequestID)

        // Set appropriate HTTP status
        w.WriteHeader(getHTTPStatus(wrappedErr))
    } else {
        // Handle unwrapped errors
        response = ErrorResponse{
            Code:    "INTERNAL_ERROR",
            Message: "An unexpected error occurred",
        }
        w.WriteHeader(http.StatusInternalServerError)
    }

    // Write JSON response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// Helper functions for error handling
func getErrorCode(err *ewrap.Error) string {
    if ctx, ok := err.GetMetadata("error_context").(*ewrap.ErrorContext); ok {
        switch ctx.Type {
        case ewrap.ErrorTypeValidation:
            return "VALIDATION_ERROR"
        case ewrap.ErrorTypePermission:
            return "PERMISSION_DENIED"
        case ewrap.ErrorTypeNotFound:
            return "NOT_FOUND"
        case ewrap.ErrorTypeDatabase:
            return "DATABASE_ERROR"
        default:
            return "INTERNAL_ERROR"
        }
    }
    return "UNKNOWN_ERROR"
}

func getHTTPStatus(err *ewrap.Error) int {
    if ctx, ok := err.GetMetadata("error_context").(*ewrap.ErrorContext); ok {
        switch ctx.Type {
        case ewrap.ErrorTypeValidation:
            return http.StatusBadRequest
        case ewrap.ErrorTypePermission:
            return http.StatusForbidden
        case ewrap.ErrorTypeNotFound:
            return http.StatusNotFound
        default:
            return http.StatusInternalServerError
        }
    }
    return http.StatusInternalServerError
}
```

This example demonstrates several advanced concepts:

- Circuit breaker integration for external service calls
- Error group pooling for efficient error collection
- Context propagation through the request lifecycle
- Sophisticated error response handling
- Panic recovery and logging
- Error type mapping to HTTP status codes
- Request-scoped error tracking
