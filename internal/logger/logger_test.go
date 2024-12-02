package logger

import (
	"testing"

	"github.com/stretchr/testify/mock"
)

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Error(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

func (m *MockLogger) Debug(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

func (m *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

func TestLoggerInterface(t *testing.T) {
	mockLogger := new(MockLogger)

	t.Run("EmptyKeyValues", func(t *testing.T) {
		mockLogger.On("Error", "error message").Return()
		mockLogger.On("Debug", "debug message").Return()
		mockLogger.On("Info", "info message").Return()

		mockLogger.Error("error message")
		mockLogger.Debug("debug message")
		mockLogger.Info("info message")

		mockLogger.AssertExpectations(t)
	})

	t.Run("NilKeyValues", func(t *testing.T) {
		mockLogger := new(MockLogger)
		mockLogger.On("Error", "error message", nil).Return()
		mockLogger.On("Debug", "debug message", nil).Return()
		mockLogger.On("Info", "info message", nil).Return()

		mockLogger.Error("error message", nil)
		mockLogger.Debug("debug message", nil)
		mockLogger.Info("info message", nil)

		mockLogger.AssertExpectations(t)
	})

	t.Run("MultipleKeyValuePairs", func(t *testing.T) {
		mockLogger := new(MockLogger)
		mockLogger.On("Error", "error message", "key1", "value1", "key2", 42).Return()
		mockLogger.On("Debug", "debug message", "key1", true, "key2", 3.14).Return()
		mockLogger.On("Info", "info message", "key1", []string{"val1", "val2"}, "key2", map[string]int{"a": 1}).Return()

		mockLogger.Error("error message", "key1", "value1", "key2", 42)
		mockLogger.Debug("debug message", "key1", true, "key2", 3.14)
		mockLogger.Info("info message", "key1", []string{"val1", "val2"}, "key2", map[string]int{"a": 1})

		mockLogger.AssertExpectations(t)
	})

	t.Run("EmptyMessage", func(t *testing.T) {
		mockLogger := new(MockLogger)
		mockLogger.On("Error", "", "key1", "value1").Return()
		mockLogger.On("Debug", "", "key1", "value1").Return()
		mockLogger.On("Info", "", "key1", "value1").Return()

		mockLogger.Error("", "key1", "value1")
		mockLogger.Debug("", "key1", "value1")
		mockLogger.Info("", "key1", "value1")

		mockLogger.AssertExpectations(t)
	})
}
