# Error Types

Error types in ewrap provide a structured way to categorize and handle different kinds of errors in your application. Understanding error types helps you make better decisions about error handling, logging, and recovery strategies.

## Understanding Error Types

Error types serve multiple purposes:

1. They help categorize errors meaningfully
2. They enable consistent error handling across your application
3. They facilitate automated error processing and reporting
4. They guide recovery strategies and user feedback

Let's explore the built-in error types and learn how to use them effectively:

```go
type ErrorType int

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

## Using Error Types

Error types are most powerful when combined with context and metadata. Here's how to use them effectively:

```go
func validateAndProcessUser(ctx context.Context, user User) error {
    // Validation errors use ErrorTypeValidation
    if err := validateUser(user); err != nil {
        return ewrap.Wrap(err, "user validation failed",
            ewrap.WithContext(ctx, ErrorTypeValidation, SeverityError)).
            WithMetadata("validation_fields", getFailedFields(err))
    }

    // Database errors use ErrorTypeDatabase
    if err := saveUser(user); err != nil {
        return ewrap.Wrap(err, "failed to save user",
            ewrap.WithContext(ctx, ErrorTypeDatabase, SeverityCritical)).
            WithMetadata("user_id", user.ID)
    }

    // External service errors use ErrorTypeExternal
    if err := notifyUserService(user); err != nil {
        return ewrap.Wrap(err, "failed to notify user service",
            ewrap.WithContext(ctx, ErrorTypeExternal, SeverityWarning)).
            WithMetadata("service", "notification")
    }

    return nil
}
```

## Error Type Patterns

Different error types often require different handling strategies. Here's a comprehensive example:

```go
func handleError(err error) {
    wrappedErr, ok := err.(*ewrap.Error)
    if !ok {
        // Handle plain errors
        return
    }

    ctx := getErrorContext(wrappedErr)
    errorType := ctx.Type
    severity := ctx.Severity

    switch errorType {
    case ErrorTypeValidation:
        // Validation errors often need user feedback
        handleValidationError(wrappedErr)

    case ErrorTypeDatabase:
        // Database errors might need retry logic
        if severity == SeverityCritical {
            notifyDatabaseAdmin(wrappedErr)
        }
        attemptDatabaseRecovery(wrappedErr)

    case ErrorTypeNetwork:
        // Network errors often benefit from circuit breaking
        handleNetworkError(wrappedErr)

    case ErrorTypePermission:
        // Permission errors need security logging
        logSecurityEvent(wrappedErr)

    default:
        // Unknown errors need investigation
        logUnexpectedError(wrappedErr)
    }
}

func handleValidationError(err *ewrap.Error) {
    // Extract validation details for user feedback
    fields, _ := err.GetMetadata("validation_fields")
    userMessage := buildUserFriendlyMessage(fields)

    // Log for debugging but don't alert
    logger.Debug("validation error occurred",
        "fields", fields,
        "user_message", userMessage)
}

func handleDatabaseError(err *ewrap.Error) {
    // Check if error is retryable
    if isRetryableError(err) {
        retryWithBackoff(func() error {
            // Retry the operation
            return nil
        })
    }

    // Log critical database errors
    logger.Error("database error occurred",
        "error", err,
        "stack", err.Stack())
}
```

## Custom Error Types

Sometimes you need domain-specific error types. Here's how to extend the system:

```go
// Define custom error types
const (
    ErrorTypePayment ErrorType = iota + 100  // Start after built-in types
    ErrorTypeInventory
    ErrorTypeShipping
)

// Create a type registry
type ErrorTypeRegistry struct {
    types map[ErrorType]string
    mu    sync.RWMutex
}

func NewErrorTypeRegistry() *ErrorTypeRegistry {
    return &ErrorTypeRegistry{
        types: make(map[ErrorType]string),
    }
}

func (r *ErrorTypeRegistry) Register(et ErrorType, name string) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.types[et] = name
}

func (r *ErrorTypeRegistry) GetName(et ErrorType) string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    if name, ok := r.types[et]; ok {
        return name
    }
    return "unknown"
}

// Usage example
func initErrorTypes() *ErrorTypeRegistry {
    registry := NewErrorTypeRegistry()

    // Register custom error types
    registry.Register(ErrorTypePayment, "payment")
    registry.Register(ErrorTypeInventory, "inventory")
    registry.Register(ErrorTypeShipping, "shipping")

    return registry
}
```

## Error Type Best Practices

### 1. Consistent Type Assignment

Be consistent in how you assign error types:

```go
// Good - consistent error typing
func processOrder(order Order) error {
    if err := validateOrder(order); err != nil {
        return ewrap.Wrap(err, "order validation failed",
            ewrap.WithErrorType(ErrorTypeValidation))
    }

    if err := checkInventory(order); err != nil {
        return ewrap.Wrap(err, "inventory check failed",
            ewrap.WithErrorType(ErrorTypeInventory))
    }

    if err := processPayment(order); err != nil {
        return ewrap.Wrap(err, "payment processing failed",
            ewrap.WithErrorType(ErrorTypePayment))
    }

    return nil
}

// Avoid - inconsistent or missing error types
func processOrder(order Order) error {
    if err := validateOrder(order); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    // ...
}
```

### 2. Error Type Hierarchy

Consider creating error type hierarchies for complex domains:

```go
// Define error type hierarchy
type ErrorCategory int

const (
    CategoryValidation ErrorCategory = iota
    CategoryInfrastructure
    CategoryBusiness
)

type DomainErrorType struct {
    Type     ErrorType
    Category ErrorCategory
    Retryable bool
}

var errorTypeRegistry = map[ErrorType]DomainErrorType{
    ErrorTypeValidation: {
        Category: CategoryValidation,
        Retryable: false,
    },
    ErrorTypeDatabase: {
        Category: CategoryInfrastructure,
        Retryable: true,
    },
    ErrorTypePayment: {
        Category: CategoryBusiness,
        Retryable: true,
    },
}
```
