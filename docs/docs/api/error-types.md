# Error Types and Severity

Understanding error types and severity levels is fundamental to using ewrap effectively. This guide explains the built-in error categorization system and how to leverage it for better error handling.

## Error Types Explained

Error types in ewrap are more than just labels - they represent distinct categories of failures that can occur in your application. Each type suggests different handling strategies and helps maintain consistency in how errors are processed throughout your system.

### Built-in Error Types

```go
const (
    ErrorTypeUnknown ErrorType = iota
    ErrorTypeValidation
    ErrorTypeNotFound
    ErrorTypePermission
    ErrorTypeDatabase
    ErrorTypeNetwork
    ErrorTypeConfiguration
    ErrorTypeInternal
    ErrorTypeExternal
)
```

Let's explore each error type and its intended use:

### ErrorTypeUnknown

Used when an error doesn't clearly fit into other categories. While it's available as a fallback, you should try to use more specific error types when possible.

```go
err := ewrap.New("unexpected error occurred",
    ewrap.WithContext(ctx, ErrorTypeUnknown, SeverityError))
```

### ErrorTypeValidation

For errors related to input validation, data formatting, or business rule violations. These errors typically indicate that the request or data being processed doesn't meet required criteria.

```go
func validateUser(user User) error {
    if user.Age < 18 {
        return ewrap.New("user must be 18 or older",
            ewrap.WithContext(ctx, ErrorTypeValidation, SeverityError)).
            WithMetadata("provided_age", user.Age).
            WithMetadata("minimum_age", 18)
    }
    return nil
}
```

### ErrorTypeNotFound

Indicates that a requested resource doesn't exist. This is particularly useful in API endpoints and database operations.

```go
func getUser(ctx context.Context, userID string) (*User, error) {
    user, err := db.FindUser(userID)
    if err == sql.ErrNoRows {
        return nil, ewrap.New("user not found",
            ewrap.WithContext(ctx, ErrorTypeNotFound, SeverityWarning)).
            WithMetadata("user_id", userID)
    }
    return user, err
}
```

### ErrorTypePermission

For authorization and authentication failures. These errors indicate that the operation failed due to insufficient permissions or invalid credentials.

```go
func validateAccess(ctx context.Context, userID string, resource string) error {
    if !hasPermission(userID, resource) {
        return ewrap.New("access denied",
            ewrap.WithContext(ctx, ErrorTypePermission, SeverityWarning)).
            WithMetadata("user_id", userID).
            WithMetadata("resource", resource).
            WithMetadata("required_role", "admin")
    }
    return nil
}
```

### ErrorTypeDatabase

Used for database-related errors, including connection issues, query failures, and transaction problems.

```go
func saveUserData(ctx context.Context, user User) error {
    if err := db.Insert(user); err != nil {
        return ewrap.Wrap(err, "failed to save user data",
            ewrap.WithContext(ctx, ErrorTypeDatabase, SeverityCritical)).
            WithMetadata("table", "users").
            WithMetadata("operation", "insert")
    }
    return nil
}
```

### ErrorTypeNetwork

For network-related failures, including API calls, service communication, and connectivity issues.

```go
func callExternalAPI(ctx context.Context, endpoint string) error {
    resp, err := http.Get(endpoint)
    if err != nil {
        return ewrap.Wrap(err, "API call failed",
            ewrap.WithContext(ctx, ErrorTypeNetwork, SeverityError)).
            WithMetadata("endpoint", endpoint).
            WithMetadata("timeout_seconds", 30)
    }
    return nil
}
```

### ErrorTypeConfiguration

Used when errors occur due to misconfiguration or invalid settings.

```go
func loadConfig(ctx context.Context, path string) (*Config, error) {
    cfg, err := parseConfig(path)
    if err != nil {
        return nil, ewrap.Wrap(err, "invalid configuration",
            ewrap.WithContext(ctx, ErrorTypeConfiguration, SeverityCritical)).
            WithMetadata("config_path", path).
            WithMetadata("invalid_fields", getInvalidFields(err))
    }
    return cfg, nil
}
```

### ErrorTypeInternal

For internal system errors that aren't caused by external factors or user input.

```go
func processData(ctx context.Context, data []byte) error {
    if err := internalProcess(data); err != nil {
        return ewrap.Wrap(err, "internal processing failed",
            ewrap.WithContext(ctx, ErrorTypeInternal, SeverityCritical)).
            WithMetadata("process_id", getCurrentProcessID()).
            WithMetadata("memory_usage", getMemoryUsage())
    }
    return nil
}
```

### ErrorTypeExternal

For errors originating from external services or systems.

```go
func callPaymentProvider(ctx context.Context, payment Payment) error {
    result, err := paymentProvider.Process(payment)
    if err != nil {
        return ewrap.Wrap(err, "payment processing failed",
            ewrap.WithContext(ctx, ErrorTypeExternal, SeverityCritical)).
            WithMetadata("provider", "stripe").
            WithMetadata("payment_id", payment.ID).
            WithMetadata("status_code", result.StatusCode)
    }
    return nil
}
```

## Severity Levels

Severity levels help indicate the impact and urgency of an error. ewrap provides four severity levels:

```go
const (
    SeverityInfo Severity = iota
    SeverityWarning
    SeverityError
    SeverityCritical
)
```

### Using Severity Levels Effectively

The choice of severity level should reflect the impact of the error on your system:

### SeverityInfo

For informational messages that don't indicate a problem but might be useful for debugging or monitoring.

```go
func auditAction(ctx context.Context, action string) error {
    return ewrap.New("action audited",
        ewrap.WithContext(ctx, ErrorTypeInternal, SeverityInfo)).
        WithMetadata("action", action).
        WithMetadata("timestamp", time.Now())
}
```

### SeverityWarning

For issues that don't prevent the system from functioning but require attention.

```go
func checkDiskSpace(ctx context.Context) error {
    if usage := getDiskUsage(); usage > 80 {
        return ewrap.New("high disk usage detected",
            ewrap.WithContext(ctx, ErrorTypeInternal, SeverityWarning)).
            WithMetadata("usage_percentage", usage).
            WithMetadata("threshold", 80)
    }
    return nil
}
```

### SeverityError

For significant issues that prevent a specific operation from completing successfully.

```go
func processOrder(ctx context.Context, order Order) error {
    if err := validateOrder(order); err != nil {
        return ewrap.Wrap(err, "order validation failed",
            ewrap.WithContext(ctx, ErrorTypeValidation, SeverityError)).
            WithMetadata("order_id", order.ID)
    }
    return nil
}
```

### SeverityCritical

For severe issues that might affect system stability or require immediate attention.

```go
func initializeDatabase(ctx context.Context) error {
    if err := db.Connect(); err != nil {
        return ewrap.Wrap(err, "database initialization failed",
            ewrap.WithContext(ctx, ErrorTypeDatabase, SeverityCritical)).
            WithMetadata("retry_count", 3).
            WithMetadata("last_error", err.Error())
    }
    return nil
}
```
