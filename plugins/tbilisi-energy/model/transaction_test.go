package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/tbilisi-energy/model"
)

// GASACC-SC-012: A run at 2026-11-01 00:30 in Tbilisi, which is 2026-10-31 20:30 UTC,
// reads from 2026-10-01 to 2026-10-31.
func TestThePreviousMonthIsAMonthOfTbilisi(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.October, 31, 20, 30, 0, 0, time.UTC)

	period := model.PreviousMonth(now)

	require.Equal(t, "2026-10-01", period.From.Format(time.DateOnly))
	require.Equal(t, "2026-10-31", period.To.Format(time.DateOnly))
}

// GASACC-SC-012: The previous month of January is the December of the year before.
func TestThePreviousMonthCrossesTheYear(t *testing.T) {
	t.Parallel()

	now := time.Date(2027, time.January, 15, 12, 0, 0, 0, model.Tbilisi())

	period := model.PreviousMonth(now)

	require.Equal(t, "2026-12-01", period.From.Format(time.DateOnly))
	require.Equal(t, "2026-12-31", period.To.Format(time.DateOnly))
}

// GASACC-SC-023: A payment carries the meter value 0, so it is no reading.
func TestAReadingCarriesAMeterValue(t *testing.T) {
	t.Parallel()

	reading := model.Transaction{MeterValue: 1532}
	payment := model.Transaction{MeterValue: 0}

	require.True(t, reading.IsReading())
	require.False(t, payment.IsReading())
}
