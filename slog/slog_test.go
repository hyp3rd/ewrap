package slog

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestAdapter(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	adapter := New(slog.New(handler))

	cases := []struct {
		level string
		emit  func()
	}{
		{level: "ERROR", emit: func() { adapter.Error("error message", "key1", "value1") }},
		{level: "DEBUG", emit: func() { adapter.Debug("debug message", "key2", "value2") }},
		{level: "INFO", emit: func() { adapter.Info("info message", "key3", "value3") }},
	}

	for _, tc := range cases {
		buf.Reset()
		tc.emit()

		out := buf.String()
		if !strings.Contains(out, tc.level) {
			t.Errorf("expected level %q in output, got %q", tc.level, out)
		}
	}
}
