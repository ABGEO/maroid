package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/logger"
	"github.com/abgeo/maroid/libs/problem"
)

func recordOf(t *testing.T, buffer *bytes.Buffer) map[string]any {
	t.Helper()

	var record map[string]any

	require.NoError(t, json.Unmarshal(buffer.Bytes(), &record), buffer.String())

	return record
}

// LOG-009: A log record of an HTTP request carries the `request` attribute, as
// the bare UUID.
func TestARecordOfARequestCarriesTheIdentifier(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	log := slog.New(logger.WithRequest(slog.NewJSONHandler(buffer, nil)))

	ctx := problem.ContextWithRequestID(t.Context(), "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e77")
	log.InfoContext(ctx, "the plugin answered")

	record := recordOf(t, buffer)
	assert.Equal(t, "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e77", record["request"])
}

// LOG-009: A record outside a request carries no `request` attribute.
func TestARecordOutsideARequestCarriesNoIdentifier(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	log := slog.New(logger.WithRequest(slog.NewJSONHandler(buffer, nil)))

	log.InfoContext(t.Context(), "the worker started")

	assert.NotContains(t, recordOf(t, buffer), "request")
}

// LOG-003: A component adds its attributes with With, and the wrapper survives,
// so a record of that component still carries the identifier.
func TestTheIdentifierSurvivesWith(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	log := slog.New(logger.WithRequest(slog.NewJSONHandler(buffer, nil))).
		With(slog.String("component", "handler"))

	ctx := problem.ContextWithRequestID(t.Context(), "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e77")
	log.ErrorContext(ctx, "the handler errored")

	record := recordOf(t, buffer)
	assert.Equal(t, "handler", record["component"])
	assert.Equal(t, "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e77", record["request"])
}
