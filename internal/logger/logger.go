// Package logger provides a standardized logging interface for the application.
// It defines a Logger interface that supports logging at different severity levels (Error, Debug, Info)
// with support for structured logging through key-value pairs.
//
// The logger package is designed to be implementation-agnostic, allowing different
// logging backends to be used as long as they implement the Logger interface.
package logger

// Logger defines the interface for error logging.
type Logger interface {
	// Error logs an error message with optional key-value pairs
	Error(msg string, keysAndValues ...any)
	// Debug logs a debug message with optional key-value pairs
	Debug(msg string, keysAndValues ...any)
	// Info logs an info message with optional key-value pairs
	Info(msg string, keysAndValues ...any)
}
