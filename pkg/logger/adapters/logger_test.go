package adapters

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestZapAdapter(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	adapter := NewZapAdapter(logger)

	t.Run("LogLevels", func(t *testing.T) {
		adapter.Error("error message", "key1", "value1")
		adapter.Info("info message", "key2", "value2")
		adapter.Debug("debug message", "key3", "value3")

		entries := recorded.All()
		assert.Len(t, entries, 2)
		assert.Equal(t, "error message", entries[0].Message)
		assert.Equal(t, "info message", entries[1].Message)
	})

	t.Run("InvalidKeyValuePairs", func(t *testing.T) {
		recorded.TakeAll()
		adapter.Info("test", 123, "value", "extra")

		entries := recorded.All()
		assert.Len(t, entries, 1)
		assert.Empty(t, entries[0].ContextMap())
	})
}

func TestLogrusAdapter(t *testing.T) {
	var buf bytes.Buffer

	logger := logrus.New()
	logger.Out = &buf
	adapter := NewLogrusAdapter(logger)

	t.Run("LogLevels", func(t *testing.T) {
		buf.Reset()
		adapter.Error("error message", "key1", "value1")
		assert.Contains(t, buf.String(), "error message")
		assert.Contains(t, buf.String(), "key1")
		assert.Contains(t, buf.String(), "value1")

		buf.Reset()
		adapter.Info("info message", "key2", "value2")
		assert.Contains(t, buf.String(), "info message")
		assert.Contains(t, buf.String(), "key2")
		assert.Contains(t, buf.String(), "value2")
	})

	t.Run("MalformedKeyValuePairs", func(t *testing.T) {
		buf.Reset()
		adapter.Info("test", "single_key")
		assert.Contains(t, buf.String(), "test")
		assert.NotContains(t, buf.String(), "single_key")
	})
}

func TestZerologAdapter(t *testing.T) {
	var buf bytes.Buffer

	logger := zerolog.New(&buf)
	adapter := NewZerologAdapter(logger)

	t.Run("LogLevels", func(t *testing.T) {
		buf.Reset()
		adapter.Error("error message", "key1", "value1")

		output := buf.String()
		assert.Contains(t, output, "error message")
		assert.Contains(t, output, "key1")
		assert.Contains(t, output, "value1")

		buf.Reset()
		adapter.Info("info message", "key2", 42)

		output = buf.String()
		assert.Contains(t, output, "info message")
		assert.Contains(t, output, "key2")
		assert.Contains(t, output, "42")
	})

	t.Run("NonStringKeys", func(t *testing.T) {
		buf.Reset()
		adapter.Info("test", 123, "value")

		output := buf.String()
		assert.Contains(t, output, "test")
		assert.NotContains(t, output, "123")
		assert.NotContains(t, output, "value")
	})
}
