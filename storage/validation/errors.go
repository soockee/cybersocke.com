package validation

import "fmt"

// ValidationError represents validation failures with categorized error codes.
// Kept simple—no complex wrapping, stays within validation package.
type ValidationError struct {
	Code    ErrorCode
	Message string
	Field   string // optional field name context
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s (field: %s)", e.Code, e.Message, e.Field)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ErrorCode categorizes validation failures for simple downstream handling.
type ErrorCode string

const (
	ErrCodeFrontmatter  ErrorCode = "FRONTMATTER_INVALID"
	ErrCodeMissingField ErrorCode = "MISSING_FIELD"
	ErrCodeInvalidField ErrorCode = "INVALID_FIELD"
	ErrCodeBadTag       ErrorCode = "BAD_TAG"
	ErrCodeDuplicateTag ErrorCode = "DUPLICATE_TAG"
	ErrCodeContract     ErrorCode = "CONTRACT_ERROR"
	ErrCodeSlug         ErrorCode = "SLUG_INVALID"
	ErrCodeTimestamp    ErrorCode = "TIMESTAMP_INVALID"
)

// NewValidationError constructs a simple validation error.
func NewValidationError(code ErrorCode, message string, field string) *ValidationError {
	return &ValidationError{Code: code, Message: message, Field: field}
}

// IsValidationError checks if an error is a ValidationError.
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}
