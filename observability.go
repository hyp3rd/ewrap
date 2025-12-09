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

func newNoopObserver() Observer {
	return noopObserver{}
}

func (noopObserver) RecordError(string)                                              {}
func (noopObserver) RecordCircuitStateTransition(string, CircuitState, CircuitState) {}
