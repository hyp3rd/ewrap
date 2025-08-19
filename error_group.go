package ewrap

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"gopkg.in/yaml.v3"
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

// ErrorOrNil returns the ErrorGroup itself if it contains errors, or nil if empty.
func (eg *ErrorGroup) ErrorOrNil() error {
	if eg.HasErrors() {
		return eg
	}

	return nil
}

// Errors returns a copy of all errors in the group.
func (eg *ErrorGroup) Errors() []error {
	eg.mu.RLock()
	defer eg.mu.RUnlock()

	return slices.Clone(eg.errors)
}

// Join aggregates all errors in the group using errors.Join.
// It returns nil if the group is empty.
func (eg *ErrorGroup) Join() error {
	eg.mu.RLock()
	defer eg.mu.RUnlock()

	return errors.Join(eg.errors...)
}

// Clear removes all errors from the group while preserving capacity.
func (eg *ErrorGroup) Clear() {
	eg.mu.Lock()
	eg.errors = eg.errors[:0]
	eg.mu.Unlock()
}

// SerializableError represents an error in a serializable format.
type SerializableError struct {
	Message    string                 `json:"message"               yaml:"message"`
	Type       string                 `json:"type"                  yaml:"type"`
	StackTrace []StackFrame           `json:"stack_trace,omitempty" yaml:"stack_trace,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"    yaml:"metadata,omitempty"`
	Cause      *SerializableError     `json:"cause,omitempty"       yaml:"cause,omitempty"`
}

// ErrorGroupSerialization represents the serializable format of an ErrorGroup.
type ErrorGroupSerialization struct {
	ErrorCount int                 `json:"error_count" yaml:"error_count"`
	Timestamp  string              `json:"timestamp"   yaml:"timestamp"`
	Errors     []SerializableError `json:"errors"      yaml:"errors"`
}

// toSerializableError converts an error to a SerializableError.
func toSerializableError(err error) SerializableError {
	if err == nil {
		return SerializableError{}
	}

	serErr := SerializableError{
		Message: err.Error(),
		Type:    "standard",
	}

	// Check if it's our custom Error type
	customErr := &Error{}
	if errors.As(err, &customErr) {
		serErr.Type = "ewrap"
		serErr.StackTrace = customErr.GetStackFrames()

		// Get metadata safely
		customErr.mu.RLock()

		if len(customErr.metadata) > 0 {
			serErr.Metadata = make(map[string]interface{}, len(customErr.metadata))
			for k, v := range customErr.metadata {
				serErr.Metadata[k] = v
			}
		}

		customErr.mu.RUnlock()

		// Handle cause
		if customErr.cause != nil {
			cause := toSerializableError(customErr.cause)
			serErr.Cause = &cause
		}
	}

	return serErr
}

// ToSerialization converts the ErrorGroup to a serializable format.
func (eg *ErrorGroup) ToSerialization() ErrorGroupSerialization {
	eg.mu.RLock()
	defer eg.mu.RUnlock()

	serializable := ErrorGroupSerialization{
		ErrorCount: len(eg.errors),
		Timestamp:  time.Now().Format(time.RFC3339),
		Errors:     make([]SerializableError, len(eg.errors)),
	}

	for i, err := range eg.errors {
		serializable.Errors[i] = toSerializableError(err)
	}

	return serializable
}

// ToJSON converts the ErrorGroup to JSON format.
func (eg *ErrorGroup) ToJSON() (string, error) {
	serializable := eg.ToSerialization()

	data, err := json.MarshalIndent(serializable, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal ErrorGroup to JSON: %w", err)
	}

	return string(data), nil
}

// ToYAML converts the ErrorGroup to YAML format.
func (eg *ErrorGroup) ToYAML() (string, error) {
	serializable := eg.ToSerialization()

	data, err := yaml.Marshal(serializable)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ErrorGroup to YAML: %w", err)
	}

	return string(data), nil
}

// MarshalJSON implements the json.Marshaler interface.
func (eg *ErrorGroup) MarshalJSON() ([]byte, error) {
	serializable := eg.ToSerialization()

	data, err := json.Marshal(serializable)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ErrorGroup to JSON: %w", err)
	}

	return data, nil
}

// MarshalYAML implements the yaml.Marshaler interface.
func (eg *ErrorGroup) MarshalYAML() (interface{}, error) {
	return eg.ToSerialization(), nil
}
