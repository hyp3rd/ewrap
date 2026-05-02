package ewrap

// Observer receives notifications about errors. Implementations must be
// goroutine-safe; calls happen synchronously from the goroutine that invoked
// (*Error).Log.
//
// Breaker-state observation lives in the ewrap/breaker subpackage so
// consumers who only need error wrapping do not depend on it.
type Observer interface {
	// RecordError is called when an error is logged.
	RecordError(message string)
}
