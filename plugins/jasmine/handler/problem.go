package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/jasmine/dto"
)

const typeEnvironmentOccupied = "/problems/dev.maroid.jasmine/environment-occupied"

// writeRequestProblem answers a body that the route refuses. A body that does not
// decode is body-invalid, and a body that breaks a rule is validation-failed with
// one item for each field.
func writeRequestProblem(w http.ResponseWriter, r *http.Request, err error) {
	var invalid validator.ValidationErrors
	if !errors.As(err, &invalid) {
		problem.Write(w, r, problem.NewBodyInvalid())

		return
	}

	failures := make([]problem.FieldFailure, 0, len(invalid))
	for _, fieldErr := range invalid {
		failures = append(failures, problem.FieldFailure{
			Detail:  validationDetail(fieldErr),
			Pointer: "#/" + fieldErr.Field(),
		})
	}

	problem.Write(w, r, problem.NewValidationFailed(failures...))
}

// writeFieldProblem answers a body whose one field names a record that the
// acting workspace does not hold.
func writeFieldProblem(w http.ResponseWriter, r *http.Request, field string, detail string) {
	problem.Write(w, r, problem.NewValidationFailed(problem.FieldFailure{
		Detail:  detail,
		Pointer: "#/" + field,
	}))
}

// writeEnvironmentOccupied answers the delete of an environment that holds a plant.
// A count of zero means that a plant arrived after the count, so the detail names
// no number.
func writeEnvironmentOccupied(w http.ResponseWriter, r *http.Request, plants int) {
	detail := "The environment holds a plant."

	switch {
	case plants == 1:
		detail = "The environment holds 1 plant."
	case plants > 1:
		detail = fmt.Sprintf("The environment holds %d plants.", plants)
	}

	problem.Write(w, r, &problem.Problem{
		Type:   typeEnvironmentOccupied,
		Title:  "The environment holds a plant.",
		Status: http.StatusConflict,
		Detail: detail,
	})
}

// validationDetail renders the reason that one rule gives.
func validationDetail(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "is required"
	case "uuid":
		return "must be a valid UUID"
	case "oneof":
		return "must be one of: " + fieldErr.Param()
	case "min":
		return fmt.Sprintf("must be at least %s characters long", fieldErr.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters long", fieldErr.Param())
	case dto.TagNotAfterTomorrow:
		return "must be a date in the form YYYY-MM-DD, not after tomorrow in UTC"
	default:
		return fmt.Sprintf("must satisfy the %q rule", fieldErr.Tag())
	}
}
