package settings_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/settings"
)

func probeSchema(t *testing.T) *settings.Schema {
	t.Helper()

	schema, err := settings.Infer(&probeModel{})
	require.NoError(t, err)

	return schema
}

func rejectedFields(t *testing.T, input map[string]any) []settings.FieldFailure {
	t.Helper()

	err := settings.Validate(probeSchema(t), input)
	require.Error(t, err)

	var invalid *settings.InvalidError

	require.ErrorAs(t, err, &invalid)

	return invalid.Fields
}

// PSET-SC-007: Each unwanted case is rejected, and the answer names that field.
func TestValidateNamesTheFieldThatCausedTheRejection(t *testing.T) {
	t.Parallel()

	t.Run("an unknown field", func(t *testing.T) {
		t.Parallel()

		fields := rejectedFields(t, map[string]any{"nickname": "abgeo"})
		require.Equal(t, []settings.FieldFailure{
			{Pointer: settings.Pointer([]string{"nickname"}), Detail: "the field is unknown"},
		}, fields)
	})

	t.Run("a value outside the list", func(t *testing.T) {
		t.Parallel()

		fields := rejectedFields(t, map[string]any{keyPeriod: "week"})
		require.Equal(t, []settings.FieldFailure{
			{
				Pointer: settings.Pointer([]string{keyPeriod}),
				Detail:  "the value is not in the list",
			},
		}, fields)
	})

	t.Run("a value that is too long", func(t *testing.T) {
		t.Parallel()

		long := strings.Repeat("a", settings.MaxValueLength+1)
		fields := rejectedFields(t, map[string]any{keyEmail: long})
		require.Equal(t, []settings.FieldFailure{
			{Pointer: settings.Pointer([]string{keyEmail}), Detail: "the value is too long"},
		}, fields)
	})

	t.Run("a value of the wrong type", func(t *testing.T) {
		t.Parallel()

		fields := rejectedFields(t, map[string]any{keyNotify: "yes"})
		require.Equal(t, []settings.FieldFailure{
			{
				Pointer: settings.Pointer([]string{keyNotify}),
				Detail:  "the value has the wrong type",
			},
		}, fields)
	})
}

// PSET-SC-007: The normal case passes, and a removal carries no value to judge.
func TestValidateAcceptsTheDeclaredFields(t *testing.T) {
	t.Parallel()

	err := settings.Validate(probeSchema(t), map[string]any{
		keyEmail:    valueEmail,
		keyPassword: valuePassword,
		keyPeriod:   valuePeriod,
		keyNotify:   true,
		keyAccount:  nil,
	})

	require.NoError(t, err)
}

// PSET-FR-006: A save that names no value for a stored secret passes validation,
// because the required check runs against the merged fields.
func TestValidateAcceptsASaveThatOmitsAStoredSecret(t *testing.T) {
	t.Parallel()

	err := settings.Validate(probeSchema(t), map[string]any{keyEmail: valueEmail})

	require.NoError(t, err)
}
