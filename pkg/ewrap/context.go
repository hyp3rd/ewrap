package ewrap

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"
)

const (
	errorContextRuntimeCallers = 2
)

// ErrorContext holds comprehensive information about an error's context.
type ErrorContext struct {
	// Timestamp when the error occurred.
	Timestamp time.Time
	// Type categorizes the error.
	Type ErrorType
	// Severity indicates the error's impact level.
	Severity Severity
	// Operation that was being performed.
	Operation string
	// Component where the error originated.
	Component string
	// RequestID for tracing.
	RequestID string
	// User associated with the operation.
	User string
	// Environment where the error occurred.
	Environment string
	// Version of the application.
	Version string
	// File and line where the error occurred.
	File string
	Line int
	// Additional context-specific data.
	Data map[string]interface{}
}

// newErrorContext creates a new ErrorContext with basic information.
func newErrorContext(ctx context.Context, errorType ErrorType, severity Severity) *ErrorContext {
	_, file, line, _ := runtime.Caller(errorContextRuntimeCallers)

	errorCtx := &ErrorContext{
		Timestamp:   time.Now(),
		Type:        errorType,
		Severity:    severity,
		File:        file,
		Line:        line,
		Data:        make(map[string]interface{}),
		Environment: getEnvironment(),
	}

	if ctx != nil {
		// Extract common context values.
		if reqID, ok := ctx.Value("request_id").(string); ok {
			errorCtx.RequestID = reqID
		}

		if user, ok := ctx.Value("user").(string); ok {
			errorCtx.User = user
		}

		if op, ok := ctx.Value("operation").(string); ok {
			errorCtx.Operation = op
		}

		if component, ok := ctx.Value("component").(string); ok {
			errorCtx.Component = component
		}
	}

	return errorCtx
}

// String returns a formatted string representation of the error context.
func (ec *ErrorContext) String() string {
	return fmt.Sprintf(
		"[%s] %s error in %s:%d (%s) - %s - RequestID: %s, User: %s",
		ec.Severity,
		ec.Type,
		ec.File,
		ec.Line,
		ec.Component,
		ec.Operation,
		ec.RequestID,
		ec.User,
	)
}

// WithContext adds context information to the error.
func WithContext(ctx context.Context, errorType ErrorType, severity Severity) Option {
	return func(err *Error) {
		errorCtx := newErrorContext(ctx, errorType, severity)

		err.mu.Lock()
		err.metadata["error_context"] = errorCtx
		err.mu.Unlock()

		if err.logger != nil {
			err.logger.Debug("error context added",
				"error_type", errorType,
				"severity", severity,
				"request_id", errorCtx.RequestID,
				"component", errorCtx.Component,
			)
		}
	}
}

// getEnvironment determines the current runtime environment.
func getEnvironment() string {
	if env := os.Getenv("APP_ENV"); env != "" {
		return env
	}

	return "development"
}
