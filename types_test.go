package ewrap

import "testing"

func TestErrorType_String(t *testing.T) {
	tests := []struct {
		name     string
		et       ErrorType
		expected string
	}{
		{"Unknown", ErrorTypeUnknown, "unknown"},
		{"Validation", ErrorTypeValidation, "validation"},
		{"NotFound", ErrorTypeNotFound, "not_found"},
		{"Permission", ErrorTypePermission, "permission"},
		{"Database", ErrorTypeDatabase, "database"},
		{"Network", ErrorTypeNetwork, "network"},
		{"Configuration", ErrorTypeConfiguration, "configuration"},
		{"Internal", ErrorTypeInternal, "internal"},
		{"External", ErrorTypeExternal, "external"},
		{"Invalid", ErrorType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.et.String(); got != tt.expected {
				t.Errorf("ErrorType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		name     string
		s        Severity
		expected string
	}{
		{"Info", SeverityInfo, "info"},
		{"Warning", SeverityWarning, "warning"},
		{"Error", SeverityError, "error"},
		{"Critical", SeverityCritical, "critical"},
		{"Invalid", Severity(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.expected {
				t.Errorf("Severity.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorTypeConstants(t *testing.T) {
	// Test that constants have expected values
	if ErrorTypeUnknown != 0 {
		t.Errorf("ErrorTypeUnknown = %d, want 0", ErrorTypeUnknown)
	}
	if ErrorTypeValidation != 1 {
		t.Errorf("ErrorTypeValidation = %d, want 1", ErrorTypeValidation)
	}
	if ErrorTypeExternal != 8 {
		t.Errorf("ErrorTypeExternal = %d, want 8", ErrorTypeExternal)
	}
}

func TestSeverityConstants(t *testing.T) {
	// Test that constants have expected values
	if SeverityInfo != 0 {
		t.Errorf("SeverityInfo = %d, want 0", SeverityInfo)
	}
	if SeverityWarning != 1 {
		t.Errorf("SeverityWarning = %d, want 1", SeverityWarning)
	}
	if SeverityCritical != 3 {
		t.Errorf("SeverityCritical = %d, want 3", SeverityCritical)
	}
}

func TestRecoverySuggestion(t *testing.T) {
	rs := RecoverySuggestion{
		Message:       "Test message",
		Actions:       []string{"action1", "action2"},
		Documentation: "https://example.com/docs",
	}

	if rs.Message != "Test message" {
		t.Errorf("RecoverySuggestion.Message = %v, want %v", rs.Message, "Test message")
	}
	if len(rs.Actions) != 2 {
		t.Errorf("len(RecoverySuggestion.Actions) = %v, want %v", len(rs.Actions), 2)
	}
	if rs.Documentation != "https://example.com/docs" {
		t.Errorf("RecoverySuggestion.Documentation = %v, want %v", rs.Documentation, "https://example.com/docs")
	}
}
