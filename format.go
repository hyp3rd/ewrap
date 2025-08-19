// Package ewrap provides enhanced error handling capabilities
package ewrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// ErrorOutput represents a formatted error output structure that can be
// serialized to various formats like JSON and YAML.
type ErrorOutput struct {
	// Message contains the main error message
	Message string `json:"message" yaml:"message"`
	// Timestamp indicates when the error occurred
	Timestamp string `json:"timestamp" yaml:"timestamp"`
	// Type categorizes the error
	Type string `json:"type" yaml:"type"`
	// Severity indicates the error's impact level
	Severity string `json:"severity" yaml:"severity"`
	// Stack contains the error stack trace
	Stack string `json:"stack" yaml:"stack"`
	// Cause contains the underlying error if any
	Cause *ErrorOutput `json:"cause,omitempty" yaml:"cause,omitempty"`
	// Context contains additional error context
	Context map[string]any `json:"context,omitempty" yaml:"context,omitempty"`
	// Metadata contains user-defined metadata
	Metadata map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// FormatOption defines formatting options for error output.
type FormatOption func(*ErrorOutput)

// WithTimestampFormat allows customizing the timestamp format in the output.
func WithTimestampFormat(format string) FormatOption {
	return func(eo *ErrorOutput) {
		if format == "" {
			return
		}

		// Attempt to parse existing timestamp and reformat
		if t, err := time.Parse(time.RFC3339, eo.Timestamp); err == nil {
			eo.Timestamp = t.Format(format)
		}
	}
}

// WithStackTrace controls whether to include the stack trace in the output.
func WithStackTrace(include bool) FormatOption {
	return func(eo *ErrorOutput) {
		if !include {
			eo.Stack = ""
		}
	}
}

// toErrorOutput converts an Error to ErrorOutput format.
func (e *Error) toErrorOutput(opts ...FormatOption) *ErrorOutput {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Extract error context if available
	var (
		ctx        *ErrorContext
		contextMap map[string]any
	)

	if rawCtx, ok := e.metadata["error_context"]; ok {
		if ctx, ok = rawCtx.(*ErrorContext); ok {
			contextMap = map[string]any{
				"request_id":  ctx.RequestID,
				"user":        ctx.User,
				"component":   ctx.Component,
				"operation":   ctx.Operation,
				"file":        ctx.File,
				"line":        ctx.Line,
				"environment": ctx.Environment,
			}
		}
	}

	// Create base output structure
	output := &ErrorOutput{
		Message:   e.msg,
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      "unknown",
		Severity:  "error",
		Stack:     e.Stack(),
		Context:   contextMap,
		Metadata:  make(map[string]any),
	}

	// Copy metadata excluding internal keys
	for k, v := range e.metadata {
		if k != "error_context" {
			output.Metadata[k] = v
		}
	}

	// Set error type and severity if available
	if ctx != nil {
		output.Type = ctx.Type.String()
		output.Severity = ctx.Severity.String()
	}

	// Handle wrapped errors
	if e.cause != nil {
		var wrappedErr *Error
		if errors.As(e.cause, &wrappedErr) {
			output.Cause = wrappedErr.toErrorOutput(opts...)
		} else {
			output.Cause = &ErrorOutput{
				Message:  e.cause.Error(),
				Type:     "unknown",
				Severity: "error",
			}
		}
	}

	// Apply formatting options
	for _, opt := range opts {
		opt(output)
	}

	return output
}

// ToJSON converts the error to a JSON string.
func (e *Error) ToJSON(opts ...FormatOption) (string, error) {
	output := e.toErrorOutput(opts...)

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal error to JSON: %w", err)
	}

	return string(data), nil
}

// ToYAML converts the error to a YAML string.
func (e *Error) ToYAML(opts ...FormatOption) (string, error) {
	output := e.toErrorOutput(opts...)

	data, err := yaml.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal error to YAML: %w", err)
	}

	return string(data), nil
}
