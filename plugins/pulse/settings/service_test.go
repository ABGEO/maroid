package settings_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
	"github.com/abgeo/maroid/plugins/pulse/settings"
)

// PULSE-SC-001: The first turn-on copies 5 check-ins and 12 questions, and a turn-off
// and a second turn-on copy nothing more.
func TestTheFirstTurnOnCopiesTheDefaultsOnce(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	ctx := pulsetest.ActingAs(pulsetest.Nina)
	absent := settings.Absent()

	first, err := pulse.Settings.Put(ctx, true, nil, &absent)
	require.NoError(t, err)
	require.NotNil(t, first.SeededAt)

	counts := func() (int, int) {
		return pulsetest.Count(t, pulse.DB, pulsetest.Nina, "SELECT count(*) FROM checkins"),
			pulsetest.Count(t, pulse.DB, pulsetest.Nina, "SELECT count(*) FROM questions")
	}

	checkins, questions := counts()
	assert.Equal(t, 5, checkins)
	assert.Equal(t, 12, questions)

	pulse.Clock.Set(pulsetest.At(24, 7, 0))

	off, err := pulse.Settings.Put(ctx, false, nil, &first.UpdatedAt)
	require.NoError(t, err)

	pulse.Clock.Set(pulsetest.At(24, 8, 0))

	again, err := pulse.Settings.Put(ctx, true, nil, &off.UpdatedAt)
	require.NoError(t, err)

	checkins, questions = counts()
	assert.Equal(t, 5, checkins)
	assert.Equal(t, 12, questions)
	assert.True(t, first.SeededAt.Equal(*again.SeededAt))
}

// PULSE-SC-001: With no row, a read answers pulse off at the validator "0" and writes
// nothing, and a write with another validator is refused.
func TestNoRowReadsOffAtTheZeroValidator(t *testing.T) {
	t.Parallel()

	pulse := pulsetest.New(t)
	ctx := pulsetest.ActingAs(pulsetest.Nina)

	current, err := pulse.Settings.Get(ctx)
	require.NoError(t, err)
	assert.False(t, current.Enabled)
	assert.True(t, current.UpdatedAt.Equal(time.Unix(0, 0)))
	assert.Equal(
		t,
		0,
		pulsetest.Count(t, pulse.DB, pulsetest.Nina, "SELECT count(*) FROM settings"),
	)

	stale := time.Unix(5, 0)
	_, err = pulse.Settings.Put(ctx, true, nil, &stale)
	require.Error(t, err)
	assert.Equal(
		t,
		0,
		pulsetest.Count(t, pulse.DB, pulsetest.Nina, "SELECT count(*) FROM questions"),
	)
}
