package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
)

func day(year int, month time.Month, dayOfMonth int) model.Day {
	return model.Day{Year: year, Month: month, Day: dayOfMonth}
}

// MAGTIACC-SC-013: The previous day is the day of Tbilisi before the one of now, and the
// portal names it as DD/MM/YYYY.
func TestThePreviousDayIsADayOfTbilisi(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 25, 5, 0, 0, 0, time.UTC)

	previous := model.PreviousDay(now)

	assert.Equal(t, day(2026, time.November, 24), previous)
	assert.Equal(t, "24/11/2026", previous.PortalForm())
	assert.Equal(t, "2026-11-24", previous.String())
}

// MAGTIACC-SC-016: A moment of the portal belongs to the day of Tbilisi that holds it.
func TestAMomentBelongsToTheDayOfTbilisi(t *testing.T) {
	t.Parallel()

	assert.Equal(t, day(2026, time.November, 24), model.DayOf(time.UnixMilli(1795464000000)))
	assert.Equal(t, day(2026, time.November, 23), model.DayOf(time.UnixMilli(1795462200000)))
}

// MAGTIACC-SC-019: A period divides into calendar months, each cut to the period.
func TestAPeriodDividesIntoMonths(t *testing.T) {
	t.Parallel()

	period := model.Period{From: day(2025, time.November, 15), To: day(2026, time.January, 10)}

	assert.Equal(t, []model.Period{
		{From: day(2025, time.November, 15), To: day(2025, time.November, 30)},
		{From: day(2025, time.December, 1), To: day(2025, time.December, 31)},
		{From: day(2026, time.January, 1), To: day(2026, time.January, 10)},
	}, period.Months())

	reversed := model.Period{From: day(2026, time.February, 1), To: day(2026, time.January, 1)}
	assert.Empty(t, reversed.Months())
}

// MAGTIACC-SC-021: The days left count the days of Tbilisi from today to the renewal.
func TestTheDaysLeftCountTheDaysOfTbilisi(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 25, 5, 0, 0, 0, time.UTC)
	renewal := time.Date(2026, time.December, 13, 1, 50, 47, 0, model.Tbilisi())

	assert.Equal(t, 18, model.DaysUntil(now, renewal))
	assert.Equal(t, 0, model.DaysUntil(now, now.Add(-time.Hour)))
}

// MAGTIACC-SC-025: A month reads from 2025-12 and counts forward.
func TestAMonthParsesAndCounts(t *testing.T) {
	t.Parallel()

	month, err := model.ParseMonth("2025-12")
	require.NoError(t, err)

	assert.Equal(t, "2026-11", month.AddMonths(11).String())
	assert.True(t, month.Next().After(month))
}

// MAGTIACC-SC-005: A stored session holds every cookie of the sign in, and one with no
// cookie is no connection.
func TestASessionRoundTripsThroughTheSecret(t *testing.T) {
	t.Parallel()

	stored, err := model.Session{Cookies: map[string]string{"PHPSESSID": "K"}}.MarshalSecret()
	require.NoError(t, err)

	session, err := model.ParseSession(stored)
	require.NoError(t, err)
	assert.Equal(t, "K", session.Cookies["PHPSESSID"])

	_, err = model.ParseSession(`{"cookies":{}}`)
	require.ErrorIs(t, err, errs.ErrNotConnected)
}
