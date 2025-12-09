package ewrap

import (
	"testing"
	"time"
)

type testObserver struct {
	errorCount  int
	transitions []stateChange
}

type stateChange struct {
	name string
	from CircuitState
	to   CircuitState
}

func (t *testObserver) RecordError(string) {
	t.errorCount++
}

func (t *testObserver) RecordCircuitStateTransition(name string, from, to CircuitState) {
	t.transitions = append(t.transitions, stateChange{name: name, from: from, to: to})
}

func TestErrorLogRecordsObserver(t *testing.T) {
	obs := &testObserver{}

	err := New("boom", WithObserver(obs))
	err.Log()

	if obs.errorCount != 1 {
		t.Fatalf("expected 1 error recorded, got %d", obs.errorCount)
	}
}

func TestCircuitBreakerObserver(t *testing.T) {
	obs := &testObserver{}

	timeout := 10 * time.Millisecond
	cb := NewCircuitBreakerWithObserver("test", 1, timeout, obs)

	cb.RecordFailure()
	time.Sleep(timeout + time.Millisecond)

	if !cb.CanExecute() {
		t.Fatalf("expected circuit breaker to allow execution after timeout")
	}

	cb.RecordSuccess()

	expected := []stateChange{
		{name: "test", from: CircuitClosed, to: CircuitOpen},
		{name: "test", from: CircuitOpen, to: CircuitHalfOpen},
		{name: "test", from: CircuitHalfOpen, to: CircuitClosed},
	}

	if len(obs.transitions) != len(expected) {
		t.Fatalf("expected %d transitions, got %d", len(expected), len(obs.transitions))
	}

	for i, exp := range expected {
		got := obs.transitions[i]
		if got != exp {
			t.Errorf("transition %d: expected %+v, got %+v", i, exp, got)
		}
	}
}

func TestObserverInheritanceInWrap(t *testing.T) {
	obs := &testObserver{}

	// Create original error with observer
	original := New("original error", WithObserver(obs))

	// Wrap the error - should inherit the observer
	wrapped := Wrap(original, "wrapped error")

	// Log both errors
	original.Log()
	wrapped.Log()

	// Should record 2 errors
	if obs.errorCount != 2 {
		t.Fatalf("expected 2 errors recorded, got %d", obs.errorCount)
	}
}

func TestCircuitBreakerSetObserver(t *testing.T) {
	obs := &testObserver{}

	// Create circuit breaker without observer
	cb := NewCircuitBreaker("test", 1, 10*time.Millisecond)

	// Set observer later
	cb.SetObserver(obs)

	// Trigger a state transition
	cb.RecordFailure()

	expected := []stateChange{
		{name: "test", from: CircuitClosed, to: CircuitOpen},
	}

	if len(obs.transitions) != len(expected) {
		t.Fatalf("expected %d transitions, got %d", len(expected), len(obs.transitions))
	}

	if obs.transitions[0] != expected[0] {
		t.Errorf("expected %+v, got %+v", expected[0], obs.transitions[0])
	}
}

func TestObserverIsOptional(t *testing.T) {
	// Create error without observer - should not panic
	err := New("test error")
	err.Log() // Should not panic

	// Create circuit breaker without observer - should not panic
	cb := NewCircuitBreaker("test", 1, 10*time.Millisecond)
	cb.RecordFailure() // Should not panic
}
