package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
)

type value struct {
	ID         string    `json:"id"`
	OccurredAt time.Time `json:"occurred_at"`
	Number     *float64  `json:"number"`
}

type day struct {
	Date     string   `json:"date"`
	Count    int      `json:"count"`
	Mean     *float64 `json:"mean"`
	YesCount *int     `json:"yes_count"`
}

// PULSE-SC-032: The mood of four check-ins of one day is one series of four values.
func TestOneQuestionInFourCheckinsIsOneSeries(t *testing.T) {
	t.Parallel()

	app := newAPI(t)
	app.TurnOn(t, nina)

	for _, hour := range []int{9, 13, 17, 22} {
		app.TickAt(t, nina, pulsetest.At(24, hour, 0))
		require.Empty(t, app.Press(t, pulsetest.At(24, hour, 1), app.Messenger.Last().ID, "start"))
		require.Empty(t, app.Press(t, pulsetest.At(24, hour, 2), app.Messenger.Last().ID, "v:3"))
	}

	mood := app.QuestionID(t, "How is your mood?")

	values := decode[page[value]](t, app.as(t, http.MethodGet,
		"/questions/"+mood+"/values?from=2026-11-24&to=2026-11-24", ""))
	assert.Len(t, values.Items, 4)

	days := decode[page[day]](
		t,
		app.as(t, http.MethodGet, "/questions/"+mood+"/days?from=2026-11-24&to=2026-11-24", ""),
	)
	require.Len(t, days.Items, 1)
	assert.Equal(t, 4, days.Items[0].Count)
}

// PULSE-SC-036: The values of 30 local days come newest first, 100 on the first page
// with a link to the next.
func TestTheValuesOfARangeComeNewestFirst(t *testing.T) {
	t.Parallel()

	app := newAPI(t)
	app.TurnOn(t, nina)
	mood := app.QuestionID(t, "How is your mood?")

	pulsetest.Exec(t, app.DB, nina, `
		INSERT INTO question_values (question_id, type, occurred_at, answered_at, source, number_value)
		SELECT $1, 'scale', moment AS occurred_at, moment AS answered_at, 'telegram', 3
		FROM (
			SELECT TIMESTAMPTZ '2026-10-01 08:00:00+04' + make_interval(days => i % 40, hours => (i / 40) * 3) AS moment
			FROM generate_series(0, 149) AS i
		) moments`, mood)

	target := "/questions/" + mood + "/values?from=2026-10-11&to=2026-11-09&limit=100"
	first := decode[page[value]](t, app.as(t, http.MethodGet, target, ""))
	require.Len(t, first.Items, 100)
	require.NotNil(t, first.Next)

	for index := 1; index < len(first.Items); index++ {
		assert.False(t, first.Items[index].OccurredAt.After(first.Items[index-1].OccurredAt))
	}

	second := decode[page[value]](t, app.as(t, http.MethodGet, *first.Next, ""))
	assert.Len(t, second.Items, 10)
	assert.Nil(t, second.Next)

	oldest := second.Items[len(second.Items)-1].OccurredAt.In(pulsetest.Tbilisi())
	assert.Equal(t, "2026-10-11", oldest.Format(time.DateOnly))
	assert.Equal(
		t,
		"2026-11-09",
		first.Items[0].OccurredAt.In(pulsetest.Tbilisi()).Format(time.DateOnly),
	)
}

// days reads the days of the question from and to as Nina.
func (app *api) days(t *testing.T, questionID, from, to string) *httptest.ResponseRecorder {
	t.Helper()

	return app.as(t, http.MethodGet, "/questions/"+questionID+"/days?from="+from+"&to="+to, "")
}

// PULSE-SC-037: A day follows the zone of Nina, a scale answers its count and mean,
// a yes or no answers its count of yes, and 367 days answer 422.
func TestTheDaysFollowTheZoneOfThePerson(t *testing.T) {
	t.Parallel()

	app := newAPI(t)
	app.TurnOn(t, nina)

	mood := app.QuestionID(t, "How is your mood?")
	exercised := app.QuestionID(t, "Did you exercise today?")

	app.insertNumber(t, mood, "scale", "2026-11-24T19:30:00Z", 3)
	app.insertNumber(t, mood, "scale", "2026-11-25T05:00:00Z", 5)

	for index, yes := range []bool{true, false, true} {
		pulsetest.Exec(t, app.DB, nina, `
			INSERT INTO question_values (question_id, type, occurred_at, answered_at, source, boolean_value)
			VALUES ($1, 'boolean', $2, $2, 'telegram', $3)`,
			exercised, fmt.Sprintf("2026-11-2%dT18:00:00Z", index+1), yes)
	}

	moods := decode[page[day]](t, app.days(t, mood, "2026-11-20", "2026-11-30"))
	require.Len(t, moods.Items, 2)
	assert.Equal(t, "2026-11-24", moods.Items[0].Date)
	assert.Equal(t, 1, moods.Items[0].Count)
	assert.InDelta(t, 3, *moods.Items[0].Mean, 0)
	assert.Equal(t, "2026-11-25", moods.Items[1].Date)
	assert.InDelta(t, 5, *moods.Items[1].Mean, 0)

	exercise := decode[page[day]](t, app.days(t, exercised, "2026-11-20", "2026-11-30")).Items
	yes := make([]int, 0, len(exercise))

	for _, item := range exercise {
		require.NotNil(t, item.YesCount)
		assert.Nil(t, item.Mean)

		yes = append(yes, *item.YesCount)
	}

	assert.Equal(t, []int{1, 0, 1}, yes)
	assert.Equal(
		t,
		http.StatusUnprocessableEntity,
		app.days(t, mood, "2026-01-01", "2027-01-02").Code,
	)
	assert.Equal(t, http.StatusOK, app.days(t, mood, "2026-01-01", "2027-01-01").Code)
}
