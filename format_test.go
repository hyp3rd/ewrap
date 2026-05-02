package ewrap

import (
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"gopkg.in/yaml.v3"
)

const (
	formatTestYear  = 2024
	formatTestLine  = 42
	formatTestMonth = 1
	formatTestDay   = 2
	formatTestHour  = 15
	formatTestMin   = 4
	formatTestSec   = 5
	dateOnlyLayout  = "2006-01-02"
	unexpectedErrFn = "Unexpected error: %v"
)

func TestWithTimestampFormat(t *testing.T) {
	t.Parallel()

	tFixed := time.Date(formatTestYear, formatTestMonth, formatTestDay, formatTestHour, formatTestMin, formatTestSec, 0, time.UTC)
	output := &ErrorOutput{
		Timestamp: tFixed.Format(time.RFC3339),
	}

	opt := WithTimestampFormat(dateOnlyLayout)
	opt(output)

	expected := tFixed.Format(dateOnlyLayout)
	if output.Timestamp != expected {
		t.Errorf("Expected timestamp %s, got %s", expected, output.Timestamp)
	}

	output.Timestamp = tFixed.Format(time.RFC3339)
	opt = WithTimestampFormat("")
	opt(output)

	if output.Timestamp != tFixed.Format(time.RFC3339) {
		t.Error("Expected timestamp to remain unchanged with empty format")
	}
}

func TestWithStackTrace(t *testing.T) {
	t.Parallel()

	output := &ErrorOutput{
		Stack: msgSomeStack,
	}

	opt := WithStackTrace(false)
	opt(output)

	if output.Stack != "" {
		t.Error("Expected stack trace to be empty when excluded")
	}

	output.Stack = msgSomeStack
	opt = WithStackTrace(true)
	opt(output)

	if output.Stack != msgSomeStack {
		t.Error("Expected stack trace to remain when included")
	}
}

func TestToErrorOutput(t *testing.T) {
	t.Parallel()

	err := New(msgTestError)

	output := err.toErrorOutput()

	if output.Message != msgTestError {
		t.Errorf("Expected message %q, got %q", msgTestError, output.Message)
	}

	if output.Type != typeUnknownStr {
		t.Errorf("Expected type %q, got %q", typeUnknownStr, output.Type)
	}

	if output.Severity != severityErrorStr {
		t.Errorf("Expected severity %q, got %q", severityErrorStr, output.Severity)
	}

	if output.Metadata == nil {
		t.Error("Expected metadata to be initialized")
	}
}

func TestToErrorOutputWithContext(t *testing.T) {
	t.Parallel()

	ctx := &ErrorContext{
		RequestID:   "req-123",
		User:        "testuser",
		Component:   "test-component",
		Operation:   "test-op",
		File:        "test.go",
		Line:        formatTestLine,
		Environment: msgTest,
		Type:        ErrorTypeInternal,
		Severity:    SeverityCritical,
	}

	err := New(msgTestError)
	_ = err.WithContext(ctx)

	output := err.GetErrorContext()
	if output == nil {
		t.Fatal("Expected context to be set")
	}

	if output.Type.String() != typeInternalStr {
		t.Errorf("Expected type %q, got %q", typeInternalStr, output.Type)
	}

	if output.Severity.String() != severityCriticalStr {
		t.Errorf("Expected severity %q, got %q", severityCriticalStr, output.Severity)
	}

	if output.RequestID != "req-123" {
		t.Errorf("Expected request_id 'req-123', got %q", output.RequestID)
	}
}

func TestToErrorOutputWithCause(t *testing.T) {
	t.Parallel()

	rootErr := New(msgRoot)
	wrappedErr := Wrap(rootErr, msgWrapped)

	output := wrappedErr.toErrorOutput()

	if output.Cause == nil {
		t.Fatal("Expected cause to be set")
	}

	if output.Cause.Message != msgRoot {
		t.Errorf("Expected cause message %q, got %q", msgRoot, output.Cause.Message)
	}
}

func TestToErrorOutputWithStandardError(t *testing.T) {
	t.Parallel()

	wrappedErr := Wrap(errStandard, "wrapped standard error")

	output := wrappedErr.toErrorOutput()

	if output.Cause == nil {
		t.Fatal("Expected cause to be set")
	}

	if output.Cause.Message != msgStandardErr {
		t.Errorf("Expected cause message %q, got %q", msgStandardErr, output.Cause.Message)
	}

	if output.Cause.Type != typeUnknownStr {
		t.Errorf("Expected cause type %q, got %q", typeUnknownStr, output.Cause.Type)
	}
}

func TestToErrorOutputWithOptions(t *testing.T) {
	t.Parallel()

	err := New(msgTestError)

	output := err.toErrorOutput(
		WithStackTrace(false),
		WithTimestampFormat(dateOnlyLayout),
	)

	if output.Stack != "" {
		t.Error("Expected stack trace to be empty")
	}

	_, parseErr := time.Parse(dateOnlyLayout, output.Timestamp)
	if parseErr != nil {
		t.Errorf("Expected timestamp in format 2006-01-02, got %s", output.Timestamp)
	}
}

func TestToJSON(t *testing.T) {
	t.Parallel()

	err := New(msgTestError)

	jsonStr, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Fatalf(unexpectedErrFn, jsonErr)
	}

	if jsonStr == "" {
		t.Error("Expected non-empty JSON string")
	}

	var output ErrorOutput

	unmarshalErr := json.Unmarshal([]byte(jsonStr), &output)
	if unmarshalErr != nil {
		t.Errorf("Failed to unmarshal JSON: %v", unmarshalErr)
	}

	if output.Message != msgTestError {
		t.Errorf("Expected message %q, got %q", msgTestError, output.Message)
	}
}

func TestToJSONWithOptions(t *testing.T) {
	t.Parallel()

	err := New(msgTestError)

	jsonStr, jsonErr := err.ToJSON(WithStackTrace(false))
	if jsonErr != nil {
		t.Fatalf(unexpectedErrFn, jsonErr)
	}

	t.Logf("err: %v", jsonStr)

	if strings.Contains(jsonStr, "stack") && !strings.Contains(jsonStr, `"stack": ""`) {
		t.Error("Expected stack trace to be excluded or empty")
	}
}

func TestToYAML(t *testing.T) {
	t.Parallel()

	err := New(msgTestError)

	yamlStr, yamlErr := err.ToYAML()
	if yamlErr != nil {
		t.Fatalf(unexpectedErrFn, yamlErr)
	}

	if yamlStr == "" {
		t.Error("Expected non-empty YAML string")
	}

	var output ErrorOutput

	unmarshalErr := yaml.Unmarshal([]byte(yamlStr), &output)
	if unmarshalErr != nil {
		t.Errorf("Failed to unmarshal YAML: %v", unmarshalErr)
	}

	if output.Message != msgTestError {
		t.Errorf("Expected message %q, got %q", msgTestError, output.Message)
	}
}

func TestToYAMLWithOptions(t *testing.T) {
	t.Parallel()

	err := New(msgTestError)

	yamlStr, yamlErr := err.ToYAML(WithStackTrace(false))
	if yamlErr != nil {
		t.Fatalf(unexpectedErrFn, yamlErr)
	}

	if strings.Contains(yamlStr, "stack:") && !strings.Contains(yamlStr, "stack: \"\"") {
		t.Error("Expected stack trace to be excluded or empty")
	}
}

func TestToErrorOutputWithMetadata(t *testing.T) {
	t.Parallel()

	err := New(msgTestError)
	_ = err.WithMetadata("custom_field", "custom_value")
	_ = err.WithMetadata("another_field", formatTestLine)

	output := err.toErrorOutput()

	if output.Metadata["custom_field"] != "custom_value" {
		t.Errorf("Expected custom_field 'custom_value', got %v", output.Metadata["custom_field"])
	}

	if output.Metadata["another_field"] != formatTestLine {
		t.Errorf("Expected another_field %d, got %v", formatTestLine, output.Metadata["another_field"])
	}

	if _, exists := output.Metadata["error_context"]; exists {
		t.Error("Expected error_context to be excluded from metadata")
	}
}

func TestToErrorOutputWithRecoverySuggestion(t *testing.T) {
	t.Parallel()

	rs := &RecoverySuggestion{
		Message:       "restart service",
		Actions:       []string{"restart"},
		Documentation: "https://example.com/recover",
	}

	err := New(msgTestError, WithRecoverySuggestion(rs))
	output := err.toErrorOutput()

	if output.Recovery == nil {
		t.Fatal("expected recovery suggestion to be present")
	}

	if output.Recovery.Message != rs.Message {
		t.Errorf("expected recovery message %q, got %q", rs.Message, output.Recovery.Message)
	}

	if output.Recovery.Documentation != rs.Documentation {
		t.Errorf("expected documentation %q, got %q", rs.Documentation, output.Recovery.Documentation)
	}
}
