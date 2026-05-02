// Package breaker implements the classic circuit-breaker pattern. It is
// independent of the parent ewrap module — consumers who only need error
// wrapping do not pay for it.
//
// The breaker is goroutine-safe. All state transitions happen under a single
// lock; observer and OnStateChange callbacks fire synchronously after the
// lock is released, so callbacks must not invoke the breaker recursively.
package breaker

import (
	"sync"
	"time"
)

// Breaker implements the circuit-breaker pattern.
type Breaker struct {
	name          string
	maxFailures   int
	timeout       time.Duration
	failureCount  int
	lastFailure   time.Time
	state         State
	observer      Observer
	mu            sync.Mutex
	onStateChange func(name string, from, to State)
}

// State represents the breaker's operational state.
type State int

const (
	// Closed indicates normal operation: requests pass through.
	Closed State = iota
	// Open indicates the breaker has tripped: requests are rejected fast.
	Open
	// HalfOpen indicates the breaker is probing recovery: a single request
	// is allowed; success closes the breaker, failure re-opens it.
	HalfOpen
)

// String returns the canonical name of the state.
func (s State) String() string {
	switch s {
	case Closed:
		return "closed"
	case Open:
		return "open"
	case HalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Observer receives notifications when the breaker changes state.
type Observer interface {
	// RecordTransition is called once per state change. Implementations
	// must be goroutine-safe and must not invoke the breaker recursively.
	RecordTransition(name string, from, to State)
}

type noopObserver struct{}

func (noopObserver) RecordTransition(string, State, State) {}

// transitionEvent captures a state change so observer/callback dispatch can
// happen outside the breaker lock.
type transitionEvent struct {
	name     string
	from, to State
	observer Observer
	callback func(string, State, State)
}

// New creates a Breaker named name that opens after maxFailures consecutive
// failures and probes recovery after timeout has elapsed in the open state.
func New(name string, maxFailures int, timeout time.Duration) *Breaker {
	return NewWithObserver(name, maxFailures, timeout, nil)
}

// NewWithObserver creates a Breaker that emits transition events to observer.
// A nil observer is replaced with a no-op implementation.
func NewWithObserver(name string, maxFailures int, timeout time.Duration, observer Observer) *Breaker {
	if observer == nil {
		observer = noopObserver{}
	}

	return &Breaker{
		name:        name,
		maxFailures: maxFailures,
		timeout:     timeout,
		state:       Closed,
		observer:    observer,
	}
}

// Name returns the breaker's identifier as supplied at construction.
func (cb *Breaker) Name() string {
	return cb.name
}

// State returns the current state. The result is a snapshot and may be stale
// by the time the caller acts on it.
func (cb *Breaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return cb.state
}

// OnStateChange installs a callback fired after each state transition. The
// callback runs synchronously outside the breaker lock and must not invoke
// the breaker recursively.
func (cb *Breaker) OnStateChange(callback func(name string, from, to State)) {
	cb.mu.Lock()
	cb.onStateChange = callback
	cb.mu.Unlock()
}

// SetObserver replaces the observer. A nil value is replaced with a no-op
// implementation so callers never need to nil-check before recording.
func (cb *Breaker) SetObserver(observer Observer) {
	if observer == nil {
		observer = noopObserver{}
	}

	cb.mu.Lock()
	cb.observer = observer
	cb.mu.Unlock()
}

// RecordFailure records a failure and potentially opens the breaker.
func (cb *Breaker) RecordFailure() {
	cb.mu.Lock()
	cb.failureCount++
	cb.lastFailure = time.Now()

	var event *transitionEvent
	if cb.state == Closed && cb.failureCount >= cb.maxFailures {
		event = cb.setStateLocked(Open)
	}

	cb.mu.Unlock()

	cb.fireTransition(event)
}

// RecordSuccess records a success. In half-open state this closes the
// breaker; in any other state it is a no-op.
func (cb *Breaker) RecordSuccess() {
	var event *transitionEvent

	cb.mu.Lock()

	if cb.state == HalfOpen {
		cb.failureCount = 0
		event = cb.setStateLocked(Closed)
	}

	cb.mu.Unlock()

	cb.fireTransition(event)
}

// CanExecute reports whether the operation guarded by the breaker should be
// attempted. When the breaker is open and the timeout has elapsed it
// transitions to half-open atomically and returns true.
func (cb *Breaker) CanExecute() bool {
	cb.mu.Lock()

	var (
		can   bool
		event *transitionEvent
	)

	switch cb.state {
	case Closed, HalfOpen:
		can = true
	case Open:
		if time.Since(cb.lastFailure) > cb.timeout {
			event = cb.setStateLocked(HalfOpen)
			can = true
		}
	default:
		// Defensive only; State is a closed enum.
	}

	cb.mu.Unlock()

	cb.fireTransition(event)

	return can
}

// setStateLocked must be called with cb.mu held. It returns a transitionEvent
// when the state actually changes; nil otherwise. The caller is responsible
// for releasing the lock and calling fireTransition.
func (cb *Breaker) setStateLocked(newState State) *transitionEvent {
	if cb.state == newState {
		return nil
	}

	oldState := cb.state
	cb.state = newState

	return &transitionEvent{
		name:     cb.name,
		from:     oldState,
		to:       newState,
		observer: cb.observer,
		callback: cb.onStateChange,
	}
}

// fireTransition dispatches observer and callback notifications for a
// completed transition. Must be called without the lock held.
func (*Breaker) fireTransition(event *transitionEvent) {
	if event == nil {
		return
	}

	if event.observer != nil {
		event.observer.RecordTransition(event.name, event.from, event.to)
	}

	if event.callback != nil {
		event.callback(event.name, event.from, event.to)
	}
}
