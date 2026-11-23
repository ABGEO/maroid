package command_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
	"github.com/abgeo/maroid/plugins/pulse/telegram/command"
)

const nina = pulsetest.Nina

type fakePauser struct {
	until *time.Time
}

func (f *fakePauser) Pause(_ context.Context, until time.Time) error {
	f.until = &until

	return nil
}

func (f *fakePauser) Resume(_ context.Context) error {
	f.until = nil

	return nil
}

// PULSE-SC-005: A word, a past date, and 400 days answer the format and change
// nothing, and /resume ends the pause.
func TestAWrongPauseAnswersTheFormat(t *testing.T) {
	t.Parallel()

	now := pulsetest.At(28, 10, 0)
	paused := now.Add(48 * time.Hour)
	pauser := &fakePauser{until: &paused}
	pause := command.NewPause(pauser, nil, func() time.Time { return now })
	ctx := pulsetest.ActingAs(nina)

	for _, text := range []string{"/pause yesterday", "/pause 2025-01-01", "/pause 400d"} {
		answer, err := pause.Reply(ctx, text)
		require.NoError(t, err)
		assert.Equal(
			t,
			"Send /pause with a date, 2026-12-01, or a duration, 3d or 12h.",
			answer,
			text,
		)
		assert.True(t, pauser.until.Equal(paused), text)
	}

	answer, err := command.NewResume(pauser, nil).Reply(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Check-ins resumed.", answer)
	assert.Nil(t, pauser.until)
}

// PULSE-SC-004: /pause 2026-12-01 stops every check-in until 00:00 of that day in
// the zone of Nina.
func TestAPauseStopsTheCheckins(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	pulse.TurnOn(t, nina)

	pause := command.NewPause(pulse.Settings, nil, pulse.Clock.Now)
	pulse.Clock.Set(pulsetest.At(28, 8, 0))

	answer, err := pause.Reply(pulsetest.ActingAs(nina), "/pause 2026-12-01")
	require.NoError(t, err)
	assert.Equal(t, "Check-ins paused until 1 Dec 2026, 00:00.", answer)

	current, err := pulse.Settings.Get(pulsetest.ActingAs(nina))
	require.NoError(t, err)
	assert.Equal(
		t,
		time.Date(2026, time.November, 30, 20, 0, 0, 0, time.UTC),
		current.PausedUntil.UTC(),
	)

	for _, day := range []int{28, 29, 30} {
		pulse.TickAt(t, nina, pulsetest.At(day, 9, 0))
	}

	assert.Empty(t, pulse.Messenger.Sent())

	pulse.TickAt(t, nina, time.Date(2026, time.December, 1, 9, 0, 0, 0, pulsetest.Tbilisi()))
	assert.Equal(t, "Morning check-in, 4 questions.", pulse.Messenger.Last().Message.Text)
}

// PULSE-SC-004: /pause 3d at 10:00 keeps "Morning" of the third day from firing.
func TestAPauseForADurationSkipsItsOccurrences(t *testing.T) {
	t.Parallel()

	short := pulsetest.New(t)
	short.TurnOn(t, nina)
	short.Clock.Set(pulsetest.At(28, 10, 0))

	_, err := command.NewPause(short.Settings, nil, short.Clock.Now).
		Reply(pulsetest.ActingAs(nina), "/pause 3d")
	require.NoError(t, err)

	current, err := short.Settings.Get(pulsetest.ActingAs(nina))
	require.NoError(t, err)
	assert.True(
		t,
		current.PausedUntil.Equal(
			time.Date(2026, time.December, 1, 10, 0, 0, 0, pulsetest.Tbilisi()),
		),
	)

	december := func(hour, minute int) time.Time {
		return time.Date(2026, time.December, 1, hour, minute, 0, 0, pulsetest.Tbilisi())
	}

	short.TickAt(t, nina, december(9, 0))
	short.TickAt(t, nina, december(10, 1))

	assert.Equal(
		t,
		0,
		pulsetest.Count(
			t,
			short.DB,
			nina,
			"SELECT count(*) FROM rounds WHERE checkin_name = 'Morning'",
		),
	)
}
