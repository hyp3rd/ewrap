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

	SetObserver(obs)
	defer SetObserver(nil)

	err := New("boom")
	err.Log()

	if obs.errorCount != 1 {
		t.Fatalf("expected 1 error recorded, got %d", obs.errorCount)
	}
}

func TestCircuitBreakerObserver(t *testing.T) {
	obs := &testObserver{}

	SetObserver(obs)
	defer SetObserver(nil)

	timeout := 10 * time.Millisecond
	cb := NewCircuitBreaker("test", 1, timeout)

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
