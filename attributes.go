package ewrap

import "errors"

// WithHTTPStatus tags the error with an HTTP status code. The first non-zero
// status found while walking the chain via errors.As is what HTTPStatus
// returns to the caller. Use net/http constants (e.g. http.StatusBadRequest).
func WithHTTPStatus(status int) Option {
	return func(err *Error) {
		err.httpStatus = status
	}
}

// HTTPStatus walks the chain and returns the first attached HTTP status
// code, or 0 if none is set.
func HTTPStatus(err error) int {
	for err != nil {
		var e *Error
		if errors.As(err, &e) && e.httpStatus != 0 {
			return e.httpStatus
		}

		err = errors.Unwrap(err)
	}

	return 0
}

// WithRetryable marks the error as transient (true) or permanent (false). It
// is consulted by IsRetryable and by middleware deciding whether to retry.
//
// If unset, IsRetryable falls back to the stdlib Temporary() bool interface
// where available.
func WithRetryable(retryable bool) Option {
	return func(err *Error) {
		err.retryable = &retryable
	}
}

// Retryable reports whether the error has been explicitly classified as
// retryable. Callers usually want IsRetryable instead since that walks the
// chain and honors stdlib markers.
func (e *Error) Retryable() (value, set bool) {
	if e.retryable == nil {
		return false, false
	}

	return *e.retryable, true
}

// IsRetryable reports whether an error should be retried. It walks the chain
// looking for an explicit ewrap classification first; falling back to the
// stdlib `interface{ Temporary() bool }` (as exposed by net.Error and
// similar) when no explicit value has been set.
func IsRetryable(err error) bool {
	for cur := err; cur != nil; cur = errors.Unwrap(cur) {
		var e *Error
		if errors.As(cur, &e) {
			if v, set := e.Retryable(); set {
				return v
			}
		}

		if t, ok := cur.(interface{ Temporary() bool }); ok {
			return t.Temporary()
		}
	}

	return false
}

// WithSafeMessage attaches a redacted variant of the error message that
// SafeError will return instead of msg. Use this when the unredacted
// message contains PII or other content that must not leak into external
// logs/sinks.
func WithSafeMessage(safe string) Option {
	return func(err *Error) {
		err.safeMsg = safe
	}
}

// SafeError returns a redacted version of the error chain suitable for
// logging into untrusted sinks. Each layer contributes either its
// WithSafeMessage value (if set) or its raw msg. Standard wrapped errors
// without a SafeError method are included verbatim — callers redacting
// upstream errors must wrap them in an ewrap.Error with WithSafeMessage.
func (e *Error) SafeError() string {
	msg := e.msg
	if e.safeMsg != "" {
		msg = e.safeMsg
	}

	if e.cause == nil || e.fullMsg {
		return msg
	}

	if c, ok := e.cause.(interface{ SafeError() string }); ok {
		return msg + ": " + c.SafeError()
	}

	return msg + ": " + e.cause.Error()
}
