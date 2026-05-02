package ewrap

import (
	"fmt"
	"log/slog"
)

// Format implements fmt.Formatter. It supports the canonical pkg/errors-style
// verbs:
//
//	%s   the error message (same as Error())
//	%q   double-quoted error message
//	%v   the error message (default)
//	%+v  the error message followed by the stack trace
func (e *Error) Format(state fmt.State, verb rune) {
	switch verb {
	case 'v':
		if state.Flag('+') {
			fmt.Fprintf(state, "%s\n%s", e.Error(), e.Stack())

			return
		}

		fmt.Fprint(state, e.Error())
	case 'q':
		fmt.Fprintf(state, "%q", e.Error())
	default:
		fmt.Fprint(state, e.Error())
	}
}

// LogValue implements slog.LogValuer so structured loggers receive the error
// as a group of fields rather than an opaque string.
func (e *Error) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("message", e.Error()),
	}

	if ctx := e.errorContext; ctx != nil {
		attrs = append(attrs,
			slog.String("type", ctx.Type.String()),
			slog.String("severity", ctx.Severity.String()),
		)

		if ctx.Component != "" {
			attrs = append(attrs, slog.String("component", ctx.Component))
		}

		if ctx.Operation != "" {
			attrs = append(attrs, slog.String("operation", ctx.Operation))
		}

		if ctx.RequestID != "" {
			attrs = append(attrs, slog.String("request_id", ctx.RequestID))
		}
	}

	if rs := e.recovery; rs != nil {
		attrs = append(attrs, slog.String("recovery", rs.Message))
	}

	e.mu.RLock()

	for k, v := range e.metadata {
		attrs = append(attrs, slog.Any(k, v))
	}

	e.mu.RUnlock()

	if e.cause != nil {
		attrs = append(attrs, slog.String("cause", e.cause.Error()))
	}

	return slog.GroupValue(attrs...)
}
