package ewrap

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

const (
	concurrentMetadataIters = 100
	formatNumber            = 42
	wrapfFormat             = "wrapped %d"
	expectedDebugCalls      = 1
	metadataIntValue        = 5
)

// MockLogger implements the Logger interface for testing.
type MockLogger struct {
	mu     sync.Mutex
	logs   []LogEntry
	called map[string]int
}

// LogEntry captures one logged message and its arguments.
type LogEntry struct {
	Level string
	Msg   string
	Args  []any
}

// NewMockLogger constructs a fresh MockLogger.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs:   make([]LogEntry, 0),
		called: make(map[string]int),
	}
}

func (m *MockLogger) Info(msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logs = append(m.logs, LogEntry{Level: severityInfoStr, Msg: msg, Args: args})
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

	m.logs = append(m.logs, LogEntry{Level: severityErrorStr, Msg: msg, Args: args})
	m.called[severityErrorStr]++
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
	t.Parallel()

	t.Run("creates error with message", testNewCreates)
	t.Run("applies options", testNewAppliesOptions)
}

func testNewCreates(t *testing.T) {
	t.Parallel()

	err := New(msgTestError)
	if err.Error() != msgTestError {
		t.Errorf("expected %q, got %q", msgTestError, err.Error())
	}

	if len(err.stack) == 0 {
		t.Error("expected stack trace to be captured")
	}

	if err.metadata != nil {
		t.Errorf("expected nil metadata before first write, got %v", err.metadata)
	}

	_ = err.WithMetadata("k", "v")

	if err.metadata == nil {
		t.Error("expected metadata to be initialized after WithMetadata")
	}
}

func testNewAppliesOptions(t *testing.T) {
	t.Parallel()

	mockLogger := NewMockLogger()

	err := New(msgTestError, WithLogger(mockLogger))
	if err.logger != mockLogger {
		t.Error("expected logger to be set")
	}

	if mockLogger.GetCallCount("debug") != expectedDebugCalls {
		t.Error("expected logger debug to be called once")
	}
}

func TestNewf(t *testing.T) {
	t.Parallel()

	err := Newf("test error %d", formatNumber)

	expected := fmt.Sprintf("test error %d", formatNumber)
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestWrap(t *testing.T) {
	t.Parallel()

	t.Run("wraps nil error returns nil", testWrapNil)
	t.Run("wraps standard error", testWrapStandard)
	t.Run("wraps custom Error preserving stack and metadata", testWrapCustom)
}

func testWrapNil(t *testing.T) {
	t.Parallel()

	result := Wrap(nil, msgTest)
	if result != nil {
		t.Error("expected nil when wrapping nil error")
	}
}

func testWrapStandard(t *testing.T) {
	t.Parallel()

	wrapped := Wrap(errOriginalLong, msgWrapped)

	if wrapped.msg != msgWrapped {
		t.Errorf("expected message %q, got %q", msgWrapped, wrapped.msg)
	}

	if !errors.Is(wrapped.cause, errOriginalLong) {
		t.Error("expected cause to be set to original error")
	}

	expected := msgWrapped + ": " + msgOriginalErr
	if wrapped.Error() != expected {
		t.Errorf("expected %q, got %q", expected, wrapped.Error())
	}
}

func testWrapCustom(t *testing.T) {
	t.Parallel()

	original := New(msgOriginal).WithMetadata(msgKey, msgValue)
	wrapped := Wrap(original, msgWrapped)

	if len(wrapped.stack) == 0 {
		t.Error("expected stack trace to be preserved")
	}

	if val, ok := wrapped.GetMetadata(msgKey); !ok || val != msgValue {
		t.Error("expected metadata to be preserved")
	}
}

func TestWrapf(t *testing.T) {
	t.Parallel()

	t.Run("wraps nil error returns nil", func(t *testing.T) {
		t.Parallel()

		result := Wrapf(nil, "test %d", formatNumber)
		if result != nil {
			t.Error("expected nil when wrapping nil error")
		}
	})

	t.Run("wraps with formatted message", func(t *testing.T) {
		t.Parallel()

		wrapped := Wrapf(errOriginal, wrapfFormat, formatNumber)

		expected := fmt.Sprintf(wrapfFormat+": "+msgOriginal, formatNumber)
		if wrapped.Error() != expected {
			t.Errorf("expected %q, got %q", expected, wrapped.Error())
		}
	})
}

func TestError_Error(t *testing.T) {
	t.Parallel()

	t.Run("returns message when no cause", func(t *testing.T) {
		t.Parallel()

		err := New("test message")
		if err.Error() != "test message" {
			t.Errorf("expected 'test message', got %q", err.Error())
		}
	})

	t.Run("returns message with cause", func(t *testing.T) {
		t.Parallel()

		err := Wrap(errCause, msgWrapped)

		expected := msgWrapped + ": " + msgCauseError
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})
}

func TestError_Cause(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for new error", func(t *testing.T) {
		t.Parallel()

		err := New(msgTest)
		if err.Cause() != nil {
			t.Error("expected nil cause for new error")
		}
	})

	t.Run("returns cause for wrapped error", func(t *testing.T) {
		t.Parallel()

		wrapped := Wrap(errOriginal, msgWrapped)
		if !errors.Is(wrapped.Cause(), errOriginal) {
			t.Error("expected cause to match original error")
		}
	})
}

func TestError_WithMetadata(t *testing.T) {
	t.Parallel()

	err := New(msgTest)
	result := err.WithMetadata(msgKey, msgValue)

	if result != err {
		t.Error("expected WithMetadata to return same error instance")
	}

	val, ok := err.GetMetadata(msgKey)
	if !ok {
		t.Error("expected metadata to be set")
	}

	if val != msgValue {
		t.Errorf("expected %q, got %v", msgValue, val)
	}
}

func TestError_WithContext(t *testing.T) {
	t.Parallel()

	err := New(msgTest)
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
	t.Parallel()

	err := New(msgTest).WithMetadata(msgKey, msgValue)

	t.Run("existing key", func(t *testing.T) {
		t.Parallel()

		val, ok := err.GetMetadata(msgKey)
		if !ok {
			t.Error("expected key to exist")
		}

		if val != msgValue {
			t.Errorf("expected %q, got %v", msgValue, val)
		}
	})

	t.Run("non-existing key", func(t *testing.T) {
		t.Parallel()

		val, ok := err.GetMetadata("nonexistent")
		if ok {
			t.Error("expected key to not exist")
		}

		if val != nil {
			t.Errorf("expected nil value, got %v", val)
		}
	})
}

func TestError_GetMetadataValue(t *testing.T) {
	t.Parallel()

	err := New(msgTest).WithMetadata("count", metadataIntValue)

	val, ok := GetMetadataValue[int](err, "count")
	if !ok || val != metadataIntValue {
		t.Errorf("expected typed metadata %d, got %v (ok=%v)", metadataIntValue, val, ok)
	}

	_, ok = GetMetadataValue[int](err, "missing")
	if ok {
		t.Error("expected missing key to return ok=false")
	}
}

func TestWithRecoverySuggestion(t *testing.T) {
	t.Parallel()

	mockLogger := NewMockLogger()
	rs := &RecoverySuggestion{Message: "restart"}
	err := New(msgTest, WithLogger(mockLogger), WithRecoverySuggestion(rs))

	if countLogsAtLevel(mockLogger.GetLogs(), severityInfoStr) != 1 {
		t.Error("expected info log when adding recovery suggestion")
	}

	retrieved := err.Recovery()
	if retrieved == nil || retrieved.Message != rs.Message {
		t.Error("expected recovery suggestion to be set")
	}

	err.Log()

	if !findRecoveryMessageInLogs(mockLogger.GetLogs(), rs.Message) {
		t.Error("expected recovery_message in error log")
	}
}

// countLogsAtLevel counts how many entries match the given level.
func countLogsAtLevel(logs []LogEntry, level string) int {
	count := 0

	for _, l := range logs {
		if l.Level == level {
			count++
		}
	}

	return count
}

// findRecoveryMessageInLogs scans error-level entries for a "recovery_message"
// key whose value matches want.
func findRecoveryMessageInLogs(logs []LogEntry, want string) bool {
	for _, entry := range logs {
		if entry.Level != severityErrorStr {
			continue
		}

		for i := 0; i < len(entry.Args); i += 2 {
			if entry.Args[i] == "recovery_message" && entry.Args[i+1] == want {
				return true
			}
		}
	}

	return false
}

func TestError_Stack(t *testing.T) {
	t.Parallel()

	err := New(msgTest)
	stack := err.Stack()

	if stack == "" {
		t.Error("expected non-empty stack trace")
	}

	if strings.Contains(stack, "runtime/") {
		t.Error("stack should not contain runtime frames")
	}

	if strings.Contains(stack, "ewrap/errors.go") {
		t.Error("stack should not contain error package frames")
	}
}

func TestError_Log(t *testing.T) {
	t.Parallel()

	t.Run("does nothing when no logger", func(t *testing.T) {
		t.Parallel()

		err := New(msgTest)
		err.Log() // Should not panic
	})

	t.Run("logs with logger", func(t *testing.T) {
		t.Parallel()

		mockLogger := NewMockLogger()
		err := New(msgTest, WithLogger(mockLogger)).WithMetadata(msgKey, msgValue)
		err.Log()

		if mockLogger.GetCallCount(severityErrorStr) != 1 {
			t.Error("expected error log to be called once")
		}

		logs := mockLogger.GetLogs()
		if len(logs) < 2 {
			t.Error("expected at least 2 log entries")
		}
	})

	t.Run("logs with cause", func(t *testing.T) {
		t.Parallel()

		mockLogger := NewMockLogger()
		err := Wrap(errOriginal, msgWrapped, WithLogger(mockLogger))
		err.Log()

		if mockLogger.GetCallCount(severityErrorStr) != 1 {
			t.Error("expected error log to be called once")
		}
	})
}

func TestCaptureStack(t *testing.T) {
	t.Parallel()

	stack := CaptureStack()
	if len(stack) == 0 {
		t.Error("expected non-empty stack trace")
	}
}

func TestError_Is(t *testing.T) {
	t.Parallel()

	t.Run("returns false for nil target", func(t *testing.T) {
		t.Parallel()

		err := New(msgTest)
		if errors.Is(err, nil) {
			t.Error("expected false for nil target")
		}
	})

	t.Run("matches sentinel error", func(t *testing.T) {
		t.Parallel()

		wrapped := Wrap(errSentinel, msgWrapped)
		if !errors.Is(wrapped, errSentinel) {
			t.Error("expected true for sentinel error in chain")
		}
	})

	t.Run("matches ewrap sentinel", func(t *testing.T) {
		t.Parallel()

		sentinel := New(msgSentinel)

		wrapped := Wrap(sentinel, msgWrapped)
		if !errors.Is(wrapped, sentinel) {
			t.Error("expected true for ewrap sentinel in chain")
		}
	})

	t.Run("prevents infinite recursion with self-reference", func(t *testing.T) {
		t.Parallel()

		err1 := New("error1")
		err2 := New("error2")

		if errors.Is(err1, err2) {
			t.Error("expected false for different errors")
		}
	})

	t.Run("non-matching error", func(t *testing.T) {
		t.Parallel()

		err := New(msgTestError)

		if errors.Is(err, errOther) {
			t.Error("expected false for non-matching error")
		}
	})
}

func TestError_Unwrap(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for new error", func(t *testing.T) {
		t.Parallel()

		err := New(msgTest)
		if err.Unwrap() != nil {
			t.Error("expected nil for new error")
		}
	})

	t.Run("returns cause for wrapped error", func(t *testing.T) {
		t.Parallel()

		wrapped := Wrap(errOriginal, msgWrapped)
		if !errors.Is(wrapped.Unwrap(), errOriginal) {
			t.Error("expected unwrap to return cause")
		}
	})
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	mockLogger := NewMockLogger()
	option := WithLogger(mockLogger)
	err := &Error{
		msg:      msgTest,
		metadata: make(map[string]any),
		stack:    CaptureStack(),
	}

	option(err)

	if err.logger != mockLogger {
		t.Error("expected logger to be set")
	}

	if mockLogger.GetCallCount("debug") != expectedDebugCalls {
		t.Error("expected debug log to be called once")
	}
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	err := New(msgTest)

	var wg sync.WaitGroup

	for i := range concurrentMetadataIters {
		wg.Go(func() {
			_ = err.WithMetadata(fmt.Sprintf("key%d", i), i)
		})

		wg.Go(func() {
			_, _ = err.GetMetadata(fmt.Sprintf("key%d", i))
		})
	}

	wg.Wait()
}
