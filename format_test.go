package ewrap

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"gopkg.in/yaml.v3"
)

func TestWithTimestampFormat(t *testing.T) {
	tFixed := time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC)
	output := &ErrorOutput{
		Timestamp: tFixed.Format(time.RFC3339),
	}

	// Test with non-empty format
	opt := WithTimestampFormat("2006-01-02")
	opt(output)

	expected := tFixed.Format("2006-01-02")
	if output.Timestamp != expected {
		t.Errorf("Expected timestamp %s, got %s", expected, output.Timestamp)
	}

	// Test with empty format
	output.Timestamp = tFixed.Format(time.RFC3339)
	opt = WithTimestampFormat("")
	opt(output)

	if output.Timestamp != tFixed.Format(time.RFC3339) {
		t.Error("Expected timestamp to remain unchanged with empty format")
	}
}

func TestWithStackTrace(t *testing.T) {
	output := &ErrorOutput{
		Stack: "some stack trace",
	}

	// Test excluding stack trace
	opt := WithStackTrace(false)
	opt(output)

	if output.Stack != "" {
		t.Error("Expected stack trace to be empty when excluded")
	}

	// Test including stack trace
	output.Stack = "some stack trace"
	opt = WithStackTrace(true)
	opt(output)

	if output.Stack != "some stack trace" {
		t.Error("Expected stack trace to remain when included")
	}
}

func TestToErrorOutput(t *testing.T) {
	// Create a basic error
	err := New("test error")

	output := err.toErrorOutput()

	if output.Message != "test error" {
		t.Errorf("Expected message 'test error', got '%s'", output.Message)
	}

	if output.Type != "unknown" {
		t.Errorf("Expected type 'unknown', got '%s'", output.Type)
	}

	if output.Severity != "error" {
		t.Errorf("Expected severity 'error', got '%s'", output.Severity)
	}

	if output.Metadata == nil {
		t.Error("Expected metadata to be initialized")
	}
}

func TestToErrorOutputWithContext(t *testing.T) {
	ctx := &ErrorContext{
		RequestID:   "req-123",
		User:        "testuser",
		Component:   "test-component",
		Operation:   "test-op",
		File:        "test.go",
		Line:        42,
		Environment: "test",
		Type:        ErrorTypeInternal,
		Severity:    SeverityCritical,
	}

	err := New("test error")
	err.WithContext(ctx)

	output := err.GetErrorContext()

	if output.Type.String() != "internal" {
		t.Errorf("Expected type 'internal', got '%s'", output.Type)
	}

	if output.Severity.String() != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", output.Severity)
	}

	if output == nil {
		t.Fatal("Expected context to be set")
	}

	if output.RequestID != "req-123" {
		t.Errorf("Expected request_id 'req-123', got '%v'", output.RequestID)
	}
}

func TestToErrorOutputWithCause(t *testing.T) {
	rootErr := New("root error")
	wrappedErr := Wrap(rootErr, "wrapped error")

	output := wrappedErr.toErrorOutput()

	if output.Cause == nil {
		t.Fatal("Expected cause to be set")
	}

	if output.Cause.Message != "root error" {
		t.Errorf("Expected cause message 'root error', got '%s'", output.Cause.Message)
	}
}

func TestToErrorOutputWithStandardError(t *testing.T) {
	stdErr := errors.New("standard error")
	wrappedErr := Wrap(stdErr, "wrapped standard error")

	output := wrappedErr.toErrorOutput()

	if output.Cause == nil {
		t.Fatal("Expected cause to be set")
	}

	if output.Cause.Message != "standard error" {
		t.Errorf("Expected cause message 'standard error', got '%s'", output.Cause.Message)
	}

	if output.Cause.Type != "unknown" {
		t.Errorf("Expected cause type 'unknown', got '%s'", output.Cause.Type)
	}
}

func TestToErrorOutputWithOptions(t *testing.T) {
	err := New("test error")

	output := err.toErrorOutput(
		WithStackTrace(false),
		WithTimestampFormat("2006-01-02"),
	)

	if output.Stack != "" {
		t.Error("Expected stack trace to be empty")
	}

	if _, err := time.Parse("2006-01-02", output.Timestamp); err != nil {
		t.Errorf("Expected timestamp in format 2006-01-02, got %s", output.Timestamp)
	}
}

func TestToJSON(t *testing.T) {
	err := New("test error")

	jsonStr, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Fatalf("Unexpected error: %v", jsonErr)
	}

	if jsonStr == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Verify it's valid JSON
	var output ErrorOutput
	unmarshalErr := json.Unmarshal([]byte(jsonStr), &output)
	if unmarshalErr != nil {
		t.Errorf("Failed to unmarshal JSON: %v", unmarshalErr)
	}

	if output.Message != "test error" {
		t.Errorf("Expected message 'test error', got '%s'", output.Message)
	}
}

func TestToJSONWithOptions(t *testing.T) {
	err := New("test error")

	jsonStr, jsonErr := err.ToJSON(WithStackTrace(false))
	if jsonErr != nil {
		t.Fatalf("Unexpected error: %v", jsonErr)
	}

	t.Logf("err: %v", jsonStr)

	// Verify stack trace is not included
	if strings.Contains(jsonStr, "stack") && !strings.Contains(jsonStr, `"stack": ""`) {
		t.Error("Expected stack trace to be excluded or empty")
	}
}

func TestToYAML(t *testing.T) {
	err := New("test error")

	yamlStr, yamlErr := err.ToYAML()
	if yamlErr != nil {
		t.Fatalf("Unexpected error: %v", yamlErr)
	}

	if yamlStr == "" {
		t.Error("Expected non-empty YAML string")
	}

	// Verify it's valid YAML
	var output ErrorOutput
	unmarshalErr := yaml.Unmarshal([]byte(yamlStr), &output)
	if unmarshalErr != nil {
		t.Errorf("Failed to unmarshal YAML: %v", unmarshalErr)
	}

	if output.Message != "test error" {
		t.Errorf("Expected message 'test error', got '%s'", output.Message)
	}
}

func TestToYAMLWithOptions(t *testing.T) {
	err := New("test error")

	yamlStr, yamlErr := err.ToYAML(WithStackTrace(false))
	if yamlErr != nil {
		t.Fatalf("Unexpected error: %v", yamlErr)
	}

	// Verify stack trace is not included
	if strings.Contains(yamlStr, "stack:") && !strings.Contains(yamlStr, "stack: \"\"") {
		t.Error("Expected stack trace to be excluded or empty")
	}
}

func TestToErrorOutputWithMetadata(t *testing.T) {
	err := New("test error")
	err.WithMetadata("custom_field", "custom_value")
	err.WithMetadata("another_field", 42)

	output := err.toErrorOutput()

	if output.Metadata["custom_field"] != "custom_value" {
		t.Errorf("Expected custom_field 'custom_value', got '%v'", output.Metadata["custom_field"])
	}

	if output.Metadata["another_field"] != 42 {
		t.Errorf("Expected another_field 42, got '%v'", output.Metadata["another_field"])
	}

	// Ensure error_context is not included in metadata
	if _, exists := output.Metadata["error_context"]; exists {
		t.Error("Expected error_context to be excluded from metadata")
	}
}

func TestToErrorOutputWithRecoverySuggestion(t *testing.T) {
	rs := &RecoverySuggestion{
		Message:       "restart service",
		Actions:       []string{"restart"},
		Documentation: "https://example.com/recover",
	}

	err := New("test error", WithRecoverySuggestion(rs))
	output := err.toErrorOutput()

	if output.Recovery == nil {
		t.Fatal("expected recovery suggestion to be present")
	}

	if output.Recovery.Message != rs.Message {
		t.Errorf("expected recovery message '%s', got '%s'", rs.Message, output.Recovery.Message)
	}

	if output.Recovery.Documentation != rs.Documentation {
		t.Errorf("expected documentation '%s', got '%s'", rs.Documentation, output.Recovery.Documentation)
	}
}
