package ewrap

// Logger defines the minimal logging interface ewrap depends on. Any logging
// library can satisfy it with a small adapter; no external logger is bundled.
//
// Implementations must accept structured key/value pairs as the variadic
// arguments. Adapters for slog live in subpackages; for zap, zerolog, logrus,
// users write their own (≤10 lines) and pass them via WithLogger.
type Logger interface {
	// Error logs an error message with optional key-value pairs.
	Error(msg string, keysAndValues ...any)
	// Debug logs a debug message with optional key-value pairs.
	Debug(msg string, keysAndValues ...any)
	// Info logs an info message with optional key-value pairs.
	Info(msg string, keysAndValues ...any)
}
