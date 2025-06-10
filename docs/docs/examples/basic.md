# Basic Examples

This guide provides practical examples of common error handling scenarios using ewrap. Each example demonstrates a specific feature or pattern, helping you understand how to use the package effectively.

## Simple Error Creation and Handling

Let's start with basic error creation and handling:

```go
package main

import (
    "context"
    "fmt"

    "github.com/hyp3rd/ewrap"
)

func main() {
    if err := processUserRegistration("john.doe@example.com"); err != nil {
        fmt.Printf("Registration failed: %v\n", err)
        return
    }
    fmt.Println("Registration successful")
}

func processUserRegistration(email string) error {
    // Create a context for the operation
    ctx := context.Background()

    // Validate email
    if err := validateEmail(email); err != nil {
        return ewrap.Wrap(err, "email validation failed",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError))
    }

    // Create user in database
    if err := createUser(email); err != nil {
        return ewrap.Wrap(err, "user creation failed",
            ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError))
    }

    return nil
}

func validateEmail(email string) error {
    if !strings.Contains(email, "@") {
        return ewrap.New("invalid email format")
    }
    return nil
}

func createUser(email string) error {
    // Simulate database operation
    return nil
}
```

## Error Groups for Multiple Validations

Here's how to collect multiple validation errors:

```go
type User struct {
    Email     string
    Password  string
    Age       int
    Username  string
}

func validateUser(ctx context.Context, user User) error {
    // Get an error group from the pool
    pool := ewrap.NewErrorGroupPool(4)
    eg := pool.Get()
    defer eg.Release()

    // Validate email
    if err := validateEmail(user.Email); err != nil {
        eg.Add(ewrap.Wrap(err, "invalid email",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError)))
    }

    // Validate password
    if len(user.Password) < 8 {
        eg.Add(ewrap.New("password too short",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError)).
            WithMetadata("min_length", 8))
    }

    // Validate age
    if user.Age < 18 {
        eg.Add(ewrap.New("user must be 18 or older",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError)).
            WithMetadata("provided_age", user.Age))
    }

    return eg.Error()
}
```

## Logging Integration

Example showing basic logging integration:

```go
type AppLogger struct {
    logger *zap.Logger
}

func (l *AppLogger) Error(msg string, keysAndValues ...any) {
    l.logger.Error(msg, convertToZapFields(keysAndValues...)...)
}

func (l *AppLogger) Debug(msg string, keysAndValues ...any) {
    l.logger.Debug(msg, convertToZapFields(keysAndValues...)...)
}

func (l *AppLogger) Info(msg string, keysAndValues ...any) {
    l.logger.Info(msg, convertToZapFields(keysAndValues...)...)
}

func processWithLogging(ctx context.Context, data []byte) error {
    logger := &AppLogger{logger: zapLogger}

    err := processData(data)
    if err != nil {
        return ewrap.Wrap(err, "data processing failed",
            ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityError),
            ewrap.WithLogger(logger))
    }

    return nil
}
```

## HTTP Handler Example

Using ewrap in an HTTP handler:

```go
func handleUserRegistration(w http.ResponseWriter, r *http.Request) {
    // Create request context with ID
    ctx := r.Context()
    requestID := generateRequestID()
    ctx = context.WithValue(ctx, "request_id", requestID)

    var user User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        respondWithError(w, ewrap.Wrap(err, "invalid request body",
            ewrap.WithContext(ctx, ewrap.ErrorTypeValidation, ewrap.SeverityError)))
        return
    }

    if err := validateUser(ctx, user); err != nil {
        respondWithError(w, err)
        return
    }

    if err := createUser(ctx, user); err != nil {
        respondWithError(w, err)
        return
    }

    respondWithJSON(w, http.StatusCreated, map[string]string{
        "message": "user created successfully",
    })
}

func respondWithError(w http.ResponseWriter, err error) {
    if wrappedErr, ok := err.(*ewrap.Error); ok {
        // Convert error to API response
        response := ErrorResponse{
            Message: wrappedErr.Error(),
            Code:    getErrorCode(wrappedErr),
            Details: getErrorDetails(wrappedErr),
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(getHTTPStatus(wrappedErr))
        json.NewEncoder(w).Encode(response)
    } else {
        // Handle unwrapped errors
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]string{
            "message": "internal server error",
        })
    }
}
```

## Database Operations

Example showing database error handling:

```go
func getUserByID(ctx context.Context, userID string) (*User, error) {
    var user User
    err := db.QueryRow("SELECT * FROM users WHERE id = $1", userID).Scan(&user)

    switch {
    case err == sql.ErrNoRows:
        return nil, ewrap.New("user not found",
            ewrap.WithContext(ctx, ewrap.ErrorTypeNotFound, ewrap.SeverityWarning)).
            WithMetadata("user_id", userID)
    case err != nil:
        return nil, ewrap.Wrap(err, "database query failed",
            ewrap.WithContext(ctx, ewrap.ErrorTypeDatabase, ewrap.SeverityError)).
            WithMetadata("user_id", userID)
    }

    return &user, nil
}
```

## Cleanup and Deferred Operations

Example showing error handling with cleanup:

```go
func processFile(ctx context.Context, path string) error {
    file, err := os.Open(path)
    if err != nil {
        return ewrap.Wrap(err, "failed to open file",
            ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityError))
    }
    defer func() {
        if closeErr := file.Close(); closeErr != nil {
            err = ewrap.Wrap(closeErr, "failed to close file",
                ewrap.WithContext(ctx, ewrap.ErrorTypeInternal, ewrap.SeverityWarning))
        }
    }()

    // Process file...
    return nil
}
```
