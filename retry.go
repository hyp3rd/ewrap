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

// RetryOption configures RetryInfo.
type RetryOption func(*RetryInfo)

// WithRetry adds retry information to the error.
func WithRetry(maxAttempts int, delay time.Duration, opts ...RetryOption) Option {
	return func(err *Error) {
		retryInfo := &RetryInfo{
			MaxAttempts: maxAttempts,
			Delay:       delay,
			LastAttempt: time.Now(),
			ShouldRetry: defaultShouldRetry,
		}

		for _, opt := range opts {
			opt(retryInfo)
		}

		err.retry = retryInfo
	}
}

// WithRetryShould sets a custom ShouldRetry function.
func WithRetryShould(fn func(error) bool) RetryOption {
	return func(ri *RetryInfo) {
		if fn != nil {
			ri.ShouldRetry = fn
		}
	}
}

// defaultShouldRetry is the default retry decision function.
// Validation errors are not retried by default.
func defaultShouldRetry(err error) bool {
	var wrappedErr *Error
	if errors.As(err, &wrappedErr) && wrappedErr.errorContext != nil {
		return wrappedErr.errorContext.Type != ErrorTypeValidation
	}

	return true
}

// CanRetry checks if the error can be retried.
func (e *Error) CanRetry() bool {
	e.mu.RLock()
	retryInfo := e.retry
	e.mu.RUnlock()

	if retryInfo == nil {
		return false
	}

	return retryInfo.CurrentAttempt < retryInfo.MaxAttempts &&
		retryInfo.ShouldRetry(e)
}

// IncrementRetry increments the retry counter.
func (e *Error) IncrementRetry() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.retry == nil {
		return
	}

	e.retry.CurrentAttempt++
	e.retry.LastAttempt = time.Now()
}
