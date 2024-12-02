package ewrap

import (
	"errors"
	"time"
)

// RetryInfo holds information about retry attempts.
type RetryInfo struct {
	// MaxAttempts is the maximum number of retry attempts.
	MaxAttempts int
	// CurrentAttempt is the current retry attempt.
	CurrentAttempt int
	// Delay is the delay between retry attempts.
	Delay time.Duration
	// LastAttempt is the timestamp of the last retry attempt.
	LastAttempt time.Time
	// ShouldRetry is a function that determines if a retry should be attempted.
	ShouldRetry func(error) bool
}

// WithRetry adds retry information to the error.
func WithRetry(maxAttempts int, delay time.Duration) Option {
	return func(err *Error) {
		retryInfo := &RetryInfo{
			MaxAttempts: maxAttempts,
			Delay:       delay,
			LastAttempt: time.Now(),
			ShouldRetry: defaultShouldRetry,
		}

		err.mu.Lock()
		err.metadata["retry_info"] = retryInfo
		err.mu.Unlock()
	}
}

// defaultShouldRetry is the default retry decision function.
func defaultShouldRetry(err error) bool {
	// Don't retry validation errors
	var wrappedErr *Error
	if errors.As(err, &wrappedErr) {
		if ctx, ok := wrappedErr.metadata["error_context"].(*ErrorContext); ok {
			return ctx.Type != ErrorTypeValidation
		}
	}

	return true
}

// CanRetry checks if the error can be retried.
func (e *Error) CanRetry() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	retryInfo, ok := e.metadata["retry_info"].(*RetryInfo)
	if !ok {
		return false
	}

	return retryInfo.CurrentAttempt < retryInfo.MaxAttempts &&
		retryInfo.ShouldRetry(e)
}

// IncrementRetry increments the retry counter.
func (e *Error) IncrementRetry() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if retryInfo, ok := e.metadata["retry_info"].(*RetryInfo); ok {
		retryInfo.CurrentAttempt++
		retryInfo.LastAttempt = time.Now()
	}
}
