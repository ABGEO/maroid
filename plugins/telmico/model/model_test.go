package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/telmico/model"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, model.Tbilisi())
}

func charge(on time.Time, meter, usage, amount float64) model.Transaction {
	return model.Transaction{
		Date:       on,
		MeterValue: new(meter),
		Usage:      new(usage),
		Amount:     new(amount),
	}
}

// ELECACC-SC-023: The two charges of one date form one reading, with the sums of the
// usage and of the charge, and the highest meter value. A payment and the first
// reading of a new meter form none.
func TestTheChargesOfOneDateFormOneReading(t *testing.T) {
	t.Parallel()

	transactions := []model.Transaction{
		charge(date(2026, time.March, 29), 1150, 254.9, 48.57),
		charge(date(2026, time.April, 28), 1163, 13.2, 2.52),
		charge(date(2026, time.April, 28), 1317, 153.3, 36.87),
		{Date: date(2026, time.April, 1), Amount: new(-48.57)},
		{Date: date(2024, time.October, 2), MeterValue: new(3.0), Usage: new(0.0)},
	}

	readings := model.ReadingsOf(transactions)

	require.Len(t, readings, 2)

	assert.Equal(t, date(2026, time.April, 28), readings[0].Date)
	assert.InDelta(t, 166.5, readings[0].Usage, 1e-9)
	assert.InDelta(t, 39.39, readings[0].Charge, 1e-9)
	assert.InDelta(t, 1317.0, readings[0].MeterValue, 0)

	assert.Equal(t, date(2026, time.March, 29), readings[1].Date)
	assert.InDelta(t, 254.9, readings[1].Usage, 1e-9)
}

// ELECACC-SC-013: The previous month is the month of Tbilisi before the one of now, and
// the portal names it with no leading zero.
func TestThePreviousMonthIsAMonthOfTbilisi(t *testing.T) {
	t.Parallel()

	runAt := time.Date(2026, time.October, 31, 20, 30, 0, 0, time.UTC)

	previous := model.PreviousMonth(runAt)

	assert.Equal(t, model.Month{Year: 2026, Month: time.October}, previous)
	assert.Equal(t, "2026-10", previous.String())
	assert.Equal(t, "2026-10", previous.PortalForm())
	assert.Equal(t, "2026-9", model.Month{Year: 2026, Month: time.September}.PortalForm())
	assert.Equal(t, date(2026, time.October, 1), previous.First())

	january := model.PreviousMonth(time.Date(2026, time.February, 10, 12, 0, 0, 0, time.UTC))
	assert.Equal(t, model.Month{Year: 2026, Month: time.January}, january)

	december := model.PreviousMonth(time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC))
	assert.Equal(t, model.Month{Year: 2025, Month: time.December}, december)
}

// ELECACC-SC-019: A period from 2025-10 to 2026-09 holds 12 months, in order.
func TestAPeriodListsItsMonths(t *testing.T) {
	t.Parallel()

	from, err := model.ParseMonth("2025-10")
	require.NoError(t, err)

	to, err := model.ParseMonth("2026-09")
	require.NoError(t, err)

	months := model.Period{From: from, To: to}.Months()

	require.Len(t, months, 12)
	assert.Equal(t, model.Month{Year: 2025, Month: time.October}, months[0])
	assert.Equal(t, model.Month{Year: 2026, Month: time.January}, months[3])
	assert.Equal(t, model.Month{Year: 2026, Month: time.September}, months[11])

	assert.Empty(t, model.Period{From: to, To: from}.Months())
	assert.True(t, to.After(from))
	assert.False(t, from.After(to))
}

// ELECACC-SC-020: A month that does not parse fails.
func TestABrokenMonthFails(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"2026-13", "2026-9", "09-2026", "2026-09-01", ""} {
		_, err := model.ParseMonth(raw)
		assert.Error(t, err, raw)
	}
}

// ELECACC-SC-005: The session goes to the secret store as JSON, and comes back whole.
func TestTheSessionRoundTripsThroughTheSecret(t *testing.T) {
	t.Parallel()

	stored, err := model.Session{Cookie: "K", CSRFToken: "T"}.MarshalSecret()
	require.NoError(t, err)
	assert.JSONEq(t, `{"cookie":"K","csrf_token":"T"}`, stored)

	session, err := model.ParseSession(stored)
	require.NoError(t, err)
	assert.Equal(t, model.Session{Cookie: "K", CSRFToken: "T"}, session)

	_, err = model.ParseSession(`{"cookie":""}`)
	require.Error(t, err)

	_, err = model.ParseSession("not json")
	require.Error(t, err)
}
