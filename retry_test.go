package ewrap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithRetry(t *testing.T) {
	maxAttempts := 3
	delay := time.Second
	err := New("test error", WithRetry(maxAttempts, delay))

	retryInfo, ok := err.metadata["retry_info"].(*RetryInfo)
	assert.True(t, ok)
	assert.Equal(t, maxAttempts, retryInfo.MaxAttempts)
	assert.Equal(t, delay, retryInfo.Delay)
	assert.Equal(t, 0, retryInfo.CurrentAttempt)
	assert.NotZero(t, retryInfo.LastAttempt)
	assert.NotNil(t, retryInfo.ShouldRetry)
}

func TestCanRetry(t *testing.T) {
	t.Run("WithValidRetryInfo", func(t *testing.T) {
		err := New("test error", WithRetry(3, time.Second))
		assert.True(t, err.CanRetry())

		err.IncrementRetry()
		assert.True(t, err.CanRetry())

		err.IncrementRetry()
		assert.True(t, err.CanRetry())

		err.IncrementRetry()
		assert.False(t, err.CanRetry())
	})

	t.Run("WithoutRetryInfo", func(t *testing.T) {
		err := New("test error")
		assert.False(t, err.CanRetry())
	})
}

func TestDefaultShouldRetry(t *testing.T) {
	t.Run("ValidationError", func(t *testing.T) {
		err := New("validation error").
			WithMetadata("error_context", &ErrorContext{Type: ErrorTypeValidation})
		assert.False(t, defaultShouldRetry(err))
	})

	t.Run("OtherError", func(t *testing.T) {
		err := New("other error").
			WithMetadata("error_context", &ErrorContext{Type: ErrorTypeInternal})
		assert.True(t, defaultShouldRetry(err))
	})

	t.Run("NoContext", func(t *testing.T) {
		err := New("no context error")
		assert.True(t, defaultShouldRetry(err))
	})
}

func TestIncrementRetry(t *testing.T) {
	t.Run("WithRetryInfo", func(t *testing.T) {
		err := New("test error", WithRetry(3, time.Second))
		initialTime := err.metadata["retry_info"].(*RetryInfo).LastAttempt

		time.Sleep(time.Millisecond)
		err.IncrementRetry()

		retryInfo := err.metadata["retry_info"].(*RetryInfo)
		assert.Equal(t, 1, retryInfo.CurrentAttempt)
		assert.True(t, retryInfo.LastAttempt.After(initialTime))
	})

	t.Run("WithoutRetryInfo", func(t *testing.T) {
		err := New("test error")
		err.IncrementRetry() // Should not panic
	})
}
