package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	jsonvalidate "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

// pointerEscaper applies the two escapes that RFC 6901 gives a pointer segment.
//
//nolint:gochecknoglobals // strings.NewReplacer is safe to share and costs one build.
var pointerEscaper = strings.NewReplacer("~", "~0", "/", "~1")

// The reason that each keyword reports to the person who filled the form.
const (
	reasonUnknown  = "the field is unknown"
	reasonType     = "the value has the wrong type"
	reasonEnum     = "the value is not in the list"
	reasonTooLong  = "the value is too long"
	reasonRequired = "the field is required"
	reasonRefused  = "the value is not permitted"
)

// FieldFailure names one field of a rejected save. Pointer is the instance
// location of the failure, as a JSON Pointer in the fragment form.
type FieldFailure struct {
	Pointer string
	Detail  string
}

// InvalidError reports that a save does not match the settings schema.
// Fields is sorted by Pointer, so the answer and the log hold one order.
type InvalidError struct {
	Fields []FieldFailure
}

// Error returns the reason of the rejection.
func (e *InvalidError) Error() string {
	pointers := make([]string, 0, len(e.Fields))
	for _, failure := range e.Fields {
		pointers = append(pointers, failure.Pointer)
	}

	return "the settings do not match the schema: " + strings.Join(pointers, ", ")
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

		failures := make(map[string]string)
		collect(invalid, failures)

		return &InvalidError{Fields: sortedFailures(failures)}
	}

	return nil
}

// asInstance drops every field that the input removes, because a removal carries no
// value for the schema to judge. It drops a secret that carries the
// mask for the same reason: that field keeps the value the row holds, and the mask is
// not that value, so the rules of the field never judge it.
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
			fields[Pointer(append(failure.InstanceLocation, property))] = reasonUnknown
		}

		return
	}

	if len(failure.InstanceLocation) == 0 {
		return
	}

	fields[Pointer(failure.InstanceLocation)] = reasonOf(lastOf(failure.ErrorKind.KeywordPath()))
}

// Pointer builds the JSON Pointer of one instance location, in the fragment form.
// RFC 6901 escapes a tilde and a slash inside a segment.
func Pointer(location []string) string {
	var builder strings.Builder

	builder.WriteString("#")

	for _, segment := range location {
		builder.WriteString("/")
		builder.WriteString(pointerEscaper.Replace(segment))
	}

	return builder.String()
}

// sortedFailures turns the collected pointers into the order that the answer and
// the log both hold.
func sortedFailures(fields map[string]string) []FieldFailure {
	failures := make([]FieldFailure, 0, len(fields))

	for _, pointer := range slices.Sorted(maps.Keys(fields)) {
		failures = append(failures, FieldFailure{Pointer: pointer, Detail: fields[pointer]})
	}

	return failures
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
