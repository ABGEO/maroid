package round_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi/telegram/message"
	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
	"github.com/abgeo/maroid/plugins/pulse/round"
)

const nina = pulsetest.Nina

type storedValue struct {
	Prompt     string    `db:"prompt"`
	Type       string    `db:"type"`
	Number     *float64  `db:"number_value"`
	Boolean    *bool     `db:"boolean_value"`
	Text       *string   `db:"text_value"`
	Latitude   *float64  `db:"latitude"`
	Longitude  *float64  `db:"longitude"`
	OccurredAt time.Time `db:"occurred_at"`
	AnsweredAt time.Time `db:"answered_at"`
	Source     string    `db:"source"`
}

func storedValues(t *testing.T, pulse *pulsetest.Pulse) []storedValue {
	t.Helper()

	var stored []storedValue

	require.NoError(t, pulse.DB.WithTx(pulsetest.ActingAs(nina), func(tx *sqlx.Tx) error {
		return tx.Select(&stored, `
			SELECT q.prompt, v.type, CAST(v.number_value AS float8) AS number_value,
			       v.boolean_value, v.text_value, v.latitude, v.longitude,
			       v.occurred_at, v.answered_at, v.source
			FROM question_values v JOIN questions q ON q.id = v.question_id
			ORDER BY v.answered_at, v.id`)
	}))

	return stored
}

// offer turns pulse on at 06:00 on the day, and fires the check-in at its time. It
// answers the message of the start prompt.
func offer(t *testing.T, pulse *pulsetest.Pulse, day int, hour int) message.ID {
	t.Helper()

	if day == 24 && pulsetest.Count(t, pulse.DB, nina, "SELECT count(*) FROM settings") == 0 {
		pulse.TurnOn(t, nina)
	}

	pulse.TickAt(t, nina, pulsetest.At(day, hour, 0))

	return pulse.Messenger.Last().ID
}

// start presses Start and answers the message of the first question.
func start(
	t *testing.T,
	pulse *pulsetest.Pulse,
	moment time.Time,
	startPrompt message.ID,
) message.ID {
	t.Helper()

	assert.Empty(t, pulse.Press(t, moment, startPrompt, round.ValueStart))

	return pulse.Messenger.Last().ID
}

// skip presses Skip on count questions in turn, and answers the message of the
// question after them.
func skip(
	t *testing.T,
	pulse *pulsetest.Pulse,
	moment time.Time,
	current message.ID,
	count int,
) message.ID {
	t.Helper()

	for range count {
		assert.Empty(t, pulse.Press(t, moment, current, round.ValueSkip))
		current = pulse.Messenger.Last().ID
	}

	return current
}

const (
	skipLabel = "Skip"
	four      = "v:4"
)

// PULSE-SC-018: Decline closes the round and sends nothing else.
func TestDeclineClosesTheRound(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	first := offer(t, pulse, 24, 9)

	assert.Empty(t, pulse.Press(t, pulsetest.At(24, 9, 2), first, round.ValueDecline))

	assert.Equal(t, "Morning check-in: declined.", pulse.Messenger.Text(first))
	assert.Len(t, pulse.Messenger.Sent(), 1)
	assert.Equal(
		t,
		1,
		pulsetest.Count(t, pulse.DB, nina, "SELECT count(*) FROM rounds WHERE state = 'declined'"),
	)
}

// PULSE-SC-019: Start sends the first question, and an answer edits it, records the
// value with the moment of the occurrence, and sends the next question.
func TestAnAnswerRecordsTheValueAndAsksTheNext(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	first := offer(t, pulse, 24, 9)
	mood := start(t, pulse, pulsetest.At(24, 9, 5), first)

	assert.Equal(t, message.ID("m2"), mood)
	assert.Equal(t, "Morning check-in, 4 questions. Started.", pulse.Messenger.Text(first))
	assert.Equal(t, "How is your mood? (1 bad, 5 great)", pulse.Messenger.Last().Message.Text)

	assert.Empty(t, pulse.Press(t, pulsetest.At(24, 9, 40), mood, four))

	assert.Equal(t, "How is your mood? 4", pulse.Messenger.Text(mood))
	assert.Equal(t, message.ID("m3"), pulse.Messenger.Last().ID)
	assert.Equal(t, "How many hours did you sleep? (h)", pulse.Messenger.Last().Message.Text)

	assert.Equal(t, []storedValue{{
		Prompt:     "How is your mood?",
		Type:       "scale",
		Number:     new(4.0),
		OccurredAt: pulsetest.At(24, 9, 0).UTC(),
		AnsweredAt: pulsetest.At(24, 9, 40).UTC(),
		Source:     "telegram",
	}}, utcValues(storedValues(t, pulse)))
}

func utcValues(stored []storedValue) []storedValue {
	for index := range stored {
		stored[index].OccurredAt = stored[index].OccurredAt.UTC()
		stored[index].AnsweredAt = stored[index].AnsweredAt.UTC()
	}

	return stored
}

// PULSE-SC-020: Skip on each question records nothing, and the round is done with no
// message after the last edit.
func TestARoundOfSkipsIsDone(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	first := offer(t, pulse, 24, 9)
	current := start(t, pulse, pulsetest.At(24, 9, 5), first)

	skip(t, pulse, pulsetest.At(24, 9, 6), current, 4)

	assert.Equal(t, "How is your mood? skipped", pulse.Messenger.Text("m2"))
	assert.Equal(t, "How many hours did you sleep? (h) skipped", pulse.Messenger.Text("m3"))
	assert.Equal(t, "How well did you sleep? skipped", pulse.Messenger.Text("m4"))
	assert.Equal(t, "Weight (kg) skipped", pulse.Messenger.Text("m5"))
	assert.Equal(t, "Morning check-in: done.", pulse.Messenger.Text(first))
	assert.Len(t, pulse.Messenger.Sent(), 5)
	assert.Empty(t, storedValues(t, pulse))

	edits := pulse.Messenger.Edits()
	assert.Equal(t, first, edits[len(edits)-1].ID)
}

// PULSE-SC-022: A number takes a reply with a comma, the bound itself, and a quick
// value, and a reply out of range gets the range while the claim stays.
func TestANumberTakesAReplyInsideItsBounds(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	first := offer(t, pulse, 24, 9)
	weight := skip(
		t,
		pulse,
		pulsetest.At(24, 9, 6),
		start(t, pulse, pulsetest.At(24, 9, 5), first),
		3,
	)

	pulse.Reply(t, pulsetest.At(24, 9, 7), message.Reply{Message: weight, Text: "900"})
	assert.Equal(
		t,
		"Send a number from 30 to 250, or press Skip.",
		pulse.Messenger.Last().Message.Text,
	)
	assert.False(t, pulse.Messenger.Released(weight))

	pulse.Reply(t, pulsetest.At(24, 9, 8), message.Reply{Message: weight, Text: "72,5"})
	assert.True(t, pulse.Messenger.Released(weight))
	assert.Equal(t, "Weight (kg) 72.5", pulse.Messenger.Text(weight))

	second := offer(t, pulse, 25, 9)
	sleep := skip(
		t,
		pulse,
		pulsetest.At(25, 9, 6),
		start(t, pulse, pulsetest.At(25, 9, 5), second),
		1,
	)
	assert.Empty(t, pulse.Press(t, pulsetest.At(25, 9, 7), sleep, "v:8"))
	weight = skip(t, pulse, pulsetest.At(25, 9, 8), pulse.Messenger.Last().ID, 1)
	pulse.Reply(t, pulsetest.At(25, 9, 9), message.Reply{Message: weight, Text: "250"})

	numbers := make(map[string][]float64)
	for _, value := range storedValues(t, pulse) {
		numbers[value.Prompt] = append(numbers[value.Prompt], *value.Number)
	}

	assert.Equal(t, map[string][]float64{
		"Weight":                        {72.5, 250},
		"How many hours did you sleep?": {8},
	}, numbers)
}

// PULSE-SC-023: A text of 1001 characters gets the limit, and 1000 characters record.
func TestATextTakesAtMostAThousandCharacters(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	first := offer(t, pulse, 24, 22)
	mainThing := skip(
		t,
		pulse,
		pulsetest.At(24, 22, 6),
		start(t, pulse, pulsetest.At(24, 22, 5), first),
		5,
	)

	pulse.Reply(
		t,
		pulsetest.At(24, 22, 7),
		message.Reply{Message: mainThing, Text: strings.Repeat("a", 1001)},
	)
	assert.Equal(
		t,
		"Send at most 1000 characters, or press Skip.",
		pulse.Messenger.Last().Message.Text,
	)
	assert.Empty(t, storedValues(t, pulse))

	pulse.Reply(
		t,
		pulsetest.At(24, 22, 8),
		message.Reply{Message: mainThing, Text: strings.Repeat("a", 1000)},
	)

	stored := storedValues(t, pulse)
	require.Len(t, stored, 1)
	assert.Equal(t, strings.Repeat("a", 1000), *stored[0].Text)
}

// PULSE-SC-024: A location question shows Send location and Skip, refuses a typed
// text, records a shared location, and removes the keyboard after the answer or the
// skip.
func TestALocationRemovesItsKeyboard(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	first := offer(t, pulse, 24, 22)
	location := skip(
		t,
		pulse,
		pulsetest.At(24, 22, 6),
		start(t, pulse, pulsetest.At(24, 22, 5), first),
		6,
	)

	assert.Equal(t, message.ID("m8"), location)
	assert.Equal(t, [][]message.Key{{
		{Label: "Send location", RequestLocation: true},
		{Label: skipLabel},
	}}, pulse.Messenger.Last().Message.Keyboard)

	pulse.Reply(t, pulsetest.At(24, 22, 7), message.Reply{Message: location, Text: "home"})
	assert.Equal(t, "Share a location, or press Skip.", pulse.Messenger.Last().Message.Text)

	pulse.Reply(t, pulsetest.At(24, 22, 8), message.Reply{
		Message:  location,
		Location: &message.Location{Latitude: 41.7151, Longitude: 44.8271},
	})
	assert.Equal(
		t,
		message.Message{Text: "Location saved.", RemoveKeyboard: true},
		pulse.Messenger.Last().Message,
	)

	stored := storedValues(t, pulse)
	require.Len(t, stored, 1)
	assert.InDelta(t, 41.7151, *stored[0].Latitude, 1e-9)
	assert.InDelta(t, 44.8271, *stored[0].Longitude, 1e-9)

	second := offer(t, pulse, 25, 22)
	location = skip(
		t,
		pulse,
		pulsetest.At(25, 22, 6),
		start(t, pulse, pulsetest.At(25, 22, 5), second),
		6,
	)
	pulse.Reply(t, pulsetest.At(25, 22, 7), message.Reply{Message: location, Text: skipLabel})

	assert.Equal(
		t,
		message.Message{Text: "Skipped.", RemoveKeyboard: true},
		pulse.Messenger.Last().Message,
	)
	assert.Len(t, storedValues(t, pulse), 1)
}

// PULSE-SC-025: A press on a round that expired records nothing, and of two parallel
// presses on one question one records the value.
func TestAClosedQuestionRecordsNothing(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	yesterday := start(t, pulse, pulsetest.At(24, 9, 5), offer(t, pulse, 24, 9))
	today := start(t, pulse, pulsetest.At(25, 9, 5), offer(t, pulse, 25, 9))

	assert.Equal(t, round.ClosedNotice, pulse.Press(t, pulsetest.At(25, 9, 6), yesterday, "v:3"))
	assert.Empty(t, storedValues(t, pulse))

	notices := make([]string, 2)

	var wait sync.WaitGroup

	for index := range notices {
		wait.Go(func() {
			notice, err := pulse.Rounds.Press(
				pulsetest.ActingAs(nina),
				message.Press{Message: today, Value: four},
			)
			assert.NoError(t, err)

			notices[index] = notice
		})
	}

	wait.Wait()

	assert.ElementsMatch(t, []string{"", round.ClosedNotice}, notices)
	assert.Equal(t, 1, pulsetest.Count(t, pulse.DB, nina, "SELECT count(*) FROM question_values"))
}

// PULSE-SC-034: An answer after midnight keeps the moment of the occurrence of the
// evening before.
func TestAValueKeepsTheMomentOfItsOccurrence(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	mood := start(t, pulse, pulsetest.At(24, 23, 0), offer(t, pulse, 24, 22))

	assert.Empty(t, pulse.Press(t, pulsetest.At(25, 0, 40), mood, "v:3"))

	stored := utcValues(storedValues(t, pulse))
	require.Len(t, stored, 1)
	assert.Equal(t, pulsetest.At(24, 22, 0).UTC(), stored[0].OccurredAt)
	assert.Equal(t, pulsetest.At(25, 0, 40).UTC(), stored[0].AnsweredAt)
}
