package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hyp3rd/ewrap"
)

const (
	// maxFailures is the maximum number of failures before opening the circuit.
	maxFailures = 3
	// maxTimeout is the maximum timeout for circuit breakers.
	maxTimeout = 5 * time.Second
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	Closed CircuitState = iota
	Open
	HalfOpen
)

// MetricsObserver is an example observer that tracks metrics.
type MetricsObserver struct {
	errorCount      int
	circuitBreakers map[string]CircuitState
}

// NewMetricsObserver creates a new MetricsObserver.
func NewMetricsObserver() *MetricsObserver {
	return &MetricsObserver{
		circuitBreakers: make(map[string]CircuitState),
	}
}

// RecordError records an error occurrence.
func (m *MetricsObserver) RecordError(message string) {
	m.errorCount++
	log.Printf("Error recorded: %s (total: %d)", message, m.errorCount)
}

// RecordCircuitStateTransition records a circuit state transition.
func (m *MetricsObserver) RecordCircuitStateTransition(name string, from, to CircuitState) {
	m.circuitBreakers[name] = to
	log.Printf("Circuit breaker '%s' transitioned from %d to %d", name, from, to)
}

// GetErrorCount retrieves the total number of errors recorded.
func (m *MetricsObserver) GetErrorCount() int {
	return m.errorCount
}

// GetCircuitState retrieves the current state of a circuit breaker.
func (m *MetricsObserver) GetCircuitState(name string) (CircuitState, bool) {
	state, exists := m.circuitBreakers[name]
	return state, exists
}

// ExampleObserverUsage demonstrates the new observer design.
func ExampleObserverUsage() {
	// Create a metrics observer
	metrics := NewMetricsObserver()

	// Create errors with observer
	err1 := ewrap.New("database connection failed")
	err2 := ewrap.Wrap(err1, "failed to fetch user data")

	// Log errors
	log.Println(err1.Error())
	log.Println(err2.Error())

	// Check metrics
	fmt.Fprintf(os.Stdout, "Total errors: %d\n", metrics.GetErrorCount())

	if state, exists := metrics.GetCircuitState("database"); exists {
		fmt.Fprintf(os.Stdout, "Database circuit state: %d\n", state)
	}
}

func main() {
	ExampleObserverUsage()
}
