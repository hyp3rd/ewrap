package ewrap

import "testing"

const (
	invalidEnumValue       = 999
	errorTypeExternalValue = 8
	severityCriticalValue  = 3
)

func TestErrorType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		et       ErrorType
		expected string
	}{
		{"Unknown", ErrorTypeUnknown, typeUnknownStr},
		{"Validation", ErrorTypeValidation, typeValidationStr},
		{"NotFound", ErrorTypeNotFound, typeNotFoundStr},
		{"Permission", ErrorTypePermission, typePermissionStr},
		{"Database", ErrorTypeDatabase, typeDatabaseStr},
		{"Network", ErrorTypeNetwork, typeNetworkStr},
		{"Configuration", ErrorTypeConfiguration, typeConfigurationStr},
		{"Internal", ErrorTypeInternal, typeInternalStr},
		{"External", ErrorTypeExternal, typeExternalStr},
		{"Invalid", ErrorType(invalidEnumValue), typeUnknownStr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.et.String(); got != tt.expected {
				t.Errorf("ErrorType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSeverity_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		s        Severity
		expected string
	}{
		{"Info", SeverityInfo, severityInfoStr},
		{"Warning", SeverityWarning, severityWarningStr},
		{"Error", SeverityError, severityErrorStr},
		{"Critical", SeverityCritical, severityCriticalStr},
		{"Invalid", Severity(invalidEnumValue), typeUnknownStr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.s.String(); got != tt.expected {
				t.Errorf("Severity.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorTypeConstants(t *testing.T) {
	t.Parallel()

	if ErrorTypeUnknown != 0 {
		t.Errorf("ErrorTypeUnknown = %d, want 0", ErrorTypeUnknown)
	}

	if ErrorTypeValidation != 1 {
		t.Errorf("ErrorTypeValidation = %d, want 1", ErrorTypeValidation)
	}

	if ErrorTypeExternal != errorTypeExternalValue {
		t.Errorf("ErrorTypeExternal = %d, want %d", ErrorTypeExternal, errorTypeExternalValue)
	}
}

func TestSeverityConstants(t *testing.T) {
	t.Parallel()

	if SeverityInfo != 0 {
		t.Errorf("SeverityInfo = %d, want 0", SeverityInfo)
	}

	if SeverityWarning != 1 {
		t.Errorf("SeverityWarning = %d, want 1", SeverityWarning)
	}

	if SeverityCritical != severityCriticalValue {
		t.Errorf("SeverityCritical = %d, want %d", SeverityCritical, severityCriticalValue)
	}
}

func TestRecoverySuggestion(t *testing.T) {
	t.Parallel()

	const wantMessage = "Test message"

	rs := RecoverySuggestion{
		Message:       wantMessage,
		Actions:       []string{"action1", "action2"},
		Documentation: "https://example.com/docs",
	}

	if rs.Message != wantMessage {
		t.Errorf("RecoverySuggestion.Message = %v, want %v", rs.Message, wantMessage)
	}

	const wantActionCount = 2

	if len(rs.Actions) != wantActionCount {
		t.Errorf("len(RecoverySuggestion.Actions) = %v, want %v", len(rs.Actions), wantActionCount)
	}

	if rs.Documentation != "https://example.com/docs" {
		t.Errorf("RecoverySuggestion.Documentation = %v, want %v", rs.Documentation, "https://example.com/docs")
	}
}
