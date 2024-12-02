# Error Wrapping

Error wrapping is a powerful feature that allows you to add context to errors as they propagate through your application. Understanding how to effectively wrap errors can significantly improve your application's debuggability and error handling capabilities.

## Understanding Error Wrapping

When an error occurs deep in your application's call stack, it often needs to pass through several layers before being handled. Each layer might need to add its own context to the error, helping to tell the complete story of what went wrong.

Consider this scenario:

```go
func getUserProfile(userID string) (*Profile, error) {
    // Low level database error occurs
    data, err := db.Query("SELECT * FROM users WHERE id = ?", userID)
    if err != nil {
        // We wrap the database error with our context
        return nil, ewrap.Wrap(err, "failed to fetch user data",
            ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError))
    }

    // Error occurs during data processing
    profile, err := parseUserData(data)
    if err != nil {
        // We wrap the parsing error with additional context
        return nil, ewrap.Wrap(err, "failed to parse user profile",
            ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityError)).
            WithMetadata("user_id", userID)
    }

    return profile, nil
}
```

## The Wrap Function

The `Wrap` function is the primary tool for error wrapping. It preserves the original error while adding new context:

```go
func Wrap(err error, msg string, opts ...Option) *Error
```

The function takes:

- The original error
- A message describing what went wrong at this level
- Optional configuration options

### Basic Usage

Here's a simple example of error wrapping:

```go
if err := validateInput(data); err != nil {
    return ewrap.Wrap(err, "input validation failed")
}
```

### Adding Context While Wrapping

You can add rich context while wrapping errors:

```go
if err := processPayment(amount); err != nil {
    return ewrap.Wrap(err, "payment processing failed",
        ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityCritical),
        ewrap.WithLogger(logger)).
        WithMetadata("amount", amount).
        WithMetadata("currency", "USD").
        WithMetadata("processor", "stripe")
}
```

## Error Chain Preservation

When you wrap an error, ewrap maintains the entire error chain. This means you can:

- Access the original error
- See all intermediate wrapping contexts
- Understand the complete error path

```go
func main() {
    err := processUserRequest()
    if err != nil {
        // Print the full error chain
        fmt.Println(err)

        // Access the root cause
        cause := errors.Unwrap(err)

        // Check if a specific error type exists in the chain
        if errors.Is(err, sql.ErrNoRows) {
            // Handle database not found case
        }
    }
}
```

## Formatted Wrapping with Wrapf

For cases where you need to include formatted messages, use `Wrapf`:

```go
func updateUser(userID string, fields map[string]interface{}) error {
    if err := db.Update(userID, fields); err != nil {
        return ewrap.Wrapf(err, "failed to update user %s", userID)
    }
    return nil
}
```

## Best Practices for Error Wrapping

### 1. Add Meaningful Context

Each wrap should add valuable information:

```go
// Good - adds specific context
err = ewrap.Wrap(err, "failed to process monthly report for January 2024",
    ewrap.WithMetadata("report_type", "monthly"),
    ewrap.WithMetadata("period", "2024-01"))

// Not as helpful - too generic
err = ewrap.Wrap(err, "processing failed")
```

### 2. Preserve Error Types

Choose error types that make sense for the current context:

```go
func validateAndSaveUser(user User) error {
    err := validateUser(user)
    if err != nil {
        // Preserve validation error type
        return ewrap.Wrap(err, "user validation failed",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError))
    }

    err = saveUser(user)
    if err != nil {
        // Use database error type for storage issues
        return ewrap.Wrap(err, "failed to save user",
            ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError))
    }

    return nil
}
```

### 3. Use Appropriate Granularity

Balance between too much and too little information:

```go
func processOrder(order Order) error {
    // Wrap high-level business operations
    if err := validateOrder(order); err != nil {
        return ewrap.Wrap(err, "order validation failed")
    }

    // Don't wrap every small utility function
    total := calculateTotal(order.Items)

    // Wrap significant state transitions or external calls
    if err := chargeCustomer(order.CustomerID, total); err != nil {
        return ewrap.Wrap(err, "payment processing failed",
            ewrap.WithMetadata("amount", total),
            ewrap.WithMetadata("customer_id", order.CustomerID))
    }

    return nil
}
```

### 4. Consider Performance

While error wrapping is lightweight, be mindful in hot paths:

```go
func processItems(items []Item) error {
    for _, item := range items {
        // In tight loops, consider if wrapping is necessary
        if err := validateItem(item); err != nil {
            return err  // Maybe don't wrap simple validation errors
        }

        // Do wrap significant errors
        if err := processItem(item); err != nil {
            return ewrap.Wrap(err, "item processing failed",
                ewrap.WithMetadata("item_id", item.ID))
        }
    }
    return nil
}
```

## Advanced Error Wrapping

### Conditional Wrapping

Sometimes you might want to wrap errors differently based on their type:

```go
func handleDatabaseOperation() error {
    err := db.Query()
    if err != nil {
        switch {
        case errors.Is(err, sql.ErrNoRows):
            return ewrap.Wrap(err, "record not found",
                ewrap.WithContext(ctx, ewrap.ErrorTypeNotFound, ewrap.SeverityWarning))
        case errors.Is(err, sql.ErrConnDone):
            return ewrap.Wrap(err, "database connection lost",
                ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityCritical))
        default:
            return ewrap.Wrap(err, "database operation failed",
                ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError))
        }
    }
    return nil
}
```

### Multi-Level Wrapping

For complex operations, you might wrap errors multiple times:

```go
func processUserOrder(ctx context.Context, userID, orderID string) error {
    user, err := getUser(userID)
    if err != nil {
        return ewrap.Wrap(err, "failed to get user",
            ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError))
    }

    order, err := getOrder(orderID)
    if err != nil {
        return ewrap.Wrap(err, "failed to get order",
            ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError))
    }

    if err := validateUserCanAccessOrder(user, order); err != nil {
        return ewrap.Wrap(err, "user not authorized to access order",
            ewrap.WithContext(ctx, ewrap.ErrorTypePermission, ewrap.SeverityWarning))
    }

    if err := processOrderPayment(order); err != nil {
        return ewrap.Wrap(err, "order payment failed",
            ewrap.WithContext(ctx, ewrap.ErrorTypeExternal, ewrap.SeverityCritical))
    }

    return nil
}
```
