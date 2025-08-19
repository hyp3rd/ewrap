package ewrap

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// MockLogger implements the logger.Logger interface for testing
type MockLogger struct {
	mu     sync.Mutex
	logs   []LogEntry
	called map[string]int
}

type LogEntry struct {
	Level string
	Msg   string
	Args  []any
}

func (m *MockLogger) Info(msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, LogEntry{Level: "info", Msg: msg, Args: args})
}

func (m *MockLogger) Debug(msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, LogEntry{Level: "debug", Msg: msg, Args: args})
	m.called["debug"]++
}

func (m *MockLogger) Error(msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, LogEntry{Level: "error", Msg: msg, Args: args})
	m.called["error"]++
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs:   make([]LogEntry, 0),
		called: make(map[string]int),
	}
}

func (m *MockLogger) GetLogs() []LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logs
}

func (m *MockLogger) GetCallCount(level string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.called[level]
}

func TestNew(t *testing.T) {
	t.Run("creates error with message", func(t *testing.T) {
		err := New("test error")
		if err.Error() != "test error" {
			t.Errorf("expected 'test error', got '%s'", err.Error())
		}

		if len(err.stack) == 0 {
			t.Error("expected stack trace to be captured")
		}

		if err.metadata == nil {
			t.Error("expected metadata to be initialized")
		}
	})

	t.Run("applies options", func(t *testing.T) {
		mockLogger := NewMockLogger()
		err := New("test error", WithLogger(mockLogger))
		if err.logger != mockLogger {
			t.Error("expected logger to be set")
		}
		if mockLogger.GetCallCount("debug") != 1 {
			t.Error("expected logger debug to be called once")
		}
	})
}

func TestNewf(t *testing.T) {
	err := Newf("test error %d", 42)
	expected := "test error 42"
	if err.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, err.Error())
	}
}

func TestWrap(t *testing.T) {
	t.Run("wraps nil error returns nil", func(t *testing.T) {
		result := Wrap(nil, "test")
		if result != nil {
			t.Error("expected nil when wrapping nil error")
		}
	})

	t.Run("wraps standard error", func(t *testing.T) {
		originalErr := errors.New("original error")
		wrapped := Wrap(originalErr, "wrapped")

		if wrapped.msg != "wrapped" {
			t.Errorf("expected message 'wrapped', got '%s'", wrapped.msg)
		}
		if wrapped.cause != originalErr {
			t.Error("expected cause to be set to original error")
		}
		expected := "wrapped: original error"
		if wrapped.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, wrapped.Error())
		}
	})

	t.Run("wraps custom Error preserving stack and metadata", func(t *testing.T) {
		original := New("original").WithMetadata("key", "value")
		wrapped := Wrap(original, "wrapped")

		if len(wrapped.stack) == 0 {
			t.Error("expected stack trace to be preserved")
		}
		if val, ok := wrapped.GetMetadata("key"); !ok || val != "value" {
			t.Error("expected metadata to be preserved")
		}
	})
}

func TestWrapf(t *testing.T) {
	t.Run("wraps nil error returns nil", func(t *testing.T) {
		result := Wrapf(nil, "test %d", 42)
		if result != nil {
			t.Error("expected nil when wrapping nil error")
		}
	})

	t.Run("wraps with formatted message", func(t *testing.T) {
		originalErr := errors.New("original")
		wrapped := Wrapf(originalErr, "wrapped %d", 42)
		expected := "wrapped 42: original"
		if wrapped.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, wrapped.Error())
		}
	})
}

func TestError_Error(t *testing.T) {
	t.Run("returns message when no cause", func(t *testing.T) {
		err := New("test message")
		if err.Error() != "test message" {
			t.Errorf("expected 'test message', got '%s'", err.Error())
		}
	})

	t.Run("returns message with cause", func(t *testing.T) {
		cause := errors.New("cause error")
		err := Wrap(cause, "wrapped")
		expected := "wrapped: cause error"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})
}

func TestError_Cause(t *testing.T) {
	t.Run("returns nil for new error", func(t *testing.T) {
		err := New("test")
		if err.Cause() != nil {
			t.Error("expected nil cause for new error")
		}
	})

	t.Run("returns cause for wrapped error", func(t *testing.T) {
		cause := errors.New("original")
		wrapped := Wrap(cause, "wrapped")
		if wrapped.Cause() != cause {
			t.Error("expected cause to match original error")
		}
	})
}

func TestError_WithMetadata(t *testing.T) {
	err := New("test")
	result := err.WithMetadata("key", "value")

	if result != err {
		t.Error("expected WithMetadata to return same error instance")
	}

	val, ok := err.GetMetadata("key")
	if !ok {
		t.Error("expected metadata to be set")
	}
	if val != "value" {
		t.Errorf("expected 'value', got '%v'", val)
	}
}

func TestError_WithContext(t *testing.T) {
	err := New("test")
	ctx := &ErrorContext{}
	result := err.WithContext(ctx)

	if result != err {
		t.Error("expected WithContext to return same error instance")
	}

	retrievedCtx := err.GetErrorContext()
	if retrievedCtx != ctx {
		t.Error("expected context to be set")
	}
}

func TestError_GetMetadata(t *testing.T) {
	err := New("test").WithMetadata("key", "value")

	t.Run("existing key", func(t *testing.T) {
		val, ok := err.GetMetadata("key")
		if !ok {
			t.Error("expected key to exist")
		}
		if val != "value" {
			t.Errorf("expected 'value', got '%v'", val)
		}
	})

	t.Run("non-existing key", func(t *testing.T) {
		val, ok := err.GetMetadata("nonexistent")
		if ok {
			t.Error("expected key to not exist")
		}
		if val != nil {
			t.Errorf("expected nil value, got '%v'", val)
		}
	})
}

func TestError_Stack(t *testing.T) {
	err := New("test")
	stack := err.Stack()

	if stack == "" {
		t.Error("expected non-empty stack trace")
	}

	// Should not contain runtime frames or error package frames
	if strings.Contains(stack, "runtime/") {
		t.Error("stack should not contain runtime frames")
	}
	if strings.Contains(stack, "ewrap/errors.go") {
		t.Error("stack should not contain error package frames")
	}
}

func TestError_Log(t *testing.T) {
	t.Run("does nothing when no logger", func(t *testing.T) {
		err := New("test")
		err.Log() // Should not panic
	})

	t.Run("logs with logger", func(t *testing.T) {
		mockLogger := NewMockLogger()
		err := New("test", WithLogger(mockLogger)).WithMetadata("key", "value")
		err.Log()

		if mockLogger.GetCallCount("error") != 1 {
			t.Error("expected error log to be called once")
		}

		logs := mockLogger.GetLogs()
		if len(logs) < 2 { // At least creation debug log and error log
			t.Error("expected at least 2 log entries")
		}
	})

	t.Run("logs with cause", func(t *testing.T) {
		mockLogger := NewMockLogger()
		cause := errors.New("original")
		err := Wrap(cause, "wrapped", WithLogger(mockLogger))
		err.Log()

		if mockLogger.GetCallCount("error") != 1 {
			t.Error("expected error log to be called once")
		}
	})
}

func TestCaptureStack(t *testing.T) {
	stack := CaptureStack()
	if len(stack) == 0 {
		t.Error("expected non-empty stack trace")
	}
}

func TestError_Is(t *testing.T) {
	t.Run("returns false for nil target", func(t *testing.T) {
		err := New("test")
		if errors.Is(err, nil) {
			t.Error("expected false for nil target")
		}
	})

	t.Run("matches sentinel error", func(t *testing.T) {
		sentinel := errors.New("sentinel")
		wrapped := Wrap(sentinel, "wrapped")
		if !errors.Is(wrapped, sentinel) {
			t.Error("expected true for sentinel error in chain")
		}
	})

	t.Run("matches ewrap sentinel", func(t *testing.T) {
		sentinel := New("sentinel")
		wrapped := Wrap(sentinel, "wrapped")
		if !errors.Is(wrapped, sentinel) {
			t.Error("expected true for ewrap sentinel in chain")
		}
	})

	t.Run("non-matching error", func(t *testing.T) {
		err := New("test error")
		target := errors.New("other")
		if errors.Is(err, target) {
			t.Error("expected false for non-matching error")
		}
	})
}

func TestError_Unwrap(t *testing.T) {
	t.Run("returns nil for new error", func(t *testing.T) {
		err := New("test")
		if err.Unwrap() != nil {
			t.Error("expected nil for new error")
		}
	})

	t.Run("returns cause for wrapped error", func(t *testing.T) {
		cause := errors.New("original")
		wrapped := Wrap(cause, "wrapped")
		if wrapped.Unwrap() != cause {
			t.Error("expected unwrap to return cause")
		}
	})
}

func TestWithLogger(t *testing.T) {
	mockLogger := NewMockLogger()
	option := WithLogger(mockLogger)
	err := &Error{
		msg:      "test",
		metadata: make(map[string]any),
		stack:    CaptureStack(),
	}

	option(err)

	if err.logger != mockLogger {
		t.Error("expected logger to be set")
	}
	if mockLogger.GetCallCount("debug") != 1 {
		t.Error("expected debug log to be called once")
	}
}

func TestConcurrentAccess(t *testing.T) {
	err := New("test")

	// Test concurrent metadata access
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			err.WithMetadata(fmt.Sprintf("key%d", i), i)
		}(i)
		go func(i int) {
			defer wg.Done()
			err.GetMetadata(fmt.Sprintf("key%d", i))
		}(i)
	}
	wg.Wait()

	// Should not panic or race
}
