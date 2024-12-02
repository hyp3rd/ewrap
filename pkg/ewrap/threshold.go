package ewrap

import (
	"sync"
	"time"
)

// CircuitBreaker implements the circuit breaker pattern for error handling.
type CircuitBreaker struct {
	name          string
	maxFailures   int
	timeout       time.Duration
	failureCount  int
	lastFailure   time.Time
	state         CircuitState
	mu            sync.RWMutex
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

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(name string, maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:        name,
		maxFailures: maxFailures,
		timeout:     timeout,
		state:       CircuitClosed,
	}
}

// OnStateChange sets a callback for state changes.
func (cb *CircuitBreaker) OnStateChange(callback func(name string, from, to CircuitState)) {
	cb.mu.Lock()
	cb.onStateChange = callback
	cb.mu.Unlock()
}

// RecordFailure records a failure and potentially opens the circuit.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailure = time.Now()

	if cb.state == CircuitClosed && cb.failureCount >= cb.maxFailures {
		cb.transitionTo(CircuitOpen)
	}
}

// RecordSuccess records a success and potentially closes the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitHalfOpen {
		cb.failureCount = 0
		cb.transitionTo(CircuitClosed)
	}
}

// CanExecute checks if the operation can be executed.
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.transitionTo(CircuitHalfOpen)
			cb.mu.Unlock()
			cb.mu.RLock()

			return true
		}

		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// transitionTo changes the circuit breaker state.
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState

	if cb.onStateChange != nil {
		go cb.onStateChange(cb.name, oldState, newState)
	}
}
