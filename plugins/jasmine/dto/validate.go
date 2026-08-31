package dto

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// validate is the shared validator used by the request Bind methods.
// validator.Validate caches struct metadata and is safe for concurrent use,
// so one instance is reused across requests instead of built per call.
//
//nolint:gochecknoglobals // shared, immutable validator instance
var validate = newValidator()

// newValidator builds the shared validator, reporting field names as they appear
// in JSON so that validation errors line up with the submitted payload.
func newValidator() *validator.Validate {
	instance := validator.New()

	instance.RegisterTagNameFunc(func(field reflect.StructField) string {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}

		return name
	})

	return instance
}
