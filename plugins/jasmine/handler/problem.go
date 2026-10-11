package handler

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/jasmine/dto"
	"github.com/abgeo/maroid/plugins/jasmine/model"
)

const typeEnvironmentOccupied = "/problems/dev.maroid.jasmine/environment-occupied"

// indexToPointer turns the name that the validator gives an item of a list,
// plant_ids[3], into the segments of a JSON pointer, plant_ids/3.
//
//nolint:gochecknoglobals // shared, immutable replacer
var indexToPointer = strings.NewReplacer("[", "/", "]", "")

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
			Pointer: "#/" + indexToPointer.Replace(fieldErr.Field()),
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
	case "oneof":
		return "must be one of: " + fieldErr.Param()
	case "min":
		return boundDetail("at least", fieldErr)
	case "max":
		return boundDetail("at most", fieldErr)
	case dto.TagMaintenanceType:
		return "must be one of: " + maintenanceTypeList()
	default:
		return fixedDetail(fieldErr.Tag())
	}
}

// fixedDetail renders the reason of a rule that takes no parameter.
func fixedDetail(tag string) string {
	switch tag {
	case "required":
		return "is required"
	case "uuid":
		return "must be a valid UUID"
	case "unique":
		return "must not hold an item two times"
	case dto.TagNotAfterTomorrow:
		return "must be a date in the form YYYY-MM-DD, not after tomorrow in UTC"
	case dto.TagNotFuture:
		return "must be a date-time in RFC 3339 form, not after the current time"
	default:
		return fmt.Sprintf("must satisfy the %q rule", tag)
	}
}

// boundDetail renders a lower or an upper bound in the unit of the field: the
// items of a list, the characters of a text, or the value of a number.
func boundDetail(relation string, fieldErr validator.FieldError) string {
	kind := fieldErr.Kind()
	if kind == reflect.Slice {
		return fmt.Sprintf("must hold %s %s items", relation, fieldErr.Param())
	}

	if kind == reflect.String {
		return fmt.Sprintf("must be %s %s characters long", relation, fieldErr.Param())
	}

	return fmt.Sprintf("must be %s %s", relation, fieldErr.Param())
}

func maintenanceTypeList() string {
	types := model.MaintenanceTypes()

	names := make([]string, 0, len(types))
	for _, kind := range types {
		names = append(names, string(kind))
	}

	return strings.Join(names, " ")
}
