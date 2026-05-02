package ewrap

import (
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"gopkg.in/yaml.v3"
)

func TestStackIterator(t *testing.T) {
	t.Parallel()

	err := New(msgTestError)
	iterator := err.GetStackIterator()

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

	if iterator.Next() != nil {
		t.Error("Expected nil after iteration complete")
	}

	iterator.Reset()

	if !iterator.HasNext() {
		t.Error("Expected frames after reset")
	}

	allFrames := iterator.AllFrames()
	if len(allFrames) != frameCount {
		t.Errorf("Expected %d frames, got %d", frameCount, len(allFrames))
	}
}

func TestStackFrameStructure(t *testing.T) {
	t.Parallel()

	err := New(msgTestError)
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
	t.Parallel()

	eg := NewErrorGroup()

	eg.Add(New("ewrap error").WithMetadata(msgKey, msgValue))
	eg.Add(New("another error"))

	serializable := eg.ToSerialization()

	const wantCount = 2

	if serializable.ErrorCount != wantCount {
		t.Errorf("Expected %d errors, got %d", wantCount, serializable.ErrorCount)
	}

	if len(serializable.Errors) != wantCount {
		t.Errorf("Expected %d serialized errors, got %d", wantCount, len(serializable.Errors))
	}

	firstErr := serializable.Errors[0]
	if firstErr.Type != "ewrap" {
		t.Errorf("Expected type 'ewrap', got %q", firstErr.Type)
	}

	if firstErr.Message != "ewrap error" {
		t.Errorf("Expected message 'ewrap error', got %q", firstErr.Message)
	}

	if firstErr.Metadata == nil || firstErr.Metadata[msgKey] != msgValue {
		t.Error("Expected metadata to be preserved")
	}

	if len(firstErr.StackTrace) == 0 {
		t.Error("Expected stack trace in serialized error")
	}
}

func TestErrorGroupJSON(t *testing.T) {
	t.Parallel()

	eg := NewErrorGroup()
	eg.Add(New(msgTestError))

	jsonStr, err := eg.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	if jsonStr == "" {
		t.Error("Expected non-empty JSON string")
	}

	var result map[string]any

	unmarshalErr := json.Unmarshal([]byte(jsonStr), &result)
	if unmarshalErr != nil {
		t.Errorf("Failed to unmarshal JSON: %v", unmarshalErr)
	}

	jsonBytes, marshalErr := json.Marshal(eg)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal using json.Marshal: %v", marshalErr)
	}

	if len(jsonBytes) == 0 {
		t.Error("Expected non-empty JSON bytes")
	}
}

func TestErrorGroupYAML(t *testing.T) {
	t.Parallel()

	eg := NewErrorGroup()
	eg.Add(New(msgTestError))

	yamlStr, err := eg.ToYAML()
	if err != nil {
		t.Fatalf("Failed to convert to YAML: %v", err)
	}

	if yamlStr == "" {
		t.Error("Expected non-empty YAML string")
	}

	var result map[string]any

	unmarshalErr := yaml.Unmarshal([]byte(yamlStr), &result)
	if unmarshalErr != nil {
		t.Errorf("Failed to unmarshal YAML: %v", unmarshalErr)
	}

	yamlData, marshalErr := yaml.Marshal(eg)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal using yaml.Marshal: %v", marshalErr)
	}

	if len(yamlData) == 0 {
		t.Error("Expected non-empty YAML bytes")
	}
}

func TestErrorGroupSerializationWithWrappedErrors(t *testing.T) {
	t.Parallel()

	eg := NewErrorGroup()

	rootErr := New(msgRootCause)
	wrappedErr := Wrap(rootErr, "wrapped error")
	eg.Add(wrappedErr)

	serializable := eg.ToSerialization()

	if len(serializable.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(serializable.Errors))
	}

	const want = "wrapped error: root cause"

	got := serializable.Errors[0]
	if got.Message != want {
		t.Errorf("Expected message %q, got %q", want, got.Message)
	}

	if got.Cause == nil {
		t.Error("Expected cause to be serialized")
	}

	if got.Cause.Message != msgRootCause {
		t.Errorf("Expected cause message %q, got %q", msgRootCause, got.Cause.Message)
	}
}

func TestErrorGroupSerializationWithStandardErrors(t *testing.T) {
	t.Parallel()

	eg := NewErrorGroup()
	eg.Add(errStandard)

	serializable := eg.ToSerialization()

	if len(serializable.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(serializable.Errors))
	}

	got := serializable.Errors[0]
	if got.Type != "standard" {
		t.Errorf("Expected type 'standard', got %q", got.Type)
	}

	if len(got.StackTrace) != 0 {
		t.Error("Expected no stack trace for standard error")
	}

	if got.Metadata != nil {
		t.Error("Expected no metadata for standard error")
	}
}

func TestEmptyErrorGroupSerialization(t *testing.T) {
	t.Parallel()

	eg := NewErrorGroup()

	serializable := eg.ToSerialization()

	if serializable.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", serializable.ErrorCount)
	}

	if len(serializable.Errors) != 0 {
		t.Errorf("Expected 0 serialized errors, got %d", len(serializable.Errors))
	}

	jsonStr, err := eg.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize empty group to JSON: %v", err)
	}

	if !strings.Contains(jsonStr, `"error_count": 0`) {
		t.Error("Expected error_count: 0 in JSON")
	}
}

func BenchmarkErrorGroupSerialization(b *testing.B) {
	const errorCount = 10

	eg := NewErrorGroup()
	for i := range errorCount {
		eg.Add(New("error").WithMetadata("index", i))
	}

	b.Run("JSON", func(b *testing.B) { benchSerializationJSON(b, eg) })
	b.Run("YAML", func(b *testing.B) { benchSerializationYAML(b, eg) })
}

func benchSerializationJSON(b *testing.B, eg *ErrorGroup) {
	b.Helper()
	b.ReportAllocs()

	for range b.N {
		_, err := eg.ToJSON()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchSerializationYAML(b *testing.B, eg *ErrorGroup) {
	b.Helper()
	b.ReportAllocs()

	for range b.N {
		_, err := eg.ToYAML()
		if err != nil {
			b.Fatal(err)
		}
	}
}
