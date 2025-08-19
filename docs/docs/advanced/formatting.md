# Error Formatting

Error formatting in ewrap provides flexible ways to present errors in different formats and contexts with proper timestamp formatting and enhanced serialization capabilities. This capability is crucial for logging, debugging, API responses, and system integration. Let's explore how to effectively format errors to meet various needs in your application.

## Understanding Error Formatting

When an error occurs in your system, you might need to present it in different ways depending on the context:

- As JSON for API responses with proper timestamp formatting
- As YAML for configuration-related errors
- As structured text for logging with recovery suggestions
- As user-friendly messages for end users
- As serialized error groups for monitoring systems

ewrap provides comprehensive formatting options to handle all these cases while maintaining the rich context and metadata associated with your errors.

## Enhanced JSON Formatting

JSON formatting now includes proper timestamp formatting and recovery suggestions:

```go
func handleAPIError(w http.ResponseWriter, err error) {
    if wrappedErr, ok := err.(*ewrap.Error); ok {
        // Convert error to JSON with enhanced formatting
        jsonOutput, err := wrappedErr.ToJSON(
            ewrap.WithTimestampFormat(time.RFC3339), // Proper timestamp formatting
            ewrap.WithStackTrace(true),
            ewrap.WithRecoverySuggestion(true), // Include recovery guidance
        )

        if err != nil {
            // Handle formatting error
            http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte(jsonOutput))
    }
}
```

### Enhanced JSON Structure

The resulting JSON now includes proper timestamp formatting and recovery suggestions:

```json
{
    "message": "failed to process user order",
    "timestamp": "2024-03-15T14:30:00Z",
    "type": "database",
    "severity": "critical",
    "stack_trace": [
        {
            "function": "main.processOrder",
            "file": "/app/main.go",
            "line": 42,
            "pc": "0x4567890"
        }
    ],
    "metadata": {
        "user_id": "12345",
        "order_id": "ord_789",
        "retry_count": 3
    },
    "recovery_suggestion": "Check database connectivity and retry with exponential backoff"
}
```

## Timestamp Formatting Options

### Standard Formats

ewrap supports various timestamp formats:

```go
// RFC3339 format (recommended for APIs and JSON)
jsonOutput, _ := err.ToJSON(ewrap.WithTimestampFormat(time.RFC3339))
// Result: "2024-03-15T14:30:00Z"

// RFC3339Nano for high precision
jsonOutput, _ := err.ToJSON(ewrap.WithTimestampFormat(time.RFC3339Nano))
// Result: "2024-03-15T14:30:00.123456789Z"

// Kitchen format for human-readable logs
jsonOutput, _ := err.ToJSON(ewrap.WithTimestampFormat(time.Kitchen))
// Result: "2:30PM"

// Custom format for specific requirements
jsonOutput, _ := err.ToJSON(ewrap.WithTimestampFormat("2006-01-02 15:04:05"))
// Result: "2024-03-15 14:30:00"
```

### Unix Timestamp Support

For systems integration requiring Unix timestamps:

```go
// Unix timestamp (seconds since epoch)
jsonOutput, _ := err.ToJSON(ewrap.WithTimestampFormat("unix"))
// Result: "1710507000"

// Unix timestamp with milliseconds
jsonOutput, _ := err.ToJSON(ewrap.WithTimestampFormat("unix_milli"))
// Result: "1710507000123"

// Unix timestamp with microseconds
jsonOutput, _ := err.ToJSON(ewrap.WithTimestampFormat("unix_micro"))
// Result: "1710507000123456"
```

## YAML Formatting with Recovery Suggestions

YAML formatting now supports recovery suggestions and enhanced metadata:

```go
func logConfigurationError(err error, logger Logger) {
    if wrappedErr, ok := err.(*ewrap.Error); ok {
        // Convert error to YAML with recovery suggestions
        yamlOutput, err := wrappedErr.ToYAML(
            ewrap.WithTimestampFormat(time.RFC3339),
            ewrap.WithStackTrace(true),
            ewrap.WithRecoverySuggestion(true),
        )

        if err != nil {
            logger.Error("failed to format error", "error", err)
            return
        }

        logger.Error("configuration error occurred", "details", yamlOutput)
    }
}
```

### Enhanced YAML Structure

The formatted YAML now includes recovery suggestions:

```yaml
message: failed to load configuration
timestamp: "2024-03-15T14:30:00Z"
type: configuration
severity: critical
stack_trace:
  - function: main.loadConfig
    file: /app/config.go
    line: 25
    pc: "0x4567890"
  - function: main.initialize
    file: /app/main.go
    line: 15
    pc: "0x4567891"
metadata:
    config_file: /etc/myapp/config.yaml
    invalid_fields:
        - database.host
        - database.port
recovery_suggestion: "Check configuration file format and validate required fields"
```

## Custom Formatting

Sometimes you need to create custom formats for specific use cases. Here's how to build custom formatters:

```go
type ErrorFormatter struct {
    TimestampFormat string
    IncludeStack    bool
    IncludeMetadata bool
    MaxStackDepth   int
}

func NewErrorFormatter() *ErrorFormatter {
    return &ErrorFormatter{
        TimestampFormat: time.RFC3339,
        IncludeStack:    true,
        IncludeMetadata: true,
        MaxStackDepth:   10,
    }
}

func (f *ErrorFormatter) Format(err *ewrap.Error) map[string]any {
    // Create base error information
    formatted := map[string]any{
        "message":   err.Error(),
        "timestamp": time.Now().Format(f.TimestampFormat),
    }

    // Add stack trace if enabled
    if f.IncludeStack {
        formatted["stack"] = f.formatStack(err.Stack())
    }

    // Add metadata if enabled
    if f.IncludeMetadata {
        metadata := make(map[string]any)
        // Extract and format metadata...
        formatted["metadata"] = metadata
    }

    return formatted
}

func (f *ErrorFormatter) formatStack(stack string) []string {
    lines := strings.Split(stack, "\n")
    if len(lines) > f.MaxStackDepth {
        lines = lines[:f.MaxStackDepth]
    }
    return lines
}
```

## User-Friendly Error Messages

When presenting errors to end users, you often need to transform technical errors into user-friendly messages while preserving the technical details for logging:

```go
type UserErrorFormatter struct {
    translations map[ErrorType]string
    logger       Logger
}

func NewUserErrorFormatter(logger Logger) *UserErrorFormatter {
    return &UserErrorFormatter{
        translations: map[ErrorType]string{
            ErrorTypeValidation:    "The provided information is invalid",
            ErrorTypeNotFound:      "The requested resource could not be found",
            ErrorTypePermission:    "You don't have permission to perform this action",
            ErrorTypeDatabase:      "A system error occurred",
            ErrorTypeNetwork:       "Connection issues detected",
            ErrorTypeConfiguration: "System configuration error",
        },
        logger: logger,
    }
}

func (f *UserErrorFormatter) FormatForUser(err error) string {
    // Always log the full technical error
    if wrappedErr, ok := err.(*ewrap.Error); ok {
        f.logger.Error("error occurred",
            "technical_details", wrappedErr.ToJSON())

        // Get error context
        ctx := getErrorContext(wrappedErr)

        // Return translated message
        if msg, ok := f.translations[ctx.Type]; ok {
            return msg
        }
    }

    // Default message for unknown errors
    return "An unexpected error occurred"
}
```

## Best Practices for Error Formatting

### 1. Security-Conscious Formatting

Be careful about what information you expose in different contexts:

```go
func formatErrorResponse(err error, internal bool) any {
    wrappedErr, ok := err.(*ewrap.Error)
    if !ok {
        return map[string]string{"message": "Internal Server Error"}
    }

    if internal {
        // Full details for internal logging
        return map[string]any{
            "message":   wrappedErr.Error(),
            "stack":     wrappedErr.Stack(),
            "metadata":  wrappedErr.GetAllMetadata(),
            "type":      getErrorContext(wrappedErr).Type,
            "severity": getErrorContext(wrappedErr).Severity,
        }
    }

    // Limited information for external responses
    return map[string]string{
        "message": sanitizeErrorMessage(wrappedErr.Error()),
        "code":    getPublicErrorCode(wrappedErr),
    }
}
```

### 2. Consistent Format Structure

Maintain consistent error format structures across your application:

```go
type StandardErrorResponse struct {
    Message   string                 `json:"message"`
    Code      string                 `json:"code"`
    Details   map[string]any `json:"details,omitempty"`
    RequestID string                 `json:"request_id,omitempty"`
    Timestamp string                 `json:"timestamp"`
}

func NewStandardErrorResponse(err error, requestID string) StandardErrorResponse {
    return StandardErrorResponse{
        Message:   getErrorMessage(err),
        Code:      getErrorCode(err),
        Details:   getErrorDetails(err),
        RequestID: requestID,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
}
```

### 3. Context-Aware Formatting

Adjust formatting based on the execution context:

```go
func formatErrorByEnvironment(err error, env string) any {
    switch env {
    case "development":
        // Include everything in development
        return formatWithFullDetails(err)
    case "testing":
        // Include stack traces but sanitize sensitive data
        return formatForTesting(err)
    case "production":
        // Minimal public information
        return formatForProduction(err)
    default:
        return formatWithDefaultSettings(err)
    }
}
```
