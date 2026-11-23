package handler_test

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
)

type checkin struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	LocalTimes  []string `json:"local_times"`
	Weekdays    []int    `json:"weekdays"`
	Enabled     bool     `json:"enabled"`
	QuestionIDs []string `json:"question_ids"`
}

func checkinJSON(name string, times []string, weekdays []int, questions []string) string {
	quote := func(items []string) string {
		quoted := make([]string, 0, len(items))
		for _, item := range items {
			quoted = append(quoted, fmt.Sprintf("%q", item))
		}

		return "[" + strings.Join(quoted, ",") + "]"
	}

	days := make([]string, 0, len(weekdays))
	for _, day := range weekdays {
		days = append(days, strconv.Itoa(day))
	}

	return fmt.Sprintf(
		`{"name":%q,"local_times":%s,"weekdays":[%s],"enabled":true,"question_ids":%s}`,
		name,
		quote(times),
		strings.Join(days, ","),
		quote(questions),
	)
}

// PULSE-SC-031: A check-in takes a name, 1 to 6 valid times, the weekdays, and 1 to
// 20 questions, and a deleted check-in leaves its values.
func TestACheckinIsCreatedAndDeleted(t *testing.T) {
	t.Parallel()

	app := newAPI(t)
	app.TurnOn(t, nina)
	energy := app.QuestionID(t, "How is your energy?")

	gym := app.as(
		t,
		http.MethodPost,
		"/checkins",
		checkinJSON("Gym", []string{"07:30"}, []int{1, 3, 5}, []string{energy}),
	)
	require.Equal(t, http.StatusCreated, gym.Code, gym.Body.String())
	assert.Equal(t, checkin{
		ID: decode[checkin](t, gym).ID, Name: "Gym", LocalTimes: []string{"07:30"},
		Weekdays: []int{1, 3, 5}, Enabled: true, QuestionIDs: []string{energy},
	}, decode[checkin](t, gym))

	assertCheckinLimits(t, app, energy)

	app.TickAt(t, nina, pulsetest.At(24, 9, 0))
	require.Empty(t, app.Press(t, pulsetest.At(24, 9, 1), app.Messenger.Last().ID, "start"))
	require.Empty(t, app.Press(t, pulsetest.At(24, 9, 2), app.Messenger.Last().ID, "v:4"))

	morning := pulsetest.ID(
		t,
		app.DB,
		nina,
		"SELECT CAST(id AS text) FROM checkins WHERE name = 'Morning'",
	)
	assert.Equal(
		t,
		http.StatusNoContent,
		app.as(t, http.MethodDelete, "/checkins/"+morning, "").Code,
	)
	assert.Equal(t, 1, pulsetest.Count(t, app.DB, nina, "SELECT count(*) FROM question_values"))
}

func assertCheckinLimits(t *testing.T, app *api, questionID string) {
	t.Helper()

	late := app.as(
		t,
		http.MethodPost,
		"/checkins",
		checkinJSON("Late", []string{"25:00"}, nil, []string{questionID}),
	)
	assert.Equal(t, http.StatusUnprocessableEntity, late.Code)
	assert.Equal(t, []string{"#/local_times/0"}, pointers(t, late))

	seven := []string{"01:00", "02:00", "03:00", "04:00", "05:00", "06:00", "07:00"}
	assert.Equal(
		t,
		http.StatusUnprocessableEntity,
		app.as(
			t,
			http.MethodPost,
			"/checkins",
			checkinJSON("Often", seven, nil, []string{questionID}),
		).Code,
	)

	many := make([]string, 21)
	for index := range many {
		many[index] = uuid.Must(uuid.NewV7()).String()
	}

	assert.Equal(
		t,
		http.StatusUnprocessableEntity,
		app.as(
			t,
			http.MethodPost,
			"/checkins",
			checkinJSON("Long", []string{"08:00"}, nil, many),
		).Code,
	)
}

// PULSE-SC-012: A check-in created at 10:00 for 09:00, and a check-in moved to 10:00
// at 10:05, fire first on the next day.
func TestANewTimeFiresFirstAtItsNextOccurrence(t *testing.T) {
	t.Parallel()

	app := newAPI(t)
	app.TurnOn(t, nina)
	mood := app.QuestionID(t, "How is your mood?")

	app.Clock.Set(pulsetest.At(24, 10, 0))
	tea := app.as(
		t,
		http.MethodPost,
		"/checkins",
		checkinJSON("Tea", []string{"09:00"}, nil, []string{mood}),
	)
	require.Equal(t, http.StatusCreated, tea.Code, tea.Body.String())

	midday := pulsetest.ID(
		t,
		app.DB,
		nina,
		"SELECT CAST(id AS text) FROM checkins WHERE name = 'Midday'",
	)
	app.Clock.Set(pulsetest.At(24, 10, 5))
	moved := app.as(
		t,
		http.MethodPut,
		"/checkins/"+midday,
		checkinJSON("Midday", []string{"10:00"}, nil, []string{mood}),
	)
	require.Equal(t, http.StatusOK, moved.Code, moved.Body.String())

	for now := pulsetest.At(24, 10, 5); !now.After(pulsetest.At(25, 9, 1)); now = now.Add(time.Minute) {
		app.TickAt(t, nina, now)
	}

	var fired []time.Time

	require.NoError(t, app.DB.WithTx(pulsetest.ActingAs(nina), func(tx *sqlx.Tx) error {
		return tx.Select(
			&fired,
			"SELECT occurrence_at FROM rounds WHERE checkin_name IN ('Tea', 'Midday') ORDER BY occurrence_at",
		)
	}))

	require.Len(t, fired, 1)
	assert.True(t, fired[0].Equal(pulsetest.At(25, 9, 0)))
	assert.Equal(
		t,
		1,
		pulsetest.Count(t, app.DB, nina, "SELECT count(*) FROM rounds WHERE checkin_name = 'Tea'"),
	)
}
