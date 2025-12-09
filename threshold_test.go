package ewrap

import (
	"sync"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	name := "test-circuit"
	maxFailures := 3
	timeout := 5 * time.Second

	cb := NewCircuitBreaker(name, maxFailures, timeout)

	if cb.name != name {
		t.Errorf("Expected name %s, got %s", name, cb.name)
	}

	if cb.maxFailures != maxFailures {
		t.Errorf("Expected maxFailures %d, got %d", maxFailures, cb.maxFailures)
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
	cb := NewCircuitBreaker("test", 2, 5*time.Second)

	// First failure - should remain closed
	cb.RecordFailure()

	if cb.state != CircuitClosed {
		t.Errorf("Expected state %v after first failure, got %v", CircuitClosed, cb.state)
	}

	if cb.failureCount != 1 {
		t.Errorf("Expected failure count 1, got %d", cb.failureCount)
	}

	// Second failure - should open circuit
	cb.RecordFailure()

	if cb.state != CircuitOpen {
		t.Errorf("Expected state %v after max failures, got %v", CircuitOpen, cb.state)
	}

	if cb.failureCount != 2 {
		t.Errorf("Expected failure count 2, got %d", cb.failureCount)
	}
}

func TestCircuitBreakerRecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, 5*time.Second)

	// Record failure to open circuit
	cb.RecordFailure()

	if cb.state != CircuitOpen {
		t.Error("Expected circuit to be open")
	}

	// Manually set to half-open
	cb.mu.Lock()
	cb.state = CircuitHalfOpen
	cb.mu.Unlock()

	// Record success - should close circuit
	cb.RecordSuccess()

	if cb.state != CircuitClosed {
		t.Errorf("Expected state %v after success in half-open, got %v", CircuitClosed, cb.state)
	}

	if cb.failureCount != 0 {
		t.Errorf("Expected failure count reset to 0, got %d", cb.failureCount)
	}
}

func TestCircuitBreakerCanExecute(t *testing.T) {
	timeout := 100 * time.Millisecond
	cb := NewCircuitBreaker("test", 1, timeout)

	// Initially closed - should allow execution
	if !cb.CanExecute() {
		t.Error("Expected CanExecute to return true for closed circuit")
	}

	// Record failure to open circuit
	cb.RecordFailure()

	if cb.CanExecute() {
		t.Error("Expected CanExecute to return false for open circuit")
	}

	// Wait for timeout and check transition to half-open
	time.Sleep(timeout + 10*time.Millisecond)

	if !cb.CanExecute() {
		t.Error("Expected CanExecute to return true after timeout (half-open)")
	}

	// Verify state is now half-open
	cb.mu.RLock()
	state := cb.state
	cb.mu.RUnlock()

	if state != CircuitHalfOpen {
		t.Errorf("Expected state %v after timeout, got %v", CircuitHalfOpen, state)
	}
}

func TestCircuitBreakerOnStateChange(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, 5*time.Second)

	var (
		stateChanges []struct {
			name string
			from CircuitState
			to   CircuitState
		}
		mu sync.Mutex
	)

	cb.OnStateChange(func(name string, from, to CircuitState) {
		mu.Lock()

		stateChanges = append(stateChanges, struct {
			name string
			from CircuitState
			to   CircuitState
		}{name, from, to})

		mu.Unlock()
	})

	// Record failure to trigger state change
	cb.RecordFailure()

	// Give goroutine time to execute
	time.Sleep(10 * time.Millisecond)

	mu.Lock()

	if len(stateChanges) != 1 {
		t.Errorf("Expected 1 state change, got %d", len(stateChanges))
	} else {
		change := stateChanges[0]
		if change.name != "test" {
			t.Errorf("Expected name 'test', got %s", change.name)
		}

		if change.from != CircuitClosed {
			t.Errorf("Expected from state %v, got %v", CircuitClosed, change.from)
		}

		if change.to != CircuitOpen {
			t.Errorf("Expected to state %v, got %v", CircuitOpen, change.to)
		}
	}

	mu.Unlock()
}

func TestCircuitBreakerTransitionTo(t *testing.T) {
	cb := NewCircuitBreaker("test", 1, 5*time.Second)

	// Test transition from closed to open
	cb.mu.Lock()
	cb.transitionTo(CircuitOpen)
	cb.mu.Unlock()

	if cb.state != CircuitOpen {
		t.Errorf("Expected state %v, got %v", CircuitOpen, cb.state)
	}

	// Test no transition when same state
	cb.mu.Lock()
	oldState := cb.state
	cb.transitionTo(CircuitOpen)
	cb.mu.Unlock()

	if cb.state != oldState {
		t.Error("Expected no state change when transitioning to same state")
	}
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	cb := NewCircuitBreaker("test", 5, 100*time.Millisecond)

	var wg sync.WaitGroup

	iterations := 100

	// Test concurrent RecordFailure calls
	for range iterations {
		wg.Go(func() {
			cb.RecordFailure()
		})
	}

	// Test concurrent CanExecute calls
	for range iterations {
		wg.Go(func() {
			cb.CanExecute()
		})
	}

	wg.Wait()

	// Verify circuit is in expected state
	if cb.state != CircuitOpen {
		t.Errorf("Expected circuit to be open after many failures, got %v", cb.state)
	}
}

func TestCircuitStates(t *testing.T) {
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
