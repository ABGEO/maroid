package handler

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/abgeo/maroid/libs/problem"
)

// requestProblem names a body that the route refuses. A body that does not decode
// is body-invalid, and a body that breaks a rule is validation-failed with one
// item for each field.
func requestProblem(err error) problem.Problem {
	var invalid validator.ValidationErrors
	if !errors.As(err, &invalid) {
		return problem.NewBodyInvalid()
	}

	failures := make([]problem.FieldFailure, 0, len(invalid))
	for _, fieldErr := range invalid {
		failures = append(failures, problem.FieldFailure{
			Detail:  validationDetail(fieldErr),
			Pointer: "#/" + fieldErr.Field(),
		})
	}

	return problem.NewValidationFailed().WithErrors(failures...)
}

// validationDetail renders the reason that one rule gives.
func validationDetail(fieldErr validator.FieldError) string {
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
