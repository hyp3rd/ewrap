// Package ewrap provides enhanced error handling capabilities with stack traces,
// error wrapping, custom error types, and logging integration.
package ewrap

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/hyp3rd/ewrap/internal/logger"
)

const (
	baseLogDataSize = 4 // error, msg, stack, and potentially cause
	runtimeCallers  = 3
)

// Error represents a custom error type with stack trace and metadata.
type Error struct {
	msg      string
	cause    error
	stack    []uintptr
	metadata map[string]interface{}
	logger   logger.Logger
	mu       sync.RWMutex // Protects metadata and logger
}

// Option defines the signature for configuration options.
type Option func(*Error)

// WithLogger sets a logger for the error.
func WithLogger(logger logger.Logger) Option {
	return func(err *Error) {
		err.mu.Lock()
		err.logger = logger
		err.mu.Unlock()

		// Log error creation if logger is available
		if logger != nil {
			logger.Debug("error created",
				"message", err.msg,
				"stack", err.Stack(),
			)
		}
	}
}

// New creates a new Error with a stack trace and applies the provided options.
func New(msg string, opts ...Option) *Error {
	err := &Error{
		msg:      msg,
		stack:    CaptureStack(),
		metadata: make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(err)
	}

	return err
}

// Wrap wraps an existing error with additional context and stack trace.
func Wrap(err error, msg string, opts ...Option) *Error {
	if err == nil {
		return nil
	}

	var (
		stack      []uintptr
		metadata   map[string]interface{}
		wrappedErr *Error
	)
	// If the error is already wrapped, preserve its stack trace and metadata
	if errors.As(err, &wrappedErr) {
		wrappedErr.mu.RLock()

		stack = wrappedErr.stack
		// Create a new metadata map with the existing values
		metadata = make(map[string]interface{}, len(wrappedErr.metadata))

		for k, v := range wrappedErr.metadata {
			metadata[k] = v
		}

		wrappedErr.mu.RUnlock()
	} else {
		stack = CaptureStack()
		metadata = make(map[string]interface{})
	}

	wrapped := &Error{
		msg:      msg,
		cause:    err,
		stack:    stack,
		metadata: metadata,
	}

	for _, opt := range opts {
		opt(wrapped)
	}

	return wrapped
}

// Wrapf wraps an error with a formatted message.
func Wrapf(err error, format string, args ...interface{}) *Error {
	if err == nil {
		return nil
	}

	return Wrap(err, fmt.Sprintf(format, args...))
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.msg, e.cause)
	}

	return e.msg
}

// Cause returns the underlying cause of the error.
func (e *Error) Cause() error {
	return e.cause
}

// WithMetadata adds metadata to the error.
func (e *Error) WithMetadata(key string, value interface{}) *Error {
	e.mu.Lock()
	e.metadata[key] = value

	if e.logger != nil {
		e.logger.Debug("metadata added",
			"key", key,
			"value", value,
			"error", e.msg,
		)
	}

	e.mu.Unlock()

	return e
}

// GetMetadata retrieves metadata from the error.
func (e *Error) GetMetadata(key string) (interface{}, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	val, ok := e.metadata[key]

	return val, ok
}

// Stack returns the stack trace as a string.
func (e *Error) Stack() string {
	var builder strings.Builder

	frames := runtime.CallersFrames(e.stack)

	for {
		frame, more := frames.Next()
		// Skip runtime frames and error package frames
		if !strings.Contains(frame.File, "runtime/") && !strings.Contains(frame.File, "ewrap/errors.go") {
			_, _ = fmt.Fprintf(&builder, "%s:%d - %s\n", frame.File, frame.Line, frame.Function)
		}

		if !more {
			break
		}
	}

	return builder.String()
}

// Log logs the error using the configured logger.
func (e *Error) Log() {
	e.mu.RLock()
	logger := e.logger
	e.mu.RUnlock()

	if logger == nil {
		return
	}

	// Create a metadata map for logging
	logData := make([]interface{}, 0, len(e.metadata)*2+baseLogDataSize)
	logData = append(logData, "error", e.msg)

	if e.cause != nil {
		logData = append(logData, "cause", e.cause.Error())
	}

	logData = append(logData, "stack", e.Stack())

	e.mu.RLock()

	for k, v := range e.metadata {
		logData = append(logData, k, v)
	}

	e.mu.RUnlock()

	logger.Error("error occurred", logData...)
}

// CaptureStack captures the current stack trace.
func CaptureStack() []uintptr {
	const depth = 32

	var pcs [depth]uintptr
	n := runtime.Callers(runtimeCallers, pcs[:])

	return pcs[:n]
}

// Is reports whether target matches err in the error chain.
func (e *Error) Is(target error) bool {
	if target == nil {
		return false
	}

	err := e
	for err != nil {
		if err.msg == target.Error() {
			return true
		}

		if err.cause == nil {
			return false
		}

		if causeErr, ok := err.cause.(*Error); ok {
			err = causeErr

			continue
		}

		return err.cause.Error() == target.Error()
	}

	return false
}

// Unwrap provides compatibility with Go 1.13 error chains.
func (e *Error) Unwrap() error {
	return e.cause
}
