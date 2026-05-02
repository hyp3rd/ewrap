package ewrap

import (
	"sync"
	"testing"
	"time"
)

const (
	thresholdMaxFailures      = 3
	thresholdConcurrency      = 100
	thresholdTimeoutSeconds   = 5
	thresholdConcurrencyLimit = 100
)

func TestNewCircuitBreaker(t *testing.T) {
	t.Parallel()

	const name = "test-circuit"

	timeout := thresholdTimeoutSeconds * time.Second
	cb := NewCircuitBreaker(name, thresholdMaxFailures, timeout)

	if cb.name != name {
		t.Errorf("Expected name %s, got %s", name, cb.name)
	}

	if cb.maxFailures != thresholdMaxFailures {
		t.Errorf("Expected maxFailures %d, got %d", thresholdMaxFailures, cb.maxFailures)
	}

	if cb.timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, cb.timeout)
	}

	if cb.state != CircuitClosed {
		t.Errorf("Expected initial state %v, got %v", CircuitClosed, cb.state)
	}

	if cb.failureCount != 0 {
		t.Errorf("Expected initial failure count 0, got %d", cb.failureCount)
	}
}

func TestCircuitBreakerRecordFailure(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker(msgTest, 2, thresholdTimeoutSeconds*time.Second)

	cb.RecordFailure()

	if cb.state != CircuitClosed {
		t.Errorf("Expected state %v after first failure, got %v", CircuitClosed, cb.state)
	}

	if cb.failureCount != 1 {
		t.Errorf("Expected failure count 1, got %d", cb.failureCount)
	}

	cb.RecordFailure()

	if cb.state != CircuitOpen {
		t.Errorf("Expected state %v after max failures, got %v", CircuitOpen, cb.state)
	}

	if cb.failureCount != 2 {
		t.Errorf("Expected failure count 2, got %d", cb.failureCount)
	}
}

func TestCircuitBreakerRecordSuccess(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker(msgTest, 1, thresholdTimeoutSeconds*time.Second)

	cb.RecordFailure()

	if cb.state != CircuitOpen {
		t.Error("Expected circuit to be open")
	}

	cb.mu.Lock()
	cb.state = CircuitHalfOpen
	cb.mu.Unlock()

	cb.RecordSuccess()

	if cb.state != CircuitClosed {
		t.Errorf("Expected state %v after success in half-open, got %v", CircuitClosed, cb.state)
	}

	if cb.failureCount != 0 {
		t.Errorf("Expected failure count reset to 0, got %d", cb.failureCount)
	}
}

func TestCircuitBreakerCanExecute(t *testing.T) {
	t.Parallel()

	timeout := thresholdConcurrencyLimit * time.Millisecond
	cb := NewCircuitBreaker(msgTest, 1, timeout)

	if !cb.CanExecute() {
		t.Error("Expected CanExecute to return true for closed circuit")
	}

	cb.RecordFailure()

	if cb.CanExecute() {
		t.Error("Expected CanExecute to return false for open circuit")
	}

	time.Sleep(timeout + 10*time.Millisecond)

	if !cb.CanExecute() {
		t.Error("Expected CanExecute to return true after timeout (half-open)")
	}

	cb.mu.Lock()
	state := cb.state
	cb.mu.Unlock()

	if state != CircuitHalfOpen {
		t.Errorf("Expected state %v after timeout, got %v", CircuitHalfOpen, state)
	}
}

func TestCircuitBreakerOnStateChange(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker(msgTest, 1, thresholdTimeoutSeconds*time.Second)

	type recordedChange struct {
		name string
		from CircuitState
		to   CircuitState
	}

	var (
		stateChanges []recordedChange
		mu           sync.Mutex
	)

	cb.OnStateChange(func(name string, from, to CircuitState) {
		mu.Lock()

		stateChanges = append(stateChanges, recordedChange{name, from, to})

		mu.Unlock()
	})

	cb.RecordFailure()

	mu.Lock()
	defer mu.Unlock()

	if len(stateChanges) != 1 {
		t.Errorf("Expected 1 state change, got %d", len(stateChanges))

		return
	}

	change := stateChanges[0]
	if change.name != msgTest {
		t.Errorf("Expected name %q, got %s", msgTest, change.name)
	}

	if change.from != CircuitClosed {
		t.Errorf("Expected from state %v, got %v", CircuitClosed, change.from)
	}

	if change.to != CircuitOpen {
		t.Errorf("Expected to state %v, got %v", CircuitOpen, change.to)
	}
}

func TestCircuitBreakerTransitionViaPublicAPI(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker(msgTest, 1, thresholdTimeoutSeconds*time.Second)

	cb.RecordFailure()

	if cb.state != CircuitOpen {
		t.Errorf("Expected state %v, got %v", CircuitOpen, cb.state)
	}

	cb.RecordFailure()

	if cb.state != CircuitOpen {
		t.Error("Expected state to remain Open on subsequent failure")
	}
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker(msgTest, thresholdTimeoutSeconds, thresholdConcurrencyLimit*time.Millisecond)

	var wg sync.WaitGroup

	for range thresholdConcurrency {
		wg.Go(func() {
			cb.RecordFailure()
		})
	}

	for range thresholdConcurrency {
		wg.Go(func() {
			cb.CanExecute()
		})
	}

	wg.Wait()

	if cb.state != CircuitOpen {
		t.Errorf("Expected circuit to be open after many failures, got %v", cb.state)
	}
}

func TestCircuitStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state    CircuitState
		expected CircuitState
	}{
		{CircuitClosed, CircuitClosed},
		{CircuitOpen, CircuitOpen},
		{CircuitHalfOpen, CircuitHalfOpen},
	}

	for _, test := range tests {
		if test.state != test.expected {
			t.Errorf("Circuit state mismatch: got %v, expected %v", test.state, test.expected)
		}
	}
}
