package ewrap

import (
	"errors"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"gopkg.in/yaml.v3"
)

func TestStackIterator(t *testing.T) {
	// Create an error to get a stack trace
	err := New("test error")
	iterator := err.GetStackIterator()

	// Test HasNext and Next
	frameCount := 0
	for iterator.HasNext() {
		frame := iterator.Next()
		if frame == nil {
			t.Error("Expected frame, got nil")
		}
		frameCount++
	}

	if frameCount == 0 {
		t.Error("Expected at least one frame")
	}

	// Test that Next returns nil after iteration is complete
	if iterator.Next() != nil {
		t.Error("Expected nil after iteration complete")
	}

	// Test Reset
	iterator.Reset()
	if !iterator.HasNext() {
		t.Error("Expected frames after reset")
	}

	// Test AllFrames
	allFrames := iterator.AllFrames()
	if len(allFrames) != frameCount {
		t.Errorf("Expected %d frames, got %d", frameCount, len(allFrames))
	}
}

func TestStackFrameStructure(t *testing.T) {
	err := New("test error")
	frames := err.GetStackFrames()

	if len(frames) == 0 {
		t.Error("Expected at least one frame")
	}

	frame := frames[0]
	if frame.Function == "" {
		t.Error("Expected function name")
	}
	if frame.File == "" {
		t.Error("Expected file name")
	}
	if frame.Line == 0 {
		t.Error("Expected line number")
	}
	if frame.PC == 0 {
		t.Error("Expected program counter")
	}
}

func TestErrorGroupSerialization(t *testing.T) {
	eg := NewErrorGroup()

	// Add different types of errors
	eg.Add(New("ewrap error").WithMetadata("key", "value"))
	eg.Add(New("another error"))

	// Test ToSerialization
	serializable := eg.ToSerialization()

	if serializable.ErrorCount != 2 {
		t.Errorf("Expected 2 errors, got %d", serializable.ErrorCount)
	}

	if len(serializable.Errors) != 2 {
		t.Errorf("Expected 2 serialized errors, got %d", len(serializable.Errors))
	}

	// Check first error
	firstErr := serializable.Errors[0]
	if firstErr.Type != "ewrap" {
		t.Errorf("Expected type 'ewrap', got '%s'", firstErr.Type)
	}

	if firstErr.Message != "ewrap error" {
		t.Errorf("Expected message 'ewrap error', got '%s'", firstErr.Message)
	}

	if firstErr.Metadata == nil || firstErr.Metadata["key"] != "value" {
		t.Error("Expected metadata to be preserved")
	}

	if len(firstErr.StackTrace) == 0 {
		t.Error("Expected stack trace in serialized error")
	}
}

func TestErrorGroupJSON(t *testing.T) {
	eg := NewErrorGroup()
	eg.Add(New("test error"))

	// Test ToJSON
	jsonStr, err := eg.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	if jsonStr == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Verify it's valid JSON by unmarshaling
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	// Test MarshalJSON interface
	jsonBytes, err := json.Marshal(eg)
	if err != nil {
		t.Fatalf("Failed to marshal using json.Marshal: %v", err)
	}

	if len(jsonBytes) == 0 {
		t.Error("Expected non-empty JSON bytes")
	}
}

func TestErrorGroupYAML(t *testing.T) {
	eg := NewErrorGroup()
	eg.Add(New("test error"))

	// Test ToYAML
	yamlStr, err := eg.ToYAML()
	if err != nil {
		t.Fatalf("Failed to convert to YAML: %v", err)
	}

	if yamlStr == "" {
		t.Error("Expected non-empty YAML string")
	}

	// Verify it's valid YAML by unmarshaling
	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &result); err != nil {
		t.Errorf("Failed to unmarshal YAML: %v", err)
	}

	// Test MarshalYAML interface
	yamlData, err := yaml.Marshal(eg)
	if err != nil {
		t.Fatalf("Failed to marshal using yaml.Marshal: %v", err)
	}

	if len(yamlData) == 0 {
		t.Error("Expected non-empty YAML bytes")
	}
}

func TestErrorGroupSerializationWithWrappedErrors(t *testing.T) {
	eg := NewErrorGroup()

	// Create a chain of wrapped errors
	rootErr := New("root cause")
	wrappedErr := Wrap(rootErr, "wrapped error")
	eg.Add(wrappedErr)

	serializable := eg.ToSerialization()

	if len(serializable.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(serializable.Errors))
	}

	err := serializable.Errors[0]
	if err.Message != "wrapped error: root cause" {
		t.Errorf("Expected wrapped message, got '%s'", err.Message)
	}

	if err.Cause == nil {
		t.Error("Expected cause to be serialized")
	}

	if err.Cause.Message != "root cause" {
		t.Errorf("Expected cause message 'root cause', got '%s'", err.Cause.Message)
	}
}

func TestErrorGroupSerializationWithStandardErrors(t *testing.T) {
	eg := NewErrorGroup()

	// Add standard Go error
	eg.Add(errors.New("standard error"))

	serializable := eg.ToSerialization()

	if len(serializable.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(serializable.Errors))
	}

	err := serializable.Errors[0]
	if err.Type != "standard" {
		t.Errorf("Expected type 'standard', got '%s'", err.Type)
	}

	if len(err.StackTrace) != 0 {
		t.Error("Expected no stack trace for standard error")
	}

	if err.Metadata != nil {
		t.Error("Expected no metadata for standard error")
	}
}

func TestEmptyErrorGroupSerialization(t *testing.T) {
	eg := NewErrorGroup()

	serializable := eg.ToSerialization()

	if serializable.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", serializable.ErrorCount)
	}

	if len(serializable.Errors) != 0 {
		t.Errorf("Expected 0 serialized errors, got %d", len(serializable.Errors))
	}

	// Test JSON serialization of empty group
	jsonStr, err := eg.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize empty group to JSON: %v", err)
	}

	if !strings.Contains(jsonStr, `"error_count": 0`) {
		t.Error("Expected error_count: 0 in JSON")
	}
}

func BenchmarkErrorGroupSerialization(b *testing.B) {
	eg := NewErrorGroup()
	for i := 0; i < 10; i++ {
		eg.Add(New("error").WithMetadata("index", i))
	}

	b.Run("JSON", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := eg.ToJSON()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("YAML", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := eg.ToYAML()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
