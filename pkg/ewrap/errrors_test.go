package ewrap

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	msg := "test error"

	err := New(msg)

	assert.NotNil(t, err)
	assert.Equal(t, msg, err.Error())
	assert.NotEmpty(t, err.Stack())
}

func TestWrap(t *testing.T) {
	cause := errors.New("original error")

	msg := "wrapped error"

	err := Wrap(cause, msg)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), msg)
	assert.Contains(t, err.Error(), cause.Error())
	assert.Equal(t, cause, err.Cause())
}

func TestWrapNil(t *testing.T) {
	err := Wrap(nil, "wrapped error")
	assert.Nil(t, err)
}

func TestWrapf(t *testing.T) {
	cause := errors.New("original error")

	err := Wrapf(cause, "wrapped error: %d", 42)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "wrapped error: 42")
	assert.Contains(t, err.Error(), cause.Error())
}

func TestErrorChain(t *testing.T) {
	cause := errors.New("root cause")

	err1 := Wrap(cause, "first wrap")

	err2 := Wrap(err1, "second wrap")

	assert.NotNil(t, err2)
	assert.Contains(t, err2.Error(), "second wrap")
	assert.Contains(t, err2.Error(), "first wrap")
	assert.Contains(t, err2.Error(), cause.Error())
}

func TestMetadata(t *testing.T) {
	err := New("test error").
		WithMetadata("key1", "value1").
		WithMetadata("key2", 42)

	val1, ok := err.GetMetadata("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val1)

	val2, ok := err.GetMetadata("key2")
	assert.True(t, ok)
	assert.Equal(t, 42, val2)

	_, ok = err.GetMetadata("nonexistent")
	assert.False(t, ok)
}

func TestIs(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := Wrap(originalErr, "wrapped error")

	assert.True(t, wrappedErr.Is(originalErr))
	assert.False(t, wrappedErr.Is(errors.New("different error")))
}

func ExampleError_WithMetadata() {
	err := New("database connection failed").
		WithMetadata("retry_count", 3).
		WithMetadata("last_error", "connection timeout")

	fmt.Printf("Error: %v\n", err)

	if retryCount, ok := err.GetMetadata("retry_count"); ok {
		// Output:
		// Error: database connection failed
		// Retry count: 3
		fmt.Printf("Retry count: %v\n", retryCount)
	}
}

// testLogger implements the Logger interface for testing
type testLogger struct {
	logs []logEntry
	mu   sync.RWMutex
}

type logEntry struct {
	level   string
	msg     string
	keyVals []interface{}
}

func newTestLogger() *testLogger {
	return &testLogger{
		logs: make([]logEntry, 0),
	}
}

func (l *testLogger) Error(msg string, keysAndValues ...interface{}) {
	l.log("error", msg, keysAndValues...)
}

func (l *testLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.log("debug", msg, keysAndValues...)
}

func (l *testLogger) Info(msg string, keysAndValues ...interface{}) {
	l.log("info", msg, keysAndValues...)
}

func (l *testLogger) log(level, msg string, keysAndValues ...interface{}) {
	l.mu.Lock()
	l.logs = append(l.logs, logEntry{
		level:   level,
		msg:     msg,
		keyVals: keysAndValues,
	})
	l.mu.Unlock()
}

func (l *testLogger) getLogs() []logEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	logs := make([]logEntry, len(l.logs))
	copy(logs, l.logs)

	return logs
}

func TestNewWithLogger(t *testing.T) {
	logger := newTestLogger()

	msg := "test error"

	err := New(msg, WithLogger(logger))

	assert.NotNil(t, err)
	assert.Equal(t, msg, err.Error())

	logs := logger.getLogs()

	assert.Len(t, logs, 1)
	assert.Equal(t, "debug", logs[0].level)
	assert.Equal(t, "error created", logs[0].msg)
}

func TestWrapWithLogger(t *testing.T) {
	logger := newTestLogger()

	cause := errors.New("original error")

	msg := "wrapped error"

	err := Wrap(cause, msg, WithLogger(logger))

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), msg)
	assert.Contains(t, err.Error(), cause.Error())

	logs := logger.getLogs()

	assert.Len(t, logs, 1)
	assert.Equal(t, "debug", logs[0].level)
}

func TestErrorLog(t *testing.T) {
	logger := newTestLogger()

	err := New("test error", WithLogger(logger)).
		WithMetadata("key1", "value1").
		WithMetadata("key2", 42)

	err.Log()

	logs := logger.getLogs()
	assert.GreaterOrEqual(t, len(logs), 2) // At least creation and log entry

	// Find the error log
	var errorLog *logEntry
	for _, log := range logs {
		if log.level == "error" && log.msg == "error occurred" {
			errorLog = &log

			break
		}
	}

	assert.NotNil(t, errorLog)
	assert.Contains(t, errorLog.keyVals, "key1")
	assert.Contains(t, errorLog.keyVals, "value1")
}

func ExampleError_withLogger() {
	logger := newTestLogger()
	err := New("database connection failed", WithLogger(logger)).
		WithMetadata("retry_count", 3).
		WithMetadata("last_error", "connection timeout")

	err.Log() // This will use the configured logger

	// In real usage, you might use your preferred logging framework
	// Output:
	// Error: database connection failed
	fmt.Printf("Error: %v\n", err)
}
