package ewrap

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

const (
	poolCapacity           = 4
	initialBuilderCapacity = 256
)

// ErrorGroupPool manages a pool of ErrorGroup instances.
// By making this a separate type, we give users control over pool lifecycle
// and configuration while maintaining encapsulation.
type ErrorGroupPool struct {
	pool sync.Pool
}

// NewErrorGroupPool creates a new pool for error groups with the specified
// initial capacity for the error slices.
func NewErrorGroupPool(initialCapacity int) *ErrorGroupPool {
	if initialCapacity < 0 {
		initialCapacity = poolCapacity // sensible default if given invalid capacity
	}

	return &ErrorGroupPool{
		pool: sync.Pool{
			New: func() any {
				return &ErrorGroup{
					errors: make([]error, 0, initialCapacity),
					pool:   nil, // Will be set when retrieved from pool
				}
			},
		},
	}
}

// Get retrieves an ErrorGroup from the pool or creates a new one if the pool is empty.
func (p *ErrorGroupPool) Get() *ErrorGroup {
	eg, ok := p.pool.Get().(*ErrorGroup)
	if !ok {
		// log and return a new ErrorGroup skipping the pool.
		slog.Error("unable to initiate an ErrorGroup from the pool")

		return NewErrorGroup()
	}

	eg.pool = p // Set the pool reference for later release
	eg.Clear()  // Ensure the error group is clean

	return eg
}

// put returns an ErrorGroup to the pool.
func (p *ErrorGroupPool) put(eg *ErrorGroup) {
	eg.Clear()
	eg.pool = nil // Clear pool reference to prevent memory leaks
	p.pool.Put(eg)
}

// ErrorGroup represents a collection of related errors.
// It maintains a reference to its pool for proper release handling.
//
//nolint:errname
type ErrorGroup struct {
	errors []error
	pool   *ErrorGroupPool // Reference to the pool this group came from
	mu     sync.RWMutex
}

// NewErrorGroup creates a standalone ErrorGroup without pooling.
// This is useful for cases where pooling isn't needed or for testing.
func NewErrorGroup() *ErrorGroup {
	return &ErrorGroup{
		errors: make([]error, 0, poolCapacity),
	}
}

// Release returns the ErrorGroup to its pool if it came from one.
// If the ErrorGroup wasn't created from a pool, Release is a no-op.
func (eg *ErrorGroup) Release() {
	if eg.pool != nil {
		eg.pool.put(eg)
	}
}

// Add appends an error to the group if it's not nil.
func (eg *ErrorGroup) Add(err error) {
	if err == nil {
		return
	}

	eg.mu.Lock()
	eg.errors = append(eg.errors, err)
	eg.mu.Unlock()
}

// HasErrors returns true if the group contains any errors.
func (eg *ErrorGroup) HasErrors() bool {
	eg.mu.RLock()
	defer eg.mu.RUnlock()

	return len(eg.errors) > 0
}

// Error implements the error interface.
func (eg *ErrorGroup) Error() string {
	eg.mu.RLock()
	defer eg.mu.RUnlock()

	switch len(eg.errors) {
	case 0:
		return ""
	case 1:
		return eg.errors[0].Error()
	default:
		var builder strings.Builder

		builder.Grow(initialBuilderCapacity) // Pre-allocate space for efficiency

		fmt.Fprintf(&builder, "%d errors occurred:\n", len(eg.errors))

		for i, err := range eg.errors {
			fmt.Fprintf(&builder, "%d: %s\n", i+1, err.Error())
		}

		return builder.String()
	}
}

// Errors returns a copy of all errors in the group.
func (eg *ErrorGroup) Errors() []error {
	eg.mu.RLock()
	defer eg.mu.RUnlock()

	result := make([]error, len(eg.errors))
	copy(result, eg.errors)

	return result
}

// Clear removes all errors from the group while preserving capacity.
func (eg *ErrorGroup) Clear() {
	eg.mu.Lock()
	eg.errors = eg.errors[:0]
	eg.mu.Unlock()
}
