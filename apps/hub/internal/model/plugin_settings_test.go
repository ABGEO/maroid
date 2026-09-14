package model_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/model"
)

// PSET-SC-018: A log record of the settings holds the key of each field and the value
// of none, so a call site that passes the entity cannot write a secret to the log.
func TestALogRecordOfTheSettingsHoldsNoValue(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))

	entity := model.PluginSettings{
		ID:       "01998aa0-1111-7000-8000-00000000000a",
		UserID:   "01998aa0-1111-7000-8000-00000000000b",
		PluginID: "dev.maroid.probe",
		Fields: model.Fields{
			"email":    {Kind: model.FieldKindText, Value: "person@example.com"},
			"password": {Kind: model.FieldKindSecret, Value: "vault:v1:Zm9yYmlkZGVu"},
		},
	}

	logger.Debug("the settings of the acting user", slog.Any("settings", entity))

	record := output.String()

	require.Contains(t, record, "dev.maroid.probe")
	require.Contains(t, record, "password")
	require.NotContains(t, record, "vault:v1:")
	require.NotContains(t, record, "person@example.com")
}

// PSET-FR-020: The entries redact on their own, so a call site that passes them
// without the entity leaks nothing either.
func TestALogRecordOfTheFieldsHoldsNoValue(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
	fields := model.Fields{
		"password": {Kind: model.FieldKindSecret, Value: "vault:v1:Zm9yYmlkZGVu"},
	}

	logger.Debug("the fields", slog.Any("fields", fields))

	require.NotContains(t, output.String(), "vault:v1:")
}
