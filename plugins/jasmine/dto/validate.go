package dto

import (
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// TagNotAfterTomorrow names the rule that refuses a date after the UTC date of
// tomorrow. A browser east of UTC reads a local date that is one day ahead of the
// UTC date for part of each day, so the rule keeps one day of margin.
const TagNotAfterTomorrow = "not_after_tomorrow"

// validate is the shared validator used by the request Validate methods.
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

	if err := instance.RegisterValidation(TagNotAfterTomorrow, notAfterTomorrow); err != nil {
		panic(err)
	}

	return instance
}

func notAfterTomorrow(field validator.FieldLevel) bool {
	date, err := time.Parse(time.DateOnly, field.Field().String())
	if err != nil {
		return false
	}

	tomorrow := time.Now().UTC().AddDate(0, 0, 1)
	latest := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.UTC)

	return !date.After(latest)
}
