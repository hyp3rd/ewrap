package breaker

import (
	"sync"
	"testing"
	"time"
)

const (
	testName             = "test"
	testMaxFailures      = 3
	testTimeoutSeconds   = 5
	testConcurrencyLimit = 100
)

type recordedTransition struct {
	name string
	from State
	to   State
}

// recordingObserver implements Observer for tests.
type recordingObserver struct {
	mu          sync.Mutex
	transitions []recordedTransition
}

func (r *recordingObserver) RecordTransition(name string, from, to State) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.transitions = append(r.transitions, recordedTransition{name, from, to})
}

func (r *recordingObserver) snapshot() []recordedTransition {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]recordedTransition, len(r.transitions))
	copy(out, r.transitions)

	return out
}

func TestNew(t *testing.T) {
	t.Parallel()

	const name = "test-circuit"

	timeout := testTimeoutSeconds * time.Second
	cb := New(name, testMaxFailures, timeout)

	if cb.Name() != name {
		t.Errorf("Name: got %s, want %s", cb.Name(), name)
	}

	if cb.maxFailures != testMaxFailures {
		t.Errorf("maxFailures: got %d, want %d", cb.maxFailures, testMaxFailures)
	}

	if cb.timeout != timeout {
		t.Errorf("timeout: got %v, want %v", cb.timeout, timeout)
	}

	if cb.State() != Closed {
		t.Errorf("State: got %v, want %v", cb.State(), Closed)
	}

	if cb.failureCount != 0 {
		t.Errorf("failureCount: got %d, want 0", cb.failureCount)
	}
}

func TestRecordFailure(t *testing.T) {
	t.Parallel()

	cb := New(testName, 2, testTimeoutSeconds*time.Second)

	cb.RecordFailure()

	if cb.State() != Closed {
		t.Errorf("State after first failure: got %v, want %v", cb.State(), Closed)
	}

	if cb.failureCount != 1 {
		t.Errorf("failureCount: got %d, want 1", cb.failureCount)
	}

	cb.RecordFailure()

	if cb.State() != Open {
		t.Errorf("State after max failures: got %v, want %v", cb.State(), Open)
	}

	if cb.failureCount != 2 {
		t.Errorf("failureCount: got %d, want 2", cb.failureCount)
	}
}

func TestRecordSuccess(t *testing.T) {
	t.Parallel()

	cb := New(testName, 1, testTimeoutSeconds*time.Second)

	cb.RecordFailure()

	if cb.State() != Open {
		t.Error("expected Breaker to be Open after first failure")
	}

	cb.mu.Lock()
	cb.state = HalfOpen
	cb.mu.Unlock()

	cb.RecordSuccess()

	if cb.State() != Closed {
		t.Errorf("State after success in half-open: got %v, want %v", cb.State(), Closed)
	}

	if cb.failureCount != 0 {
		t.Errorf("failureCount reset: got %d, want 0", cb.failureCount)
	}
}

func TestCanExecute(t *testing.T) {
	t.Parallel()

	timeout := testConcurrencyLimit * time.Millisecond
	cb := New(testName, 1, timeout)

	if !cb.CanExecute() {
		t.Error("expected CanExecute true for closed breaker")
	}

	cb.RecordFailure()

	if cb.CanExecute() {
		t.Error("expected CanExecute false for open breaker")
	}

	time.Sleep(timeout + 10*time.Millisecond)

	if !cb.CanExecute() {
		t.Error("expected CanExecute true after timeout (half-open)")
	}

	if got := cb.State(); got != HalfOpen {
		t.Errorf("State after timeout: got %v, want %v", got, HalfOpen)
	}
}

func TestOnStateChange(t *testing.T) {
	t.Parallel()

	cb := New(testName, 1, testTimeoutSeconds*time.Second)

	type recordedChange struct {
		name string
		from State
		to   State
	}

	var (
		changes []recordedChange
		mu      sync.Mutex
	)

	cb.OnStateChange(func(name string, from, to State) {
		mu.Lock()

		changes = append(changes, recordedChange{name, from, to})

		mu.Unlock()
	})

	cb.RecordFailure()

	mu.Lock()
	defer mu.Unlock()

	if len(changes) != 1 {
		t.Errorf("expected 1 state change, got %d", len(changes))

		return
	}

	change := changes[0]
	if change.name != testName {
		t.Errorf("name: got %s, want %s", change.name, testName)
	}

	if change.from != Closed {
		t.Errorf("from: got %v, want %v", change.from, Closed)
	}

	if change.to != Open {
		t.Errorf("to: got %v, want %v", change.to, Open)
	}
}

func TestTransitionViaPublicAPI(t *testing.T) {
	t.Parallel()

	cb := New(testName, 1, testTimeoutSeconds*time.Second)

	cb.RecordFailure()

	if cb.State() != Open {
		t.Errorf("State: got %v, want %v", cb.State(), Open)
	}

	cb.RecordFailure()

	if cb.State() != Open {
		t.Error("expected State to remain Open on subsequent failure")
	}
}

func TestConcurrency(t *testing.T) {
	t.Parallel()

	cb := New(testName, testTimeoutSeconds, testConcurrencyLimit*time.Millisecond)

	var wg sync.WaitGroup

	for range testConcurrencyLimit {
		wg.Go(func() {
			cb.RecordFailure()
		})
	}

	for range testConcurrencyLimit {
		wg.Go(func() {
			cb.CanExecute()
		})
	}

	wg.Wait()

	if cb.State() != Open {
		t.Errorf("expected breaker to be Open after many failures, got %v", cb.State())
	}
}

func TestStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state    State
		expected State
		name     string
	}{
		{Closed, Closed, "closed"},
		{Open, Open, "open"},
		{HalfOpen, HalfOpen, "half-open"},
	}

	for _, tt := range tests {
		if tt.state != tt.expected {
			t.Errorf("state mismatch: got %v, expected %v", tt.state, tt.expected)
		}

		if tt.state.String() != tt.name {
			t.Errorf("State.String(): got %q, want %q", tt.state.String(), tt.name)
		}
	}
}

func TestObserverViaConstructor(t *testing.T) {
	t.Parallel()

	obs := &recordingObserver{}

	timeout := 10 * time.Millisecond
	cb := NewWithObserver(testName, 1, timeout, obs)

	cb.RecordFailure()
	time.Sleep(timeout + time.Millisecond)

	if !cb.CanExecute() {
		t.Fatal("expected breaker to allow execution after timeout")
	}

	cb.RecordSuccess()

	expected := []recordedTransition{
		{name: testName, from: Closed, to: Open},
		{name: testName, from: Open, to: HalfOpen},
		{name: testName, from: HalfOpen, to: Closed},
	}

	got := obs.snapshot()
	if len(got) != len(expected) {
		t.Fatalf("expected %d transitions, got %d", len(expected), len(got))
	}

	for i, exp := range expected {
		if got[i] != exp {
			t.Errorf("transition %d: expected %+v, got %+v", i, exp, got[i])
		}
	}
}

func TestSetObserver(t *testing.T) {
	t.Parallel()

	obs := &recordingObserver{}

	cb := New(testName, 1, 10*time.Millisecond)

	cb.SetObserver(obs)

	cb.RecordFailure()

	got := obs.snapshot()
	if len(got) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(got))
	}

	want := recordedTransition{name: testName, from: Closed, to: Open}
	if got[0] != want {
		t.Errorf("transition: got %+v, want %+v", got[0], want)
	}
}

func TestNoObserverIsSafe(t *testing.T) {
	t.Parallel()

	cb := New(testName, 1, 10*time.Millisecond)
	cb.RecordFailure() // Must not panic with default no-op observer
}
