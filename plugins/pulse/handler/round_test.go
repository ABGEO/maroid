package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
	"github.com/abgeo/maroid/plugins/pulse/round"
)

type roundItem struct {
	CheckinID   *string `json:"checkin_id"`
	CheckinName string  `json:"checkin_name"`
	State       string  `json:"state"`
	Questions   []struct {
		QuestionID string `json:"question_id"`
		State      string `json:"state"`
		Value      *value `json:"value"`
	} `json:"questions"`
}

// fireDay runs the day of PULSE-SC-042: Morning done with Weight skipped, Midday
// expired by Afternoon, and Afternoon declined.
func fireDay(t *testing.T, app *api) {
	t.Helper()

	app.TickAt(t, nina, pulsetest.At(24, 9, 0))
	require.Empty(
		t,
		app.Press(t, pulsetest.At(24, 9, 1), app.Messenger.Last().ID, round.ValueStart),
	)

	for _, answer := range []string{"v:3", "v:7", "v:4", round.ValueSkip} {
		require.Empty(t, app.Press(t, pulsetest.At(24, 9, 2), app.Messenger.Last().ID, answer))
	}

	app.TickAt(t, nina, pulsetest.At(24, 13, 0))
	app.TickAt(t, nina, pulsetest.At(24, 17, 0))
	require.Empty(
		t,
		app.Press(t, pulsetest.At(24, 17, 1), app.Messenger.Last().ID, round.ValueDecline),
	)
}

func (app *api) rounds(t *testing.T, query string) []roundItem {
	t.Helper()

	recorder := app.as(t, http.MethodGet, "/rounds"+query, "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	return decode[page[roundItem]](t, recorder).Items
}

// PULSE-SC-042: The rounds come newest first, each with its questions in order, a
// value or the state of each.
func TestTheRoundsComeNewestFirstWithTheirQuestions(t *testing.T) {
	t.Parallel()

	app := newAPI(t)
	app.TurnOn(t, nina)
	fireDay(t, app)

	items := app.rounds(t, "")
	require.Len(t, items, 3)

	names := []string{items[0].CheckinName, items[1].CheckinName, items[2].CheckinName}
	states := []string{items[0].State, items[1].State, items[2].State}

	assert.Equal(t, []string{"Afternoon", "Midday", "Morning"}, names)
	assert.Equal(t, []string{"declined", "expired", "done"}, states)

	morning := items[2].Questions
	require.Len(t, morning, 4)

	wanted := []string{
		"How is your mood?",
		"How many hours did you sleep?",
		"How well did you sleep?",
		"Weight",
	}
	for index, prompt := range wanted {
		assert.Equal(t, app.QuestionID(t, prompt), morning[index].QuestionID, prompt)
	}

	for index, number := range []float64{3, 7, 4} {
		assert.Equal(t, "answered", morning[index].State)
		require.NotNil(t, morning[index].Value)
		assert.InDelta(t, number, *morning[index].Value.Number, 0)
	}

	assert.Equal(t, "skipped", morning[3].State)
	assert.Nil(t, morning[3].Value)

	require.Len(t, items[1].Questions, 1)
	assert.Equal(t, "expired", items[1].Questions[0].State)
	require.Len(t, items[0].Questions, 1)
	assert.Equal(t, "pending", items[0].Questions[0].State)
}

// PULSE-SC-043: The rounds narrow to one check-in, to one state, or to both, and a
// filter that names nothing answers 400.
func TestTheRoundsNarrowToACheckinAndAState(t *testing.T) {
	t.Parallel()

	app := newAPI(t)
	app.TurnOn(t, nina)
	fireDay(t, app)

	morning := pulsetest.ID(t, app.DB, nina,
		"SELECT CAST(id AS text) FROM checkins WHERE name = 'Morning'")

	byCheckin := app.rounds(t, "?checkin_id="+morning)
	require.Len(t, byCheckin, 1)
	assert.Equal(t, "Morning", byCheckin[0].CheckinName)
	assert.Equal(t, morning, *byCheckin[0].CheckinID)

	byState := app.rounds(t, "?state=expired")
	require.Len(t, byState, 1)
	assert.Equal(t, "Midday", byState[0].CheckinName)

	assert.Empty(t, app.rounds(t, "?checkin_id="+morning+"&state=expired"))

	for _, query := range []string{"?state=lost", "?checkin_id=morning"} {
		assert.Equal(
			t,
			http.StatusBadRequest,
			app.as(t, http.MethodGet, "/rounds"+query, "").Code,
			query,
		)
	}
}
