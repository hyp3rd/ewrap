//go:build go1.21

package adapters

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlogAdapter(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)
	adapter := NewSlogAdapter(logger)

	t.Run("LogLevels", func(t *testing.T) {
		buf.Reset()
		adapter.Error("error message", "key1", "value1")
		output := buf.String()
		assert.Contains(t, output, "error message")
		assert.Contains(t, output, "key1")
		assert.Contains(t, output, "value1")

		buf.Reset()
		adapter.Debug("debug message", "key2", "value2")
		output = buf.String()
		assert.Contains(t, output, "debug message")
		assert.Contains(t, output, "key2")
		assert.Contains(t, output, "value2")
	})
}
