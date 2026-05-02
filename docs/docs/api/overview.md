# Package Overview

A condensed reference of the public surface of `github.com/hyp3rd/ewrap`
and its subpackages. For longer prose see [Features](../features/error-creation.md).

## Module layout

```text
github.com/hyp3rd/ewrap            // root package — error type, options, formatting
github.com/hyp3rd/ewrap/breaker    // circuit breaker (independent)
github.com/hyp3rd/ewrap/slog       // slog adapter
```

## Constructors

```go
func New(msg string, opts ...Option) *Error
func NewSkip(skip int, msg string, opts ...Option) *Error
func Newf(format string, args ...any) *Error            // honours %w
func Wrap(err error, msg string, opts ...Option) *Error // nil-safe
func WrapSkip(skip int, err error, msg string, opts ...Option) *Error
func Wrapf(err error, format string, args ...any) *Error // nil-safe
```

## `*Error` methods

```go
// stdlib interfaces
func (e *Error) Error() string                           // cached
func (e *Error) Unwrap() error
func (e *Error) Format(state fmt.State, verb rune)       // %s %v %q %+v
func (e *Error) LogValue() slog.Value                    // structured slog

// inspection
func (e *Error) Cause() error
func (e *Error) Stack() string                           // cached
func (e *Error) GetStackIterator() *StackIterator
func (e *Error) GetStackFrames() []StackFrame
func (e *Error) GetErrorContext() *ErrorContext
func (e *Error) Recovery() *RecoverySuggestion
func (e *Error) Retry() *RetryInfo
func (e *Error) Retryable() (value, set bool)
func (e *Error) SafeError() string

// metadata
func (e *Error) WithMetadata(key string, value any) *Error
func (e *Error) WithContext(ctx *ErrorContext) *Error
func (e *Error) GetMetadata(key string) (any, bool)
func GetMetadataValue[T any](e *Error, key string) (T, bool)

// retry control
func (e *Error) CanRetry() bool
func (e *Error) IncrementRetry()

// logging
func (e *Error) Log()
```

## Options (`type Option func(*Error)`)

| Option | Purpose |
| --- | --- |
| `WithLogger(Logger)` | Attach a structured logger consulted by `(*Error).Log` |
| `WithObserver(Observer)` | Attach an observer that's called from `(*Error).Log` |
| `WithStackDepth(n int)` | Override stack capture depth (0 disables) |
| `WithContext(ctx, type, severity)` | Build an `ErrorContext` from `context.Context` |
| `WithRecoverySuggestion(*RecoverySuggestion)` | Attach recovery guidance |
| `WithRetry(maxAttempts, delay, opts...)` | Attach a retry policy |
| `WithRetryShould(func(error) bool)` | Customise the retry predicate (passed to `WithRetry`) |
| `WithHTTPStatus(int)` | Tag with an HTTP status code |
| `WithRetryable(bool)` | Mark as retryable / permanent (tri-state via pointer) |
| `WithSafeMessage(string)` | Attach a redacted variant returned by `SafeError` |

## Top-level helpers

```go
func HTTPStatus(err error) int            // walks chain; 0 if unset
func IsRetryable(err error) bool          // chain + stdlib Temporary() fallback
func CaptureStack() []uintptr             // raw PC slice at the call site
func GetMetadataValue[T any](e *Error, key string) (T, bool)
```

## Types

```go
type Error struct{ /* unexported */ }            // implements error, fmt.Formatter, slog.LogValuer
type ErrorContext struct{ Type ErrorType; Severity Severity; ... }
type RecoverySuggestion struct{ Message string; Actions []string; Documentation string }
type RetryInfo struct{ MaxAttempts, CurrentAttempt int; Delay time.Duration; ... }
type StackFrame struct{ Function, File string; Line int; PC uintptr }
type StackTrace []StackFrame
type StackIterator struct{ /* unexported */ }
type ErrorOutput struct{ /* JSON/YAML output schema */ }
type ErrorGroup struct{ /* aggregator */ }
type ErrorGroupPool struct{ /* pool */ }
type SerializableError struct{ /* group serialization */ }
type ErrorGroupSerialization struct{ /* group envelope */ }
```

## Interfaces

```go
type Logger interface {
    Error(msg string, keysAndValues ...any)
    Debug(msg string, keysAndValues ...any)
    Info(msg string, keysAndValues ...any)
}

type Observer interface {
    RecordError(message string)
}
```

## Enums

```go
type ErrorType int
const (
    ErrorTypeUnknown ErrorType = iota
    ErrorTypeValidation
    ErrorTypeNotFound
    ErrorTypePermission
    ErrorTypeDatabase
    ErrorTypeNetwork
    ErrorTypeConfiguration
    ErrorTypeInternal
    ErrorTypeExternal
)

type Severity int
const (
    SeverityInfo Severity = iota
    SeverityWarning
    SeverityError
    SeverityCritical
)
```

## Subpackage: `ewrap/breaker`

```go
type State int                    // Closed, Open, HalfOpen; String() supported
type Breaker struct{ /* ... */ }
type Observer interface {
    RecordTransition(name string, from, to State)
}

func New(name string, maxFailures int, timeout time.Duration) *Breaker
func NewWithObserver(name string, maxFailures int, timeout time.Duration, obs Observer) *Breaker

func (cb *Breaker) Name() string
func (cb *Breaker) State() State
func (cb *Breaker) CanExecute() bool
func (cb *Breaker) RecordFailure()
func (cb *Breaker) RecordSuccess()
func (cb *Breaker) OnStateChange(callback func(name string, from, to State))
func (cb *Breaker) SetObserver(obs Observer)
```

See [Circuit Breaker](../features/circuit-breaker.md).

## Subpackage: `ewrap/slog`

```go
type Adapter struct{ /* unexported */ }

func New(logger *slog.Logger) *Adapter

func (a *Adapter) Error(msg string, keysAndValues ...any)
func (a *Adapter) Debug(msg string, keysAndValues ...any)
func (a *Adapter) Info(msg string, keysAndValues ...any)
```

See [`ewrap/slog`](../features/slog-adapter.md).
