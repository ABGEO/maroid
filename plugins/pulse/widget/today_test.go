package widget_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginapi/telegram/message"
	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
	"github.com/abgeo/maroid/plugins/pulse/round"
	"github.com/abgeo/maroid/plugins/pulse/widget"
)

const (
	nina = pulsetest.Nina
	beka = pulsetest.Beka

	// middle answers a scale from 1 to 5 with 3.
	middle = "v:3"
)

type day struct {
	*pulsetest.Pulse

	today pluginapi.Widget
}

// newDay answers pulse with the defaults of Nina, turned on at 08:00 on the day.
func newDay(t *testing.T, onDay int) *day {
	t.Helper()

	app := pulsetest.New(t)
	app.Clock.Set(pulsetest.At(onDay, 8, 0))
	app.TurnOn(t, nina)

	return &day{Pulse: app, today: widget.NewToday(app.DB, app.Clock.Now)}
}

func (d *day) loadAt(t *testing.T, moment time.Time) pluginapi.WidgetAnswer {
	t.Helper()

	return d.loadAs(pulsetest.ActingAs(nina), t, moment)
}

func (d *day) loadAs(ctx context.Context, t *testing.T, moment time.Time) pluginapi.WidgetAnswer {
	t.Helper()

	d.Clock.Set(moment)

	answer, err := d.today.Load(ctx)
	require.NoError(t, err)

	return answer
}

func (d *day) checkinID(t *testing.T, name string) string {
	t.Helper()

	return pulsetest.ID(
		t,
		d.DB,
		nina,
		"SELECT CAST(id AS text) FROM checkins WHERE name = $1",
		name,
	)
}

func (d *day) startMessage(t *testing.T, name string) message.ID {
	t.Helper()

	return message.ID(pulsetest.ID(t, d.DB, nina,
		"SELECT start_message_id FROM rounds WHERE checkin_name = $1", name))
}

// addCheckin adds a daily check-in at the local time with the question mood, as if
// Nina created it at the moment.
func (d *day) addCheckin(t *testing.T, name string, clock string, created time.Time) {
	t.Helper()

	pulsetest.Exec(t, d.DB, nina, `
		INSERT INTO checkins (name, local_times, fires_after)
		VALUES ($1, CAST(string_to_array($2, ',') AS time[]), $3);`, name, clock, created)
	pulsetest.Exec(t, d.DB, nina, `
		INSERT INTO checkin_questions (checkin_id, question_id, position)
		SELECT c.id, q.id, 1 FROM checkins c, questions q
		WHERE c.name = $1 AND q.prompt = 'How is your mood?';`, name)
}

// answerMorning starts Morning and answers each of its four questions.
func (d *day) answerMorning(t *testing.T, moment time.Time) {
	t.Helper()

	id := d.startMessage(t, "Morning")
	require.Empty(t, d.Press(t, moment, id, round.ValueStart))

	for _, answer := range []string{middle, "v:7", "v:4", round.ValueSkip} {
		require.Empty(t, d.Press(t, moment, d.Messenger.Last().ID, answer))
	}
}

func (d *day) item(t *testing.T, name string, meta string) pluginapi.ListItem {
	t.Helper()

	return pluginapi.ListItem{Label: name, Meta: meta, Link: "/history/" + d.checkinID(t, name)}
}

func list(items ...pluginapi.ListItem) pluginapi.WidgetAnswer {
	return pluginapi.ListAnswer(pluginapi.ListData{Items: items, More: "/history"})
}

// PULSE-SC-045: The widget lists the rounds of today and the occurrences still to
// come, by time then by name, each with its state and its page of history.
func TestTodayListsTheRoundsAndTheUpcomingOccurrences(t *testing.T) {
	t.Parallel()

	app := newDay(t, 24)
	app.addCheckin(t, "Water", "13:00", pulsetest.At(24, 8, 0))

	app.TickAt(t, nina, pulsetest.At(24, 9, 0))
	app.answerMorning(t, pulsetest.At(24, 9, 1))
	app.TickAt(t, nina, pulsetest.At(24, 13, 0))
	require.Empty(t, app.Press(t, pulsetest.At(24, 13, 1),
		app.startMessage(t, "Midday"), round.ValueDecline))

	assert.Equal(t, list(
		app.item(t, "Morning", "09:00 · Done"),
		app.item(t, "Midday", "13:00 · Declined"),
		app.item(t, "Water", "13:00 · Waiting to start"),
		app.item(t, "Afternoon", "17:00 · Upcoming"),
		app.item(t, "Evening", "22:00 · Upcoming"),
	), app.loadAt(t, pulsetest.At(24, 14, 0)))

	assert.Equal(t, pluginapi.WidgetMeta{
		Key:        "today",
		Title:      "Today",
		Kind:       pluginapi.WidgetKindList,
		Size:       pluginapi.WidgetSizeM,
		Permission: "checkins.use",
	}, app.today.Meta())
}

// PULSE-SC-046: A round waits to start, then counts its answered questions while it
// is in progress.
func TestTodayCountsTheAnswersOfARoundInProgress(t *testing.T) {
	t.Parallel()

	app := newDay(t, 24)
	app.TickAt(t, nina, pulsetest.At(24, 22, 0))

	assert.Equal(t,
		list(app.item(t, "Evening", "22:00 · Waiting to start")),
		app.loadAt(t, pulsetest.At(24, 22, 1)))

	require.Empty(t, app.Press(t, pulsetest.At(24, 22, 2),
		app.startMessage(t, "Evening"), round.ValueStart))

	for _, answer := range []string{middle, middle, "v:1", round.ValueSkip} {
		require.Empty(t, app.Press(t, pulsetest.At(24, 22, 3), app.Messenger.Last().ID, answer))
	}

	assert.Equal(t,
		list(app.item(t, "Evening", "22:00 · 3 of 7 answered")),
		app.loadAt(t, pulsetest.At(24, 22, 4)))
}

// PULSE-SC-047: A round of yesterday does not show, and a round whose check-in was
// deleted keeps its name and links to the whole history.
func TestTodayKeepsTheDayAndTheRoundOfADeletedCheckin(t *testing.T) {
	t.Parallel()

	app := newDay(t, 23)
	app.TickAt(t, nina, pulsetest.At(23, 22, 0))
	app.TickAt(t, nina, pulsetest.At(24, 9, 0))
	app.TickAt(t, nina, pulsetest.At(24, 13, 0))
	pulsetest.Exec(t, app.DB, nina, "DELETE FROM checkins WHERE name = 'Midday'")

	assert.Equal(t, list(
		app.item(t, "Morning", "09:00 · Expired"),
		pluginapi.ListItem{Label: "Midday", Meta: "13:00 · Waiting to start", Link: "/history"},
		app.item(t, "Afternoon", "17:00 · Upcoming"),
		app.item(t, "Evening", "22:00 · Upcoming"),
	), app.loadAt(t, pulsetest.At(24, 14, 0)))
}

// PULSE-SC-048: After a move to Chicago, today follows the new zone, so the round of
// Morning falls on the day before and every check-in is upcoming.
func TestTodayFollowsTheZoneOfTheRead(t *testing.T) {
	t.Parallel()

	app := newDay(t, 24)
	app.TickAt(t, nina, pulsetest.At(24, 9, 0))
	app.answerMorning(t, pulsetest.At(24, 9, 1))

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	assert.Equal(t, list(
		app.item(t, "Morning", "09:00 · Upcoming"),
		app.item(t, "Midday", "13:00 · Upcoming"),
		app.item(t, "Afternoon", "17:00 · Upcoming"),
		app.item(t, "Evening", "22:00 · Upcoming"),
	), app.loadAs(pulsetest.ActingIn(nina, chicago), t, pulsetest.At(24, 10, 30)))
}

// PULSE-SC-049: An occurrence inside the pause, and one of a check-in turned off, do
// not show.
func TestTodayLeavesOutThePauseAndATurnedOffCheckin(t *testing.T) {
	t.Parallel()

	app := newDay(t, 24)
	app.TickAt(t, nina, pulsetest.At(24, 9, 0))
	app.answerMorning(t, pulsetest.At(24, 9, 1))
	app.Clock.Set(pulsetest.At(24, 14, 0))
	require.NoError(t, app.Settings.Pause(pulsetest.ActingAs(nina), pulsetest.At(24, 18, 0)))

	assert.Equal(t, list(
		app.item(t, "Morning", "09:00 · Done"),
		app.item(t, "Evening", "22:00 · Upcoming"),
	), app.loadAt(t, pulsetest.At(24, 14, 0)))

	pulsetest.Exec(t, app.DB, nina, "UPDATE checkins SET enabled = false WHERE name = 'Evening'")

	assert.Equal(t,
		list(app.item(t, "Morning", "09:00 · Done")),
		app.loadAt(t, pulsetest.At(24, 14, 0)))
}

// PULSE-SC-050: An earlier occurrence with no round never shows, whether the
// check-in came after its time or the tick has not fired it yet.
func TestTodayLeavesOutAnOccurrenceThatNeverFired(t *testing.T) {
	t.Parallel()

	app := newDay(t, 24)
	app.TickAt(t, nina, pulsetest.At(24, 9, 0))
	app.addCheckin(t, "Water", "09:00", pulsetest.At(24, 10, 0))

	assert.Equal(t, list(
		app.item(t, "Morning", "09:00 · Waiting to start"),
		app.item(t, "Midday", "13:00 · Upcoming"),
		app.item(t, "Afternoon", "17:00 · Upcoming"),
		app.item(t, "Evening", "22:00 · Upcoming"),
	), app.loadAt(t, pulsetest.At(24, 10, 0)))

	assert.Equal(t, list(
		app.item(t, "Morning", "09:00 · Waiting to start"),
		app.item(t, "Afternoon", "17:00 · Upcoming"),
		app.item(t, "Evening", "22:00 · Upcoming"),
	), app.loadAt(t, pulsetest.At(24, 13, 1)))
}

// PULSE-SC-051: A person who never turned pulse on, and a person who turned it off,
// read "not set up" with a link to the preferences, and nothing of another person.
func TestTodayAsksAPersonWithPulseOffToTurnItOn(t *testing.T) {
	t.Parallel()

	app := newDay(t, 24)
	app.TickAt(t, nina, pulsetest.At(24, 9, 0))

	notSetUp := pluginapi.NotSetUpAnswer("Turn pulse on to receive check-ins.", "/preferences")

	assert.Equal(t, notSetUp, app.loadAs(pulsetest.ActingAs(beka), t, pulsetest.At(24, 14, 0)))

	_, err := app.Settings.Put(pulsetest.ActingAs(nina), false, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, notSetUp, app.loadAt(t, pulsetest.At(24, 14, 0)))
}

// PULSE-SC-052: With no item, a pause shows its end, and no pause shows that no
// check-in comes today. A round of today still shows during a pause.
func TestTodayExplainsAnEmptyDay(t *testing.T) {
	t.Parallel()

	endOfPause := time.Date(2026, time.December, 1, 0, 0, 0, 0, pulsetest.Tbilisi())

	paused := newDay(t, 24)
	require.NoError(t, paused.Settings.Pause(pulsetest.ActingAs(nina), endOfPause))
	assert.Equal(t,
		pluginapi.EmptyAnswer("Check-ins paused until 1 Dec 2026, 00:00."),
		paused.loadAt(t, pulsetest.At(24, 8, 30)))

	idle := newDay(t, 24)
	pulsetest.Exec(t, idle.DB, nina, "UPDATE checkins SET enabled = false")
	assert.Equal(t,
		pluginapi.EmptyAnswer("No check-in today."),
		idle.loadAt(t, pulsetest.At(24, 8, 30)))

	answered := newDay(t, 24)
	answered.TickAt(t, nina, pulsetest.At(24, 9, 0))
	answered.answerMorning(t, pulsetest.At(24, 9, 1))
	answered.Clock.Set(pulsetest.At(24, 9, 30))
	require.NoError(t, answered.Settings.Pause(pulsetest.ActingAs(nina), endOfPause))
	assert.Equal(t,
		list(answered.item(t, "Morning", "09:00 · Done")),
		answered.loadAt(t, pulsetest.At(24, 9, 30)))
}
