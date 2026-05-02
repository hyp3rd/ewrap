// Package ewrap provides enhanced error handling capabilities with stack traces,
// error wrapping, custom error types, and logging integration.
package ewrap

import (
	"errors"
	"fmt"
	"maps"
	"runtime"
	"strings"
	"sync"
)

const (
	baseLogDataSize = 4
	// defaultStackDepth is the default number of frames captured when no
	// override is supplied via WithStackDepth.
	defaultStackDepth = 32
	// callerSkipNew is the number of frames runtime.Callers should skip so
	// the captured stack starts at the user's call site rather than inside
	// ewrap. Tuned for direct calls to New / Wrap / Newf / Wrapf.
	callerSkipNew = 3
)

// Error represents a custom error type with stack trace and structured metadata.
//
// Fields populated by package-provided options (ErrorContext, RecoverySuggestion,
// RetryInfo) are stored in dedicated typed fields so they cannot collide with
// arbitrary user-supplied metadata keys. The formatted error string and stack
// trace are computed lazily and cached on first access.
type Error struct {
	msg          string
	cause        error
	stack        []uintptr
	metadata     map[string]any
	errorContext *ErrorContext
	recovery     *RecoverySuggestion
	retry        *RetryInfo
	logger       Logger
	observer     Observer

	// httpStatus carries an HTTP status code attached via WithHTTPStatus.
	// Zero means unset.
	httpStatus int
	// retryable holds an explicit retry classification (tri-state via pointer:
	// nil = not classified, &true / &false = explicit).
	retryable *bool
	// safeMsg is a redacted variant of msg returned by SafeError when set.
	safeMsg string

	// fullMsg is set when msg already includes the cause text (e.g. constructed
	// via Newf with %w). When true, Error() returns msg verbatim.
	fullMsg bool

	// mu protects metadata mutation and retry mutation. Cached strings use
	// sync.Once so they need no separate lock.
	mu sync.RWMutex

	// Cached lazy outputs. errOnce/errStr cache Error(); stackOnce/stackStr
	// cache the formatted stack trace.
	errOnce   sync.Once
	errStr    string
	stackOnce sync.Once
	stackStr  string
}

// Option defines the signature for configuration options.
type Option func(*Error)

// WithLogger sets a logger for the error.
func WithLogger(log Logger) Option {
	return func(err *Error) {
		err.logger = log

		if log != nil {
			log.Debug("error created",
				"message", err.msg,
				"stack", err.Stack(),
			)
		}
	}
}

// WithObserver sets an observer for the error.
func WithObserver(observer Observer) Option {
	return func(err *Error) {
		err.observer = observer
	}
}

// WithStackDepth overrides the default number of stack frames captured.
// Pass 0 to disable stack capture entirely; clamps to a minimum of 1
// otherwise. Must be supplied at construction time (it has no effect on
// already-constructed errors).
func WithStackDepth(depth int) Option {
	return func(err *Error) {
		if depth <= 0 {
			err.stack = nil

			return
		}

		err.stack = capturePCs(callerSkipNew, depth)
	}
}

// New creates a new Error with a stack trace and applies the provided options.
func New(msg string, opts ...Option) *Error {
	return newAt(callerSkipNew, msg, opts...)
}

// NewSkip is like New but skips an additional frames stack frames so callers
// wrapping New in a helper see their own location captured.
func NewSkip(skip int, msg string, opts ...Option) *Error {
	return newAt(callerSkipNew+skip, msg, opts...)
}

func newAt(skip int, msg string, opts ...Option) *Error {
	err := &Error{
		msg:   msg,
		stack: capturePCs(skip, defaultStackDepth),
	}

	for _, opt := range opts {
		opt(err)
	}

	return err
}

// Newf creates a new Error with a formatted message.
//
// If format contains the %w verb, the matching argument is preserved as the
// error's cause so that errors.Is/As walk through it. The resulting Error
// behaves like fmt.Errorf with respect to message text and unwrap chain.
func Newf(format string, args ...any) *Error {
	return newfAt(callerSkipNew, format, args...)
}

func newfAt(skip int, format string, args ...any) *Error {
	if !strings.Contains(format, "%w") {
		return newAt(skip+1, fmt.Sprintf(format, args...))
	}

	formatted := fmt.Errorf(format, args...)

	var cause error
	if u, ok := formatted.(interface{ Unwrap() error }); ok {
		cause = u.Unwrap()
	} else if u, ok := formatted.(interface{ Unwrap() []error }); ok {
		if causes := u.Unwrap(); len(causes) > 0 {
			cause = causes[0]
		}
	}

	return &Error{
		msg:     formatted.Error(),
		cause:   cause,
		stack:   capturePCs(skip+1, defaultStackDepth),
		fullMsg: true,
	}
}

// Wrap wraps an existing error with additional context, capturing a stack
// trace at the wrap site. Returns nil if err is nil.
//
// When the wrapped error is itself an *Error, the wrapper inherits its
// metadata, error context, recovery suggestion, retry info, logger and
// observer. Each wrapper retains its own stack frames so deep chains carry
// the full call history.
func Wrap(err error, msg string, opts ...Option) *Error {
	return wrapAt(callerSkipNew, err, msg, opts...)
}

// WrapSkip is like Wrap but skips an additional skip stack frames.
func WrapSkip(skip int, err error, msg string, opts ...Option) *Error {
	return wrapAt(callerSkipNew+skip, err, msg, opts...)
}

func wrapAt(skip int, err error, msg string, opts ...Option) *Error {
	if err == nil {
		return nil
	}

	wrapped := &Error{
		msg:   msg,
		cause: err,
		stack: capturePCs(skip, defaultStackDepth),
	}

	var inner *Error
	if errors.As(err, &inner) {
		inner.mu.RLock()

		if len(inner.metadata) > 0 {
			wrapped.metadata = maps.Clone(inner.metadata)
		}

		wrapped.errorContext = inner.errorContext
		wrapped.recovery = inner.recovery
		wrapped.retry = inner.retry
		wrapped.observer = inner.observer
		wrapped.logger = inner.logger
		wrapped.httpStatus = inner.httpStatus
		wrapped.retryable = inner.retryable
		inner.mu.RUnlock()
	}

	for _, opt := range opts {
		opt(wrapped)
	}

	return wrapped
}

// Wrapf wraps an error with a formatted message.
func Wrapf(err error, format string, args ...any) *Error {
	if err == nil {
		return nil
	}

	return wrapAt(callerSkipNew, err, fmt.Sprintf(format, args...))
}

// Error implements the error interface. The result is computed once on first
// call and cached; subsequent calls are lock-free reads.
func (e *Error) Error() string {
	e.errOnce.Do(func() {
		switch {
		case e.fullMsg, e.cause == nil:
			e.errStr = e.msg
		default:
			e.errStr = e.msg + ": " + e.cause.Error()
		}
	})

	return e.errStr
}

// Cause returns the underlying cause of the error.
func (e *Error) Cause() error {
	return e.cause
}

// WithMetadata adds metadata to the error.
//
// The key namespace is reserved for user data; package-managed values (error
// context, recovery suggestion, retry info) live in dedicated accessors.
func (e *Error) WithMetadata(key string, value any) *Error {
	e.mu.Lock()

	if e.metadata == nil {
		e.metadata = make(map[string]any)
	}

	e.metadata[key] = value
	log := e.logger
	e.mu.Unlock()

	if log != nil {
		log.Debug("metadata added",
			"key", key,
			"value", value,
			"error", e.msg,
		)
	}

	return e
}

// WithContext attaches an existing ErrorContext to the error.
func (e *Error) WithContext(ctx *ErrorContext) *Error {
	e.errorContext = ctx

	if e.logger != nil {
		e.logger.Debug("context added",
			"context", ctx,
			"error", e.msg,
		)
	}

	return e
}

// WithRecoverySuggestion attaches recovery guidance to the error.
func WithRecoverySuggestion(rs *RecoverySuggestion) Option {
	return func(err *Error) {
		err.recovery = rs

		if err.logger != nil && rs != nil {
			logData := []any{"message", rs.Message}
			if len(rs.Actions) > 0 {
				logData = append(logData, "actions", rs.Actions)
			}

			if rs.Documentation != "" {
				logData = append(logData, "documentation", rs.Documentation)
			}

			err.logger.Info("recovery suggestion added", logData...)
		}
	}
}

// GetMetadata retrieves user-defined metadata from the error.
func (e *Error) GetMetadata(key string) (any, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	val, ok := e.metadata[key]

	return val, ok
}

// GetMetadataValue retrieves user-defined metadata and casts it to type T.
func GetMetadataValue[T any](e *Error, key string) (T, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var zero T

	val, ok := e.metadata[key]
	if !ok {
		return zero, false
	}

	typedVal, ok := val.(T)
	if !ok {
		return zero, false
	}

	return typedVal, true
}

// GetErrorContext returns the structured error context, or nil if none was set.
func (e *Error) GetErrorContext() *ErrorContext {
	return e.errorContext
}

// Recovery returns the recovery suggestion attached to the error, or nil.
func (e *Error) Recovery() *RecoverySuggestion {
	return e.recovery
}

// Retry returns the retry information attached to the error, or nil.
func (e *Error) Retry() *RetryInfo {
	return e.retry
}

// Stack returns the stack trace as a string, with runtime and ewrap-package
// frames filtered out so callers see their own code first. The result is
// computed once and cached.
func (e *Error) Stack() string {
	e.stackOnce.Do(func() {
		if len(e.stack) == 0 {
			return
		}

		var builder strings.Builder

		frames := runtime.CallersFrames(e.stack)

		for {
			frame, more := frames.Next()
			if !isInternalFrame(frame) {
				_, _ = fmt.Fprintf(&builder, "%s:%d - %s\n", frame.File, frame.Line, frame.Function)
			}

			if !more {
				break
			}
		}

		e.stackStr = builder.String()
	})

	return e.stackStr
}

// Log logs the error using the configured logger.
func (e *Error) Log() {
	if e.observer != nil {
		e.observer.RecordError(e.msg)
	}

	if e.logger == nil {
		return
	}

	e.mu.RLock()
	logData := make([]any, 0, len(e.metadata)*2+baseLogDataSize)
	logData = append(logData, "error", e.msg)

	if e.cause != nil {
		logData = append(logData, "cause", e.cause.Error())
	}

	logData = append(logData, "stack", e.Stack())

	for key, val := range e.metadata {
		logData = append(logData, key, val)
	}

	e.mu.RUnlock()

	if e.recovery != nil {
		logData = appendRecoverySuggestion(logData, e.recovery)
	}

	e.logger.Error("error occurred", logData...)
}

// CaptureStack captures the current stack trace at the call site using the
// default depth.
func CaptureStack() []uintptr {
	return capturePCs(callerSkipNew, defaultStackDepth)
}

// capturePCs returns the program counters of the current call stack starting
// skip frames up. The slice is sized to depth so callers with shallow stacks
// don't carry empty trailing slots.
func capturePCs(skip, depth int) []uintptr {
	if depth <= 0 {
		return nil
	}

	pcs := make([]uintptr, depth)
	n := runtime.Callers(skip, pcs)

	return pcs[:n]
}

// Unwrap provides compatibility with Go 1.13 error chains. errors.Is and
// errors.As walk the chain via this method; the package-level Is method is
// intentionally not implemented so the stdlib semantics apply unchanged.
func (e *Error) Unwrap() error {
	return e.cause
}

// isInternalFrame returns true for frames the user shouldn't see in a stack
// trace: runtime internals and ewrap's own non-test implementation. Test
// files in the same package are allowed through so users running ewrap's
// own tests still see useful traces.
func isInternalFrame(frame runtime.Frame) bool {
	if strings.HasPrefix(frame.Function, "runtime.") {
		return true
	}

	if !strings.HasPrefix(frame.Function, "github.com/hyp3rd/ewrap.") {
		return false
	}

	if strings.HasSuffix(frame.File, "_test.go") {
		return false
	}

	return true
}

// appendRecoverySuggestion extracts recovery suggestion data for logging.
func appendRecoverySuggestion(logData []any, rs *RecoverySuggestion) []any {
	logData = append(logData, "recovery_message", rs.Message)

	if len(rs.Actions) > 0 {
		logData = append(logData, "recovery_actions", rs.Actions)
	}

	if rs.Documentation != "" {
		logData = append(logData, "recovery_documentation", rs.Documentation)
	}

	return logData
}
