package dto

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// Error messages returned by the HTTP handlers.
const (
	MessageInternalError      = "internal error"
	MessageNotFound           = "not found"
	MessageInvalidRequestBody = "invalid request body"
	MessageValidationFailed   = "validation failed"
)

// FieldError describes a single request field that failed validation.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ErrorResponse represents an error payload returned by the HTTP handlers.
type ErrorResponse struct {
	Error  string       `json:"error"`
	Fields []FieldError `json:"fields,omitempty"`
}

// NewErrorResponse creates a new ErrorResponse with the given message.
func NewErrorResponse(message string) ErrorResponse {
	return ErrorResponse{Error: message, Fields: nil}
}

// NewRequestErrorResponse converts a request binding error into an ErrorResponse,
// listing the offending fields when the payload failed validation and falling
// back to a generic message when it could not be decoded at all.
func NewRequestErrorResponse(err error) ErrorResponse {
	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		return NewErrorResponse(MessageInvalidRequestBody)
	}

	fields := make([]FieldError, 0, len(validationErrs))
	for _, fieldErr := range validationErrs {
		fields = append(fields, FieldError{
			Field:   fieldErr.Field(),
			Message: validationMessage(fieldErr),
		})
	}

	return ErrorResponse{Error: MessageValidationFailed, Fields: fields}
}

// validationMessage renders a human-readable message for a failed validation rule.
func validationMessage(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "is required"
	case "uuid4":
		return "must be a valid UUID"
	case "min":
		return fmt.Sprintf("must be at least %s characters long", fieldErr.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters long", fieldErr.Param())
	default:
		return fmt.Sprintf("must satisfy the %q rule", fieldErr.Tag())
	}
}
