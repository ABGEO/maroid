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

// GASACC-SC-023: A reading is the operation 125. A payment and a control reading,
// which carries a meter value, are no reading.
func TestAReadingIsTheOperationOfARealReading(t *testing.T) {
	t.Parallel()

	reading := model.Transaction{TypeID: 125, MeterValue: 1532, Consumption: 21}
	payment := model.Transaction{TypeID: 196}
	control := model.Transaction{TypeID: 172, MeterValue: 982}

	require.True(t, reading.IsReading())
	require.False(t, payment.IsReading())
	require.False(t, control.IsReading())
}

// GASACC-SC-019: A period splits into its calendar months, the first and the last cut
// to the period.
func TestAPeriodSplitsIntoItsMonths(t *testing.T) {
	t.Parallel()

	period := model.Period{
		From: time.Date(2025, time.October, 15, 0, 0, 0, 0, model.Tbilisi()),
		To:   time.Date(2026, time.January, 10, 0, 0, 0, 0, model.Tbilisi()),
	}

	split := period.Months()

	months := make([]string, 0, len(split))
	for _, month := range split {
		months = append(
			months,
			month.From.Format(time.DateOnly)+".."+month.To.Format(time.DateOnly),
		)
	}

	require.Equal(t, []string{
		"2025-10-15..2025-10-31",
		"2025-11-01..2025-11-30",
		"2025-12-01..2025-12-31",
		"2026-01-01..2026-01-10",
	}, months)
}

// GASACC-SC-019: A period within one month stays one period.
func TestAPeriodWithinAMonthStaysWhole(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, time.September, 5, 0, 0, 0, 0, model.Tbilisi())
	period := model.Period{From: day, To: day}

	require.Equal(t, []model.Period{period}, period.Months())
}
