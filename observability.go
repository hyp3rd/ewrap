package ewrap

// Observer defines hooks for observing errors and circuit breaker state transitions.
type Observer interface {
	// RecordError is called when an error is logged.
	RecordError(message string)
	// RecordCircuitStateTransition is called when a circuit breaker changes state.
	RecordCircuitStateTransition(name string, from, to CircuitState)
}

// noopObserver provides a no-op implementation of the Observer interface.
type noopObserver struct{}

func (noopObserver) RecordError(string)                                              {}
func (noopObserver) RecordCircuitStateTransition(string, CircuitState, CircuitState) {}

var observer Observer = noopObserver{}

// SetObserver sets the global observer. Passing nil resets to a no-op observer.
func SetObserver(o Observer) {
	if o == nil {
		observer = noopObserver{}
		return
	}
	observer = o
}
