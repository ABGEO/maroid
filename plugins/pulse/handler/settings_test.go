package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
)

type settingsBody struct {
	Enabled     bool    `json:"enabled"`
	PausedUntil *string `json:"paused_until"`
}

// PULSE-SC-001: With no row, GET answers off with the tag "0", and PUT with
// If-Match "0" turns pulse on and copies the defaults.
func TestTheZeroTagCreatesTheSettings(t *testing.T) {
	t.Parallel()

	app := newAPI(t)

	read := app.as(t, http.MethodGet, "/settings", "")
	require.Equal(t, http.StatusOK, read.Code)
	assert.Equal(t, `"0"`, read.Header().Get("ETag"))
	assert.False(t, decode[settingsBody](t, read).Enabled)

	stale := app.send(
		t,
		nina,
		http.MethodPut,
		"/settings",
		`{"enabled":true,"paused_until":null}`,
		`"5"`,
	)
	assert.Equal(t, http.StatusPreconditionFailed, stale.Code)

	on := app.send(
		t,
		nina,
		http.MethodPut,
		"/settings",
		`{"enabled":true,"paused_until":null}`,
		`"0"`,
	)
	require.Equal(t, http.StatusOK, on.Code, on.Body.String())
	assert.True(t, decode[settingsBody](t, on).Enabled)
	assert.Equal(t, 12, pulsetest.Count(t, app.DB, nina, "SELECT count(*) FROM questions"))

	past := app.send(t, nina, http.MethodPut, "/settings",
		`{"enabled":true,"paused_until":"2026-11-01T00:00:00Z"}`, on.Header().Get("ETag"))
	assert.Equal(t, http.StatusUnprocessableEntity, past.Code)
	assert.Equal(t, []string{"#/paused_until"}, pointers(t, past))

	missing := app.send(
		t,
		nina,
		http.MethodPut,
		"/settings",
		`{"enabled":true}`,
		on.Header().Get("ETag"),
	)
	assert.Equal(t, []string{"#/paused_until"}, pointers(t, missing))
}

// PULSE-SC-039: Beka reads pulse off, empty pages, and 404 for each record of Nina,
// and counts no row of Nina in any table.
func TestAnotherPersonReadsNothingOfNina(t *testing.T) {
	t.Parallel()

	app := newAPI(t)
	app.TurnOn(t, nina)
	app.TickAt(t, nina, pulsetest.At(24, 9, 0))

	mood := app.QuestionID(t, "How is your mood?")
	app.insertNumber(t, mood, "scale", "2026-11-24T05:00:00Z", 4)
	morning := pulsetest.ID(
		t,
		app.DB,
		nina,
		"SELECT CAST(id AS text) FROM checkins WHERE name = 'Morning'",
	)

	read := func(target string) int {
		return app.send(t, beka, http.MethodGet, target, "", "").Code
	}

	settings := app.send(t, beka, http.MethodGet, "/settings", "", "")
	assert.False(t, decode[settingsBody](t, settings).Enabled)

	for _, target := range []string{"/questions", "/questions?archived=true", "/checkins", "/rounds"} {
		recorder := app.send(t, beka, http.MethodGet, target, "", "")
		assert.Equal(t, http.StatusOK, recorder.Code, target)
		assert.Empty(t, decode[page[question]](t, recorder).Items, target)
	}

	for _, target := range []string{
		"/questions/" + mood,
		"/questions/" + mood + "/values?from=2026-11-24&to=2026-11-24",
		"/questions/" + mood + "/days?from=2026-11-24&to=2026-11-24",
		"/checkins/" + morning,
	} {
		assert.Equal(t, http.StatusNotFound, read(target), target)
	}

	for _, table := range pulsetest.Tables() {
		assert.Equal(t, 0, pulsetest.Count(t, app.DB, beka, "SELECT count(*) FROM "+table), table)
	}
}
