package slog

import "log/slog"

// Adapter wraps a *slog.Logger so it can be passed to ewrap.WithLogger.
type Adapter struct {
	logger *slog.Logger
}

// New returns an Adapter backed by logger.
func New(logger *slog.Logger) *Adapter {
	return &Adapter{logger: logger}
}

// Error logs an error message with optional key-value pairs.
func (a *Adapter) Error(msg string, keysAndValues ...any) {
	a.logger.Error(msg, keysAndValues...)
}

// Debug logs a debug message with optional key-value pairs.
func (a *Adapter) Debug(msg string, keysAndValues ...any) {
	a.logger.Debug(msg, keysAndValues...)
}

// Info logs an info message with optional key-value pairs.
func (a *Adapter) Info(msg string, keysAndValues ...any) {
	a.logger.Info(msg, keysAndValues...)
}
