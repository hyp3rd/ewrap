package ewrap

import (
	"testing"
	"time"
)

func TestWithRetry(t *testing.T) {
	maxAttempts := 3
	delay := time.Second
	err := New("test error", WithRetry(maxAttempts, delay))

	retryInfo := err.Retry()
	if retryInfo == nil {
		t.Fatal("expected non-nil retry info")
	}

	if retryInfo.MaxAttempts != maxAttempts {
		t.Errorf("MaxAttempts: got %d, want %d", retryInfo.MaxAttempts, maxAttempts)
	}

	if retryInfo.Delay != delay {
		t.Errorf("Delay: got %v, want %v", retryInfo.Delay, delay)
	}

	if retryInfo.CurrentAttempt != 0 {
		t.Errorf("CurrentAttempt: got %d, want 0", retryInfo.CurrentAttempt)
	}

	if retryInfo.LastAttempt.IsZero() {
		t.Error("LastAttempt: expected non-zero time")
	}

	if retryInfo.ShouldRetry == nil {
		t.Error("ShouldRetry: expected non-nil predicate")
	}
}

func TestCanRetry(t *testing.T) {
	t.Run("WithValidRetryInfo", func(t *testing.T) {
		err := New("test error", WithRetry(3, time.Second))
		if !err.CanRetry() {
			t.Error("expected CanRetry true with attempts remaining")
		}

		err.IncrementRetry()

		if !err.CanRetry() {
			t.Error("expected CanRetry true after first increment")
		}

		err.IncrementRetry()

		if !err.CanRetry() {
			t.Error("expected CanRetry true after second increment")
		}

		err.IncrementRetry()

		if err.CanRetry() {
			t.Error("expected CanRetry false after maxAttempts increments")
		}
	})

	t.Run("WithoutRetryInfo", func(t *testing.T) {
		err := New("test error")
		if err.CanRetry() {
			t.Error("expected CanRetry false without retry info")
		}
	})
}

func TestWithRetryCustomShouldRetry(t *testing.T) {
	shouldRetry := func(error) bool { return false }
	err := New("test error", WithRetry(3, time.Second, WithRetryShould(shouldRetry)))

	if err.CanRetry() {
		t.Error("expected CanRetry false with predicate returning false")
	}
}

func TestDefaultShouldRetry(t *testing.T) {
	t.Run("ValidationError", func(t *testing.T) {
		err := New("validation error").
			WithContext(&ErrorContext{Type: ErrorTypeValidation})
		if defaultShouldRetry(err) {
			t.Error("expected defaultShouldRetry false for validation error")
		}
	})

	t.Run("OtherError", func(t *testing.T) {
		err := New("other error").
			WithContext(&ErrorContext{Type: ErrorTypeInternal})
		if !defaultShouldRetry(err) {
			t.Error("expected defaultShouldRetry true for internal error")
		}
	})

	t.Run("NoContext", func(t *testing.T) {
		err := New("no context error")
		if !defaultShouldRetry(err) {
			t.Error("expected defaultShouldRetry true when no context set")
		}
	})
}

func TestIncrementRetry(t *testing.T) {
	t.Run("WithRetryInfo", func(t *testing.T) {
		err := New("test error", WithRetry(3, time.Second))
		initialTime := err.Retry().LastAttempt

		time.Sleep(time.Millisecond)
		err.IncrementRetry()

		retryInfo := err.Retry()
		if retryInfo.CurrentAttempt != 1 {
			t.Errorf("CurrentAttempt: got %d, want 1", retryInfo.CurrentAttempt)
		}

		if !retryInfo.LastAttempt.After(initialTime) {
			t.Error("LastAttempt: expected to advance after increment")
		}
	})

	t.Run("WithoutRetryInfo", func(t *testing.T) {
		err := New("test error")
		err.IncrementRetry() // Should not panic
	})
}
