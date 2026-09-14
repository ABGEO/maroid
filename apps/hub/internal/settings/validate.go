package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	jsonvalidate "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

// The reason that each keyword reports to the person who filled the form.
const (
	reasonUnknown  = "the field is unknown"
	reasonType     = "the value has the wrong type"
	reasonEnum     = "the value is not in the list"
	reasonTooLong  = "the value is too long"
	reasonRequired = "the field is required"
	reasonRefused  = "the value is not permitted"
)

// InvalidError reports that a save does not match the settings schema.
type InvalidError struct {
	Fields map[string]string
}

// Error returns the reason of the rejection.
func (e *InvalidError) Error() string {
	keys := make([]string, 0, len(e.Fields))
	for key := range e.Fields {
		keys = append(keys, key)
	}

	return "the settings do not match the schema: " + strings.Join(keys, ", ")
}

// Validate reports each field of the input that the settings schema refuses.
func Validate(schema *Schema, input map[string]any) error {
	instance, err := asInstance(schema, input)
	if err != nil {
		return err
	}

	if err = schema.Compiled.Validate(instance); err != nil {
		var invalid *jsonvalidate.ValidationError
		if !errors.As(err, &invalid) {
			return fmt.Errorf("validating the settings: %w", err)
		}

		fields := make(map[string]string)
		collect(invalid, fields)

		return &InvalidError{Fields: fields}
	}

	return nil
}

// asInstance drops every field that the input removes, because a removal carries no
// value for the schema to judge. See PSET-FR-007. It drops a secret that carries the
// mask for the same reason: that field keeps the value the row holds, and the mask is
// not that value, so the rules of the field never judge it. See PSET-DD-010.
func asInstance(schema *Schema, input map[string]any) (any, error) {
	given := make(map[string]any, len(input))

	for key, value := range input {
		if value == nil || keepsSecret(schema.Kinds[key], value) {
			continue
		}

		given[key] = value
	}

	encoded, err := json.Marshal(given)
	if err != nil {
		return nil, fmt.Errorf("encoding the settings input: %w", err)
	}

	var instance any

	if err = json.Unmarshal(encoded, &instance); err != nil {
		return nil, fmt.Errorf("decoding the settings input: %w", err)
	}

	return instance, nil
}

// collect walks the causes and names the field of each leaf.
func collect(failure *jsonvalidate.ValidationError, fields map[string]string) {
	if len(failure.Causes) == 0 {
		record(failure, fields)

		return
	}

	for _, cause := range failure.Causes {
		collect(cause, fields)
	}
}

// record names the field of one leaf failure. One additionalProperties failure can
// name more than one property, so it adds every one.
func record(failure *jsonvalidate.ValidationError, fields map[string]string) {
	if unknown, ok := failure.ErrorKind.(*kind.AdditionalProperties); ok {
		for _, property := range unknown.Properties {
			fields[property] = reasonUnknown
		}

		return
	}

	field := lastOf(failure.InstanceLocation)
	if field == "" {
		return
	}

	fields[field] = reasonOf(lastOf(failure.ErrorKind.KeywordPath()))
}

func reasonOf(keyword string) string {
	switch keyword {
	case "type":
		return reasonType
	case "enum":
		return reasonEnum
	case "maxLength":
		return reasonTooLong
	default:
		return reasonRefused
	}
}

func lastOf(path []string) string {
	if len(path) == 0 {
		return ""
	}

	return path[len(path)-1]
}
