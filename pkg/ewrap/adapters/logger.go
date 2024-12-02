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

func (z *ZapAdapter) Error(msg string, keysAndValues ...interface{}) {
	fields := convertToZapFields(keysAndValues...)
	z.logger.Error(msg, fields...)
}

func (z *ZapAdapter) Debug(msg string, keysAndValues ...interface{}) {
	fields := convertToZapFields(keysAndValues...)
	z.logger.Debug(msg, fields...)
}

func (z *ZapAdapter) Info(msg string, keysAndValues ...interface{}) {
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

func (l *LogrusAdapter) Error(msg string, keysAndValues ...interface{}) {
	fields := convertToLogrusFields(keysAndValues...)
	l.logger.WithFields(fields).Error(msg)
}

func (l *LogrusAdapter) Debug(msg string, keysAndValues ...interface{}) {
	fields := convertToLogrusFields(keysAndValues...)
	l.logger.WithFields(fields).Debug(msg)
}

func (l *LogrusAdapter) Info(msg string, keysAndValues ...interface{}) {
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

func (z *ZerologAdapter) Error(msg string, keysAndValues ...interface{}) {
	event := z.logger.Error()
	addZerologFields(event, keysAndValues...)
	event.Msg(msg)
}

func (z *ZerologAdapter) Debug(msg string, keysAndValues ...interface{}) {
	event := z.logger.Debug()
	addZerologFields(event, keysAndValues...)
	event.Msg(msg)
}

func (z *ZerologAdapter) Info(msg string, keysAndValues ...interface{}) {
	event := z.logger.Info()
	addZerologFields(event, keysAndValues...)
	event.Msg(msg)
}

// Helper functions to convert key-value pairs to logger-specific formats.
func convertToZapFields(keysAndValues ...interface{}) []zap.Field {
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

func convertToLogrusFields(keysAndValues ...interface{}) logrus.Fields {
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

func addZerologFields(event *zerolog.Event, keysAndValues ...interface{}) {
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
