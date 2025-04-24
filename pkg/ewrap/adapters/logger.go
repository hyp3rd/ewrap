// Package adapters provides logging adapters for popular logging frameworks
package adapters

import (
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

const keyValuePairSize = 2

// ZapAdapter adapts Zap logger to the ewrap.Logger interface.
type ZapAdapter struct {
	logger *zap.Logger
}

// NewZapAdapter creates a new Zap logger adapter.
func NewZapAdapter(logger *zap.Logger) *ZapAdapter {
	return &ZapAdapter{logger: logger}
}

// Error logs an error message with optional key-value pairs.
func (z *ZapAdapter) Error(msg string, keysAndValues ...any) {
	fields := convertToZapFields(keysAndValues...)
	z.logger.Error(msg, fields...)
}

// Debug logs a debug message with optional key-value pairs.
func (z *ZapAdapter) Debug(msg string, keysAndValues ...any) {
	fields := convertToZapFields(keysAndValues...)
	z.logger.Debug(msg, fields...)
}

// Info logs an info message with optional key-value pairs.
func (z *ZapAdapter) Info(msg string, keysAndValues ...any) {
	fields := convertToZapFields(keysAndValues...)
	z.logger.Info(msg, fields...)
}

// LogrusAdapter adapts Logrus logger to the ewrap.Logger interface.
type LogrusAdapter struct {
	logger *logrus.Logger
}

// NewLogrusAdapter creates a new Logrus logger adapter.
func NewLogrusAdapter(logger *logrus.Logger) *LogrusAdapter {
	return &LogrusAdapter{logger: logger}
}

// Error logs an error message with optional key-value pairs.
func (l *LogrusAdapter) Error(msg string, keysAndValues ...any) {
	fields := convertToLogrusFields(keysAndValues...)
	l.logger.WithFields(fields).Error(msg)
}

// Debug logs a debug message with optional key-value pairs.
func (l *LogrusAdapter) Debug(msg string, keysAndValues ...any) {
	fields := convertToLogrusFields(keysAndValues...)
	l.logger.WithFields(fields).Debug(msg)
}

// Info logs an info message with optional key-value pairs.
func (l *LogrusAdapter) Info(msg string, keysAndValues ...any) {
	fields := convertToLogrusFields(keysAndValues...)
	l.logger.WithFields(fields).Info(msg)
}

// ZerologAdapter adapts Zerolog logger to the ewrap.Logger interface.
type ZerologAdapter struct {
	logger zerolog.Logger
}

// NewZerologAdapter creates a new Zerolog logger adapter.
func NewZerologAdapter(logger zerolog.Logger) *ZerologAdapter {
	return &ZerologAdapter{logger: logger}
}

// Error logs an error message with optional key-value pairs.
func (z *ZerologAdapter) Error(msg string, keysAndValues ...any) {
	event := z.logger.Error()
	addZerologFields(event, keysAndValues...)
	event.Msg(msg)
}

// Debug logs a debug message with optional key-value pairs.
func (z *ZerologAdapter) Debug(msg string, keysAndValues ...any) {
	event := z.logger.Debug()
	addZerologFields(event, keysAndValues...)
	event.Msg(msg)
}

// Info logs an info message with optional key-value pairs.
func (z *ZerologAdapter) Info(msg string, keysAndValues ...any) {
	event := z.logger.Info()
	addZerologFields(event, keysAndValues...)
	event.Msg(msg)
}

// Helper functions to convert key-value pairs to logger-specific formats.
func convertToZapFields(keysAndValues ...any) []zap.Field {
	fields := make([]zap.Field, 0, len(keysAndValues)/keyValuePairSize)

	for i := 0; i < len(keysAndValues); i += keyValuePairSize {
		if i+1 < len(keysAndValues) {
			key, ok := keysAndValues[i].(string)
			if !ok {
				continue
			}

			fields = append(fields, zap.Any(key, keysAndValues[i+1]))
		}
	}

	return fields
}

// convertToLogrusFields converts key-value pairs to Logrus fields.
// It iterates over the provided key-value pairs and adds them to a Logrus fields map.
// If the number of arguments is odd, the last argument is ignored.
func convertToLogrusFields(keysAndValues ...any) logrus.Fields {
	fields := make(logrus.Fields)

	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key, ok := keysAndValues[i].(string)
			if !ok {
				continue
			}

			fields[key] = keysAndValues[i+1]
		}
	}

	return fields
}

// addZerologFields adds key-value pairs to a Zerolog event.
// It iterates over the provided key-value pairs and adds them to the event.
// If the number of arguments is odd, the last argument is ignored.
func addZerologFields(event *zerolog.Event, keysAndValues ...any) {
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key, ok := keysAndValues[i].(string)
			if !ok {
				continue
			}

			event.Interface(key, keysAndValues[i+1])
		}
	}
}
