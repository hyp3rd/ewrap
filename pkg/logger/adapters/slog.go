//go:build go1.21

package adapters

import "log/slog"

// SlogAdapter adapts slog.Logger to the ewrap.Logger interface.
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a new slog logger adapter.
func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
	return &SlogAdapter{logger: logger}
}

// Error logs an error message with optional key-value pairs.
func (s *SlogAdapter) Error(msg string, keysAndValues ...any) {
	s.logger.Error(msg, keysAndValues...)
}

// Debug logs a debug message with optional key-value pairs.
func (s *SlogAdapter) Debug(msg string, keysAndValues ...any) {
	s.logger.Debug(msg, keysAndValues...)
}

// Info logs an info message with optional key-value pairs.
func (s *SlogAdapter) Info(msg string, keysAndValues ...any) {
	s.logger.Info(msg, keysAndValues...)
}
