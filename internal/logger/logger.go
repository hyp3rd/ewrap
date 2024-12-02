package logger

// Logger defines the interface for error logging.
type Logger interface {
	// Error logs an error message with optional key-value pairs
	Error(msg string, keysAndValues ...interface{})
	// Debug logs a debug message with optional key-value pairs
	Debug(msg string, keysAndValues ...interface{})
	// Info logs an info message with optional key-value pairs
	Info(msg string, keysAndValues ...interface{})
}
