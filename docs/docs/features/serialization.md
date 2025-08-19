# Serialization and Error Groups

ewrap provides comprehensive serialization capabilities for both individual errors and error groups, enabling structured export for monitoring systems, APIs, and debugging purposes. The serialization system supports multiple formats and is optimized for performance.

## Error Group Serialization

### Basic Serialization

Export entire error groups to structured formats:

```go
// Create an error group with multiple errors
pool := ewrap.NewErrorGroupPool(4)
eg := pool.Get()
defer eg.Release()

// Add various errors
eg.Add(ewrap.New("database connection failed",
    ewrap.WithErrorType(ewrap.ErrorTypeDatabase),
    ewrap.WithSeverity(ewrap.SeverityCritical)))

eg.Add(ewrap.New("validation error",
    ewrap.WithErrorType(ewrap.ErrorTypeValidation),
    ewrap.WithSeverity(ewrap.SeverityError)))

// Serialize to JSON
jsonOutput, err := eg.ToJSON(
    ewrap.WithTimestampFormat(time.RFC3339),
    ewrap.WithStackTrace(true),
    ewrap.WithRecoverySuggestion(true))

if err != nil {
    log.Printf("Serialization failed: %v", err)
    return
}

fmt.Println(jsonOutput)
```

### JSON Output Structure

The JSON serialization produces a structured format:

```json
{
  "error_group": {
    "timestamp": "2024-03-15T14:30:00Z",
    "total_errors": 2,
    "errors": [
      {
        "message": "database connection failed",
        "type": "database",
        "severity": "critical",
        "timestamp": "2024-03-15T14:30:00Z",
        "metadata": {},
        "stack_trace": [
          {
            "function": "main.connectDatabase",
            "file": "/app/main.go",
            "line": 42,
            "pc": "0x4567890"
          }
        ],
        "recovery_suggestion": "Check database connectivity and connection pool settings"
      },
      {
        "message": "validation error",
        "type": "validation",
        "severity": "error",
        "timestamp": "2024-03-15T14:30:00Z",
        "metadata": {},
        "stack_trace": []
      }
    ]
  }
}
```

### YAML Serialization

Export error groups to YAML format:

```go
yamlOutput, err := eg.ToYAML(
    ewrap.WithTimestampFormat("2006-01-02T15:04:05Z07:00"),
    ewrap.WithStackTrace(false)) // Exclude stack traces for cleaner output

if err != nil {
    log.Printf("YAML serialization failed: %v", err)
    return
}

fmt.Println(yamlOutput)
```

### YAML Output Structure

```yaml
error_group:
  timestamp: "2024-03-15T14:30:00Z"
  total_errors: 2
  errors:
    - message: "database connection failed"
      type: "database"
      severity: "critical"
      timestamp: "2024-03-15T14:30:00Z"
      metadata: {}
      recovery_suggestion: "Check database connectivity and connection pool settings"
    - message: "validation error"
      type: "validation"
      severity: "error"
      timestamp: "2024-03-15T14:30:00Z"
      metadata: {}
```

## Individual Error Serialization

### Enhanced JSON Serialization

Individual errors can be serialized with full context:

```go
err := ewrap.New("payment processing failed",
    ewrap.WithErrorType(ewrap.ErrorTypeExternal),
    ewrap.WithSeverity(ewrap.SeverityCritical),
    ewrap.WithRecoverySuggestion("Retry with exponential backoff or contact payment provider")).
    WithMetadata("payment_id", "pay_12345").
    WithMetadata("amount", 99.99).
    WithMetadata("currency", "USD")

jsonOutput, serErr := err.ToJSON(
    ewrap.WithTimestampFormat(time.RFC3339),
    ewrap.WithStackTrace(true),
    ewrap.WithRecoverySuggestion(true))

if serErr != nil {
    log.Printf("Error serialization failed: %v", serErr)
    return
}
```

### Custom Serialization Options

#### Timestamp Formatting

Configure timestamp formats for different use cases:

```go
// RFC3339 format (recommended for APIs)
jsonOutput, _ := err.ToJSON(ewrap.WithTimestampFormat(time.RFC3339))

// Custom format for logs
jsonOutput, _ := err.ToJSON(ewrap.WithTimestampFormat("2006-01-02 15:04:05"))

// Unix timestamp for systems integration
jsonOutput, _ := err.ToJSON(ewrap.WithTimestampFormat("unix"))
```

#### Stack Trace Control

Control stack trace inclusion in serialized output:

```go
// Include full stack traces (for debugging)
jsonOutput, _ := err.ToJSON(ewrap.WithStackTrace(true))

// Exclude stack traces (for production logs)
jsonOutput, _ := err.ToJSON(ewrap.WithStackTrace(false))

// Include only application frames
jsonOutput, _ := err.ToJSON(
    ewrap.WithStackTrace(true),
    ewrap.WithStackFilter(func(frame StackFrame) bool {
        return strings.Contains(frame.File, "/myapp/") &&
               !strings.Contains(frame.File, "/vendor/")
    }))
```

#### Recovery Suggestions

Control recovery suggestion inclusion:

```go
// Include recovery suggestions (for operational use)
jsonOutput, _ := err.ToJSON(ewrap.WithRecoverySuggestion(true))

// Exclude recovery suggestions (for end-user APIs)
jsonOutput, _ := err.ToJSON(ewrap.WithRecoverySuggestion(false))
```

## Integration with errors.Join

### Standard Library Compatibility

ewrap error groups integrate seamlessly with Go's standard `errors.Join`:

```go
// Create error group
eg := pool.Get()
eg.Add(err1)
eg.Add(err2)
eg.Add(err3)

// Get standard errors.Join result
standardErr := eg.Join()

// Use with standard library functions
if errors.Is(standardErr, expectedErr) {
    // Handle specific error
}

var targetErr *MyCustomError
if errors.As(standardErr, &targetErr) {
    // Handle custom error type
}

// The joined error maintains ewrap capabilities
if ewrapGroup, ok := standardErr.(*ewrap.ErrorGroup); ok {
    // Can still serialize the group
    jsonOutput, _ := ewrapGroup.ToJSON(ewrap.WithTimestampFormat(time.RFC3339))
}
```

### Preserving ewrap Features

When using `errors.Join`, ewrap features are preserved:

```go
eg := pool.Get()
eg.Add(ewrap.New("error 1", ewrap.WithErrorType(ewrap.ErrorTypeDatabase)))
eg.Add(ewrap.New("error 2", ewrap.WithErrorType(ewrap.ErrorTypeNetwork)))

// Join preserves individual error metadata
joinedErr := eg.Join()

// Individual errors maintain their ewrap features
fmt.Printf("Joined error: %v\n", joinedErr)

// Can still access individual errors
for _, err := range eg.Errors() {
    if ewrapErr, ok := err.(*ewrap.Error); ok {
        fmt.Printf("Error type: %s, Severity: %s\n",
            ewrapErr.ErrorType(), ewrapErr.Severity())
    }
}
```

## Performance Optimizations

### Go 1.25+ Features

ewrap leverages modern Go features for efficient serialization:

```go
// Uses maps.Clone for efficient metadata copying
func (err *Error) Clone() *Error {
    cloned := &Error{
        message:   err.message,
        errorType: err.errorType,
        severity:  err.severity,
        timestamp: err.timestamp,
        stack:     err.stack,
        metadata:  maps.Clone(err.metadata), // Efficient copying
    }
    return cloned
}

// Uses slices.Clone for error group copying
func (eg *ErrorGroup) Clone() *ErrorGroup {
    return &ErrorGroup{
        errors: slices.Clone(eg.errors), // Efficient slice copying
        mutex:  sync.RWMutex{},
    }
}
```

### Memory Management

Serialization is optimized for memory efficiency:

```go
// Pre-allocated buffers for JSON marshaling
type SerializationBuffer struct {
    jsonBuffer  bytes.Buffer
    yamlBuffer  bytes.Buffer
}

// Reuse buffers across serialization operations
func (sb *SerializationBuffer) SerializeToJSON(err *Error) (string, error) {
    sb.jsonBuffer.Reset() // Reuse existing buffer

    encoder := json.NewEncoder(&sb.jsonBuffer)
    if err := encoder.Encode(err); err != nil {
        return "", err
    }

    return sb.jsonBuffer.String(), nil
}
```

## API Integration Examples

### REST API Error Responses

```go
func handleAPIError(w http.ResponseWriter, r *http.Request, err error) {
    if ewrapErr, ok := err.(*ewrap.Error); ok {
        jsonResponse, serErr := ewrapErr.ToJSON(
            ewrap.WithTimestampFormat(time.RFC3339),
            ewrap.WithStackTrace(false), // Don't expose stack traces in API
            ewrap.WithRecoverySuggestion(false)) // Keep suggestions internal

        if serErr != nil {
            http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")

        // Set appropriate status code based on error type
        statusCode := getStatusCodeForError(ewrapErr)
        w.WriteHeader(statusCode)

        w.Write([]byte(jsonResponse))
        return
    }

    // Fallback for non-ewrap errors
    http.Error(w, err.Error(), http.StatusInternalServerError)
}

func getStatusCodeForError(err *ewrap.Error) int {
    switch err.ErrorType() {
    case ewrap.ErrorTypeValidation:
        return http.StatusBadRequest
    case ewrap.ErrorTypeNotFound:
        return http.StatusNotFound
    case ewrap.ErrorTypePermission:
        return http.StatusForbidden
    case ewrap.ErrorTypeDatabase, ewrap.ErrorTypeNetwork:
        return http.StatusInternalServerError
    default:
        return http.StatusInternalServerError
    }
}
```

### Structured Logging Integration

```go
func logErrorGroup(logger *slog.Logger, eg *ewrap.ErrorGroup) {
    jsonOutput, err := eg.ToJSON(
        ewrap.WithTimestampFormat(time.RFC3339),
        ewrap.WithStackTrace(true),
        ewrap.WithRecoverySuggestion(true))

    if err != nil {
        logger.Error("failed to serialize error group", "error", err)
        return
    }

    logger.Error("error group occurred",
        "error_count", len(eg.Errors()),
        "errors", jsonOutput)
}
```

### Monitoring System Integration

```go
func sendToMonitoring(eg *ewrap.ErrorGroup, metricsClient *prometheus.Client) {
    for _, err := range eg.Errors() {
        if ewrapErr, ok := err.(*ewrap.Error); ok {
            // Send metrics
            metricsClient.Counter("errors_total").
                WithLabelValues(
                    string(ewrapErr.ErrorType()),
                    string(ewrapErr.Severity())).
                Inc()

            // Send structured data to monitoring
            jsonData, serErr := ewrapErr.ToJSON(
                ewrap.WithTimestampFormat(time.RFC3339),
                ewrap.WithStackTrace(false))

            if serErr == nil {
                metricsClient.SendCustomMetric("error_details", jsonData)
            }
        }
    }
}
```

The serialization features in ewrap provide comprehensive support for structured error export, enabling seamless integration with monitoring systems, APIs, and debugging workflows while maintaining excellent performance characteristics.
