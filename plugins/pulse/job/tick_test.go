package job_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi/telegram/message"
	"github.com/abgeo/maroid/plugins/pulse/job"
	"github.com/abgeo/maroid/plugins/pulse/model"
	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
	"github.com/abgeo/maroid/plugins/pulse/repository"
	"github.com/abgeo/maroid/plugins/pulse/round"
)

const (
	nina = pulsetest.Nina
	beka = pulsetest.Beka
)

func texts(sent []pulsetest.Sent) []string {
	out := make([]string, 0, len(sent))
	for _, item := range sent {
		out = append(out, item.Message.Text)
	}

	return out
}

// PULSE-SC-002: A tick for a person who never turned pulse on writes no row and
// calls no service.
func TestATickForAPersonWhoIsOffDoesNothing(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	pulse.TickAt(t, beka, pulsetest.At(24, 9, 0))

	for _, table := range pulsetest.Tables() {
		assert.Equal(t, 0, pulsetest.Count(t, pulse.DB, beka, "SELECT count(*) FROM "+table), table)
	}

	assert.Zero(t, pulse.Messenger.Calls())
}

// PULSE-SC-003: A turn-off expires the offered round and stops the ticks, and the
// records stay.
func TestATurnOffExpiresTheOpenRound(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	pulse.TurnOn(t, nina)
	pulse.TickAt(t, nina, pulsetest.At(24, 9, 0))

	pulse.Clock.Set(pulsetest.At(24, 9, 10))
	_, err := pulse.Settings.Put(pulsetest.ActingAs(nina), false, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "Morning check-in: expired.", pulse.Messenger.Text("m1"))
	assert.Equal(
		t,
		1,
		pulsetest.Count(t, pulse.DB, nina, "SELECT count(*) FROM rounds WHERE state = 'expired'"),
	)

	pulse.TickAt(t, nina, pulsetest.At(24, 13, 0))

	assert.Len(t, pulse.Messenger.Sent(), 1)
	assert.Equal(t, 12, pulsetest.Count(t, pulse.DB, nina, "SELECT count(*) FROM questions"))
	assert.Equal(t, 5, pulsetest.Count(t, pulse.DB, nina, "SELECT count(*) FROM checkins"))
}

// PULSE-SC-008: After a change of the zone to Europe/Berlin, "Morning" fires at 09:00
// Berlin on that day and the next, and at no other moment.
func TestAChangeOfTheZoneFiresAtTheNewLocalTime(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	pulse.TurnOn(t, nina)

	zoneChange := time.Date(2026, time.November, 24, 6, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.November, 25, 8, 1, 0, 0, time.UTC)

	for now := pulsetest.At(24, 9, 0); !now.After(end); now = now.Add(time.Minute) {
		zone := pulsetest.Tbilisi()
		if !now.Before(zoneChange) {
			zone = pulsetest.Berlin()
		}

		pulse.Clock.Set(now)
		require.NoError(t, pulse.Tick.Run(pulsetest.ActingIn(nina, zone)))
	}

	var fired []time.Time

	require.NoError(t, pulse.DB.WithTx(pulsetest.ActingAs(nina), func(tx *sqlx.Tx) error {
		return tx.Select(
			&fired,
			"SELECT occurrence_at FROM rounds WHERE checkin_name = 'Morning' ORDER BY occurrence_at",
		)
	}))

	for index := range fired {
		fired[index] = fired[index].UTC()
	}

	assert.Equal(t, []time.Time{
		time.Date(2026, time.November, 24, 5, 0, 0, 0, time.UTC),
		time.Date(2026, time.November, 24, 8, 0, 0, 0, time.UTC),
		time.Date(2026, time.November, 25, 8, 0, 0, 0, time.UTC),
	}, fired)
}

// PULSE-SC-010: Of "Gym" at 07:30 and "Run" and "Bike" at 08:00, a tick at 08:00
// fires "Bike", then "Run", and marks "Gym" at 07:30.
func TestOnlyTheLatestOccurrenceFires(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	pulse.TurnOn(t, nina)

	mood := pulse.QuestionID(t, "How is your mood?")
	gym := ""

	require.NoError(t, pulse.DB.WithTx(pulsetest.ActingAs(nina), func(tx *sqlx.Tx) error {
		for _, item := range []struct {
			name  string
			clock model.Clock
		}{{"Gym", model.Clock{Hour: 7, Minute: 30}}, {"Run", model.Clock{Hour: 8}}, {"Bike", model.Clock{Hour: 8}}} {
			checkin := &model.Checkin{
				Name:        item.name,
				LocalTimes:  model.Clocks{item.clock},
				Weekdays:    model.Weekdays{},
				Enabled:     true,
				FiresAfter:  pulsetest.At(24, 7, 0),
				QuestionIDs: model.IDs{mood},
			}
			if err := repository.NewCheckin(tx).Insert(t.Context(), checkin); err != nil {
				return fmt.Errorf("storing the check-in: %w", err)
			}

			if item.name == "Gym" {
				gym = checkin.ID
			}
		}

		return nil
	}))

	pulse.TickAt(t, nina, pulsetest.At(24, 8, 0))

	assert.Equal(t, []string{"Bike check-in, 1 question.", "Run check-in, 1 question."},
		texts(pulse.Messenger.Sent()))

	var firesAfter time.Time

	require.NoError(t, pulse.DB.WithTx(pulsetest.ActingAs(nina), func(tx *sqlx.Tx) error {
		return tx.Get(&firesAfter, "SELECT fires_after FROM checkins WHERE id = $1", gym)
	}))
	assert.True(t, firesAfter.Equal(pulsetest.At(24, 7, 30)))
}

// PULSE-SC-014: A round that fires expires the round of 09:00, edits its waiting
// question, and releases its claim.
func TestAFiringRoundExpiresTheEarlierRound(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	pulse.TurnOn(t, nina)
	pulse.TickAt(t, nina, pulsetest.At(24, 9, 0))

	require.Empty(t, pulse.Press(t, pulsetest.At(24, 9, 5), "m1", round.ValueStart))

	for _, id := range []message.ID{"m2", "m3", "m4"} {
		require.Empty(t, pulse.Press(t, pulsetest.At(24, 9, 6), id, round.ValueSkip))
	}

	require.Equal(t, "Weight (kg)", pulse.Messenger.Text("m5"))

	pulse.TickAt(t, nina, pulsetest.At(24, 13, 0))

	assert.Equal(t, "Morning check-in: expired.", pulse.Messenger.Text("m1"))
	assert.Equal(t, "Weight (kg) expired", pulse.Messenger.Text("m5"))
	assert.True(t, pulse.Messenger.Released("m5"))
	assert.Equal(t, 0, pulsetest.Count(t, pulse.DB, nina, "SELECT count(*) FROM question_values"))
	assert.Equal(t, 0, pulsetest.Count(t, pulse.DB, nina,
		"SELECT count(*) FROM prompts p JOIN rounds r ON r.id = p.round_id "+
			"WHERE r.checkin_name = 'Morning' AND p.state IN ('pending', 'waiting')"))
	assert.Equal(t, "Midday check-in, 1 question.", pulse.Messenger.Last().Message.Text)
}

// PULSE-SC-015: Two ticks of one occurrence that overlap store one round and send one
// start prompt.
func TestTwoTicksOfOneOccurrenceFireOneRound(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	pulse.TurnOn(t, nina)

	ticks := []*job.Tick{
		job.NewTick(
			pulse.DB,
			pulse.Rounds,
			func() time.Time { return pulsetest.At(24, 9, 0).Add(10 * time.Second) },
		),
		job.NewTick(
			pulse.DB,
			pulse.Rounds,
			func() time.Time { return pulsetest.At(24, 9, 0).Add(20 * time.Second) },
		),
	}

	var wait sync.WaitGroup

	for _, tick := range ticks {
		wait.Go(func() {
			assert.NoError(t, tick.Run(pulsetest.ActingAs(nina)))
		})
	}

	wait.Wait()

	assert.Equal(t, 1, pulsetest.Count(t, pulse.DB, nina, "SELECT count(*) FROM rounds"))
	assert.Len(t, pulse.Messenger.Sent(), 1)
}

// PULSE-SC-017: The start prompt names the check-in and the count, and a check-in
// whose every question is archived fires no round.
func TestTheStartPromptNamesTheCheckin(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	pulse.TurnOn(t, nina)

	pulse.TickAt(t, nina, pulsetest.At(24, 13, 0))
	assert.Equal(t, "Midday check-in, 1 question.", pulse.Messenger.Last().Message.Text)

	pulse.TickAt(t, nina, pulsetest.At(24, 22, 0))
	assert.Equal(t, message.Message{
		Text: "Evening check-in, 7 questions.",
		Buttons: [][]message.Button{{
			{Label: "Start", Value: "start"},
			{Label: "Decline", Value: "decline"},
		}},
	}, pulse.Messenger.Last().Message)

	pulsetest.Exec(
		t,
		pulse.DB,
		nina,
		"UPDATE questions SET archived_at = now() WHERE prompt IN ('How was your week?', 'A note on the week?')",
	)

	pulse.TickAt(t, nina, time.Date(2026, time.November, 29, 18, 0, 0, 0, pulsetest.Tbilisi()))
	assert.Equal(
		t,
		0,
		pulsetest.Count(
			t,
			pulse.DB,
			nina,
			"SELECT count(*) FROM rounds WHERE checkin_name = 'Weekly'",
		),
	)
}
