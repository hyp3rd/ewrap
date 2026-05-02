// Package breaker implements the classic circuit-breaker pattern. It is
// independent of the parent ewrap module — consumers who only need error
// wrapping do not pay for it.
//
// The breaker is goroutine-safe. All state transitions happen under a single
// lock; observer and OnStateChange callbacks fire synchronously after the
// lock is released, so callbacks must not invoke the breaker recursively.
package breaker
