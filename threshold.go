package ewrap

import (
	"sync"
	"time"
)

// CircuitBreaker implements the circuit breaker pattern for error handling.
//
// Notifications (observer and OnStateChange callback) are fired synchronously
// after the lock has been released, so callers must not invoke the breaker
// recursively from a callback.
type CircuitBreaker struct {
	name          string
	maxFailures   int
	timeout       time.Duration
	failureCount  int
	lastFailure   time.Time
	state         CircuitState
	observer      Observer
	mu            sync.Mutex
	onStateChange func(name string, from, to CircuitState)
}

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// CircuitClosed indicates normal operation.
	CircuitClosed CircuitState = iota
	// CircuitOpen indicates the circuit is broken.
	CircuitOpen
	// CircuitHalfOpen indicates the circuit is testing recovery.
	CircuitHalfOpen
)

// transitionEvent captures a state change so observer/callback dispatch can
// happen outside the breaker lock.
type transitionEvent struct {
	name     string
	from, to CircuitState
	observer Observer
	callback func(string, CircuitState, CircuitState)
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(name string, maxFailures int, timeout time.Duration) *CircuitBreaker {
	return NewCircuitBreakerWithObserver(name, maxFailures, timeout, nil)
}

// NewCircuitBreakerWithObserver creates a new circuit breaker with an observer.
func NewCircuitBreakerWithObserver(name string, maxFailures int, timeout time.Duration, observer Observer) *CircuitBreaker {
	if observer == nil {
		observer = newNoopObserver()
	}

	return &CircuitBreaker{
		name:        name,
		maxFailures: maxFailures,
		timeout:     timeout,
		state:       CircuitClosed,
		observer:    observer,
	}
}

// OnStateChange sets a callback for state changes.
func (cb *CircuitBreaker) OnStateChange(callback func(name string, from, to CircuitState)) {
	cb.mu.Lock()
	cb.onStateChange = callback
	cb.mu.Unlock()
}

// SetObserver sets an observer for the circuit breaker.
func (cb *CircuitBreaker) SetObserver(observer Observer) {
	if observer == nil {
		observer = newNoopObserver()
	}

	cb.mu.Lock()
	cb.observer = observer
	cb.mu.Unlock()
}

// RecordFailure records a failure and potentially opens the circuit.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	cb.failureCount++
	cb.lastFailure = time.Now()

	var event *transitionEvent
	if cb.state == CircuitClosed && cb.failureCount >= cb.maxFailures {
		event = cb.setStateLocked(CircuitOpen)
	}
	cb.mu.Unlock()

	cb.fireTransition(event)
}

// RecordSuccess records a success and potentially closes the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()

	var event *transitionEvent

	if cb.state == CircuitHalfOpen {
		cb.failureCount = 0
		event = cb.setStateLocked(CircuitClosed)
	}
	cb.mu.Unlock()

	cb.fireTransition(event)
}

// CanExecute checks if the operation can be executed. When the breaker is
// open and the timeout has elapsed it transitions to half-open atomically.
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.Lock()

	var (
		can   bool
		event *transitionEvent
	)

	switch cb.state {
	case CircuitClosed, CircuitHalfOpen:
		can = true
	case CircuitOpen:
		if time.Since(cb.lastFailure) > cb.timeout {
			event = cb.setStateLocked(CircuitHalfOpen)
			can = true
		}
	default:
		// Unknown state — refuse to execute and record nothing. Defensive
		// only; CircuitState is a closed enum.
	}

	cb.mu.Unlock()

	cb.fireTransition(event)

	return can
}

// setStateLocked must be called with cb.mu held. Returns a transitionEvent
// when the state actually changes; nil otherwise. The caller is responsible
// for releasing the lock and calling fireTransition.
func (cb *CircuitBreaker) setStateLocked(newState CircuitState) *transitionEvent {
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
func (*CircuitBreaker) fireTransition(event *transitionEvent) {
	if event == nil {
		return
	}

	if event.observer != nil {
		event.observer.RecordCircuitStateTransition(event.name, event.from, event.to)
	}

	if event.callback != nil {
		event.callback(event.name, event.from, event.to)
	}
}
