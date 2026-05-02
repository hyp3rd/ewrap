package ewrap

// Canonical string forms for ErrorType and Severity. These are the values
// returned by String() and used in serialized payloads, so they're worth
// pinning as named constants rather than free-floating literals.
const (
	typeUnknownStr       = "unknown"
	typeValidationStr    = "validation"
	typeNotFoundStr      = "not_found"
	typePermissionStr    = "permission"
	typeDatabaseStr      = "database"
	typeNetworkStr       = "network"
	typeConfigurationStr = "configuration"
	typeInternalStr      = "internal"
	typeExternalStr      = "external"

	severityInfoStr     = "info"
	severityWarningStr  = "warning"
	severityErrorStr    = "error"
	severityCriticalStr = "critical"
)

// ErrorType represents the type of error that occurred.
type ErrorType int

const (
	// ErrorTypeUnknown represents an unknown error type.
	ErrorTypeUnknown ErrorType = iota
	// ErrorTypeValidation represents a validation error.
	ErrorTypeValidation
	// ErrorTypeNotFound represents a not found error.
	ErrorTypeNotFound
	// ErrorTypePermission represents a permission error.
	ErrorTypePermission
	// ErrorTypeDatabase represents a database error.
	ErrorTypeDatabase
	// ErrorTypeNetwork represents a network error.
	ErrorTypeNetwork
	// ErrorTypeConfiguration represents a configuration error.
	ErrorTypeConfiguration
	// ErrorTypeInternal indicates internal system errors.
	ErrorTypeInternal
	// ErrorTypeExternal indicates errors from external services.
	ErrorTypeExternal
)

// String returns the string representation of the error type, useful for
// logging and error reporting.
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeValidation:
		return typeValidationStr
	case ErrorTypeNotFound:
		return typeNotFoundStr
	case ErrorTypePermission:
		return typePermissionStr
	case ErrorTypeDatabase:
		return typeDatabaseStr
	case ErrorTypeNetwork:
		return typeNetworkStr
	case ErrorTypeConfiguration:
		return typeConfigurationStr
	case ErrorTypeInternal:
		return typeInternalStr
	case ErrorTypeExternal:
		return typeExternalStr
	case ErrorTypeUnknown:
		fallthrough
	default:
		return typeUnknownStr
	}
}

// Severity represents the impact level of an error.
type Severity int

const (
	// SeverityInfo indicates an informational message.
	SeverityInfo Severity = iota
	// SeverityWarning indicates a warning that needs attention.
	SeverityWarning
	// SeverityError indicates a significant error.
	SeverityError
	// SeverityCritical indicates a critical system failure.
	SeverityCritical
)

// String returns the string representation of the severity level.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return severityInfoStr
	case SeverityWarning:
		return severityWarningStr
	case SeverityError:
		return severityErrorStr
	case SeverityCritical:
		return severityCriticalStr
	default:
		return typeUnknownStr
	}
}

// RecoverySuggestion provides guidance on how to recover from an error.
type RecoverySuggestion struct {
	// Message provides a human-readable explanation.
	Message string `json:"message" yaml:"message"`
	// Actions lists specific steps that can be taken.
	Actions []string `json:"actions" yaml:"actions"`
	// Documentation links to relevant documentation.
	Documentation string `json:"documentation" yaml:"documentation"`
}
