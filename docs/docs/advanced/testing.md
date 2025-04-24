# Testing Error Handling

Testing error handling is crucial for building reliable applications. Good error handling tests not only verify that errors are caught and handled correctly but also ensure that error contexts, metadata, and performance characteristics meet your requirements. Let's explore how to effectively test error handling using ewrap.

## Understanding Error Testing

Testing error handling requires a different mindset from testing normal application flow. We need to verify not just that errors are caught, but that they carry the right information, perform efficiently, and integrate properly with the rest of our system. Let's break this down into manageable pieces.

## Unit Testing Error Handling

Let's start with basic unit tests that verify error creation and handling:

```go
func TestErrorCreation(t *testing.T) {
    // We'll create a structured test to verify different aspects of error creation
    testCases := []struct {
        name           string
        message        string
        errorType     ErrorType
        severity      Severity
        metadata      map[string]any
        expectedStack bool
    }{
        {
            name:       "Basic Error",
            message:    "something went wrong",
            errorType:  ErrorTypeUnknown,
            severity:   SeverityError,
            metadata:   nil,
            expectedStack: true,
        },
        {
            name:       "Database Error with Metadata",
            message:    "connection failed",
            errorType:  ErrorTypeDatabase,
            severity:   SeverityCritical,
            metadata: map[string]any{
                "host": "localhost",
                "port": 5432,
            },
            expectedStack: true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Create the error with test case parameters
            err := ewrap.New(tc.message,
                ewrap.WithContext(context.Background(), tc.errorType, tc.severity))

            // Add metadata if provided
            if tc.metadata != nil {
                for k, v := range tc.metadata {
                    err = err.WithMetadata(k, v)
                }
            }

            // Verify error properties
            if err.Error() != tc.message {
                t.Errorf("Expected message %q, got %q", tc.message, err.Error())
            }

            // Verify stack trace presence
            if tc.expectedStack && err.Stack() == "" {
                t.Error("Expected stack trace, but none was captured")
            }

            // Verify metadata
            if tc.metadata != nil {
                for k, v := range tc.metadata {
                    if mv, ok := err.GetMetadata(k); !ok || mv != v {
                        t.Errorf("Metadata %q: expected %v, got %v", k, v, mv)
                    }
                }
            }
        })
    }
}
```

## Testing Error Wrapping

Error wrapping requires special attention to ensure context is preserved:

```go
func TestErrorWrapping(t *testing.T) {
    // Create a mock logger to verify logging behavior
    mockLogger := NewMockLogger(t)

    // Create a base error
    baseErr := errors.New("base error")

    // Create our wrapped error
    wrappedErr := ewrap.Wrap(baseErr, "operation failed",
        ewrap.WithLogger(mockLogger),
        ewrap.WithContext(context.Background(), ErrorTypeDatabase, SeverityCritical))

    // Test error chain
    t.Run("Error Chain", func(t *testing.T) {
        // Verify the complete error message
        expectedMsg := "operation failed: base error"
        if wrappedErr.Error() != expectedMsg {
            t.Errorf("Expected message %q, got %q", expectedMsg, wrappedErr.Error())
        }

        // Verify we can unwrap to the original error
        if !errors.Is(wrappedErr, baseErr) {
            t.Error("Wrapped error should match the original error")
        }

        // Verify the error chain is preserved
        cause := wrappedErr.Unwrap()
        if cause != baseErr {
            t.Error("Unwrapped error should be the base error")
        }
    })

    // Test context preservation
    t.Run("Context Preservation", func(t *testing.T) {
        ctx := getErrorContext(wrappedErr)

        if ctx.Type != ErrorTypeDatabase {
            t.Errorf("Expected error type %v, got %v", ErrorTypeDatabase, ctx.Type)
        }

        if ctx.Severity != SeverityCritical {
            t.Errorf("Expected severity %v, got %v", SeverityCritical, ctx.Severity)
        }
    })
}
```

## Testing Error Groups

Error groups require testing both individual operations and concurrent behavior:

```go
func TestErrorGroup(t *testing.T) {
    // Test pool creation and basic operations
    t.Run("Basic Operations", func(t *testing.T) {
        pool := ewrap.NewErrorGroupPool(4)
        eg := pool.Get()
        defer eg.Release()

        // Add some errors
        eg.Add(ewrap.New("error 1"))
        eg.Add(ewrap.New("error 2"))

        // Verify error count
        if !eg.HasErrors() {
            t.Error("Expected errors in group")
        }

        // Verify error message format
        errMsg := eg.Error()
        if !strings.Contains(errMsg, "error 1") || !strings.Contains(errMsg, "error 2") {
            t.Error("Error message doesn't contain all errors")
        }
    })

    // Test concurrent operations
    t.Run("Concurrent Operations", func(t *testing.T) {
        pool := ewrap.NewErrorGroupPool(4)
        eg := pool.Get()
        defer eg.Release()

        var wg sync.WaitGroup
        for i := 0; i < 100; i++ {
            wg.Add(1)
            go func(i int) {
                defer wg.Done()
                eg.Add(ewrap.New(fmt.Sprintf("concurrent error %d", i)))
            }(i)
        }

        wg.Wait()

        // Verify all errors were captured
        errs := eg.Errors()
        if len(errs) != 100 {
            t.Errorf("Expected 100 errors, got %d", len(errs))
        }
    })
}
```

## Testing Circuit Breakers

Circuit breakers require testing state transitions and timing behavior:

```go
func TestCircuitBreaker(t *testing.T) {
    t.Run("State Transitions", func(t *testing.T) {
        cb := ewrap.NewCircuitBreaker("test", 3, time.Second)

        // Should start closed
        if !cb.CanExecute() {
            t.Error("Circuit breaker should start in closed state")
        }

        // Record failures until open
        for i := 0; i < 3; i++ {
            cb.RecordFailure()
        }

        // Should now be open
        if cb.CanExecute() {
            t.Error("Circuit breaker should be open after failures")
        }

        // Wait for timeout
        time.Sleep(time.Second + 100*time.Millisecond)

        // Should be half-open
        if !cb.CanExecute() {
            t.Error("Circuit breaker should be half-open after timeout")
        }

        // Record success to close
        cb.RecordSuccess()

        // Should be closed
        if !cb.CanExecute() {
            t.Error("Circuit breaker should be closed after success")
        }
    })
}
```

## Performance Testing

Performance testing is crucial for error handling code:

```go
func BenchmarkErrorOperations(b *testing.B) {
    // Benchmark error creation
    b.Run("Creation", func(b *testing.B) {
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            _ = ewrap.New("test error")
        }
    })

    // Benchmark error wrapping
    b.Run("Wrapping", func(b *testing.B) {
        baseErr := errors.New("base error")
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            _ = ewrap.Wrap(baseErr, "wrapped error")
        }
    })

    // Benchmark error group operations
    b.Run("ErrorGroup", func(b *testing.B) {
        pool := ewrap.NewErrorGroupPool(4)
        b.ReportAllocs()
        b.RunParallel(func(pb *testing.PB) {
            for pb.Next() {
                eg := pool.Get()
                eg.Add(errors.New("test error"))
                _ = eg.Error()
                eg.Release()
            }
        })
    })
}
```

## Integration Testing

Testing error handling in integration scenarios:

```go
func TestErrorIntegration(t *testing.T) {
    // Create a test server
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Simulate error handling in an HTTP server
        err := processRequest(r)
        if err != nil {
            // Convert error to API response
            resp := formatErrorResponse(err)
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(resp)
            return
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer srv.Close()

    // Test error handling through the entire stack
    t.Run("Integration", func(t *testing.T) {
        resp, err := http.Get(srv.URL)
        if err != nil {
            t.Fatal(err)
        }
        defer resp.Body.Close()

        // Verify error response format
        if resp.StatusCode != http.StatusInternalServerError {
            t.Errorf("Expected status 500, got %d", resp.StatusCode)
        }

        var errorResp map[string]any
        if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
            t.Fatal(err)
        }

        // Verify error response structure
        requiredFields := []string{"message", "code", "timestamp"}
        for _, field := range requiredFields {
            if _, ok := errorResp[field]; !ok {
                t.Errorf("Missing required field: %s", field)
            }
        }
    })
}
```
