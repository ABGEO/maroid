package figures_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/drivvo/figures"
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

const (
	oilChange                            = "Oil change"
	filter                               = "Filter"
	insurance                            = "Insurance"
	litresPer100 model.ConsumptionFormat = 4
	kmPerLitre   model.ConsumptionFormat = 1
)

func day(year int, month time.Month, date int) time.Time {
	return time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
}

func amount(value float64) *model.Amount {
	return new(model.AmountOf(value))
}

func period(from, to time.Time) model.Period {
	return model.Period{From: &from, To: &to}
}

func refuel(id string, odometer, volume float64, full bool, at time.Time) figures.Entry {
	return figures.Entry{
		ID:         id,
		Kind:       model.KindRefuel,
		OccurredAt: at,
		Odometer:   odometer,
		Amount:     amount(volume * 3),
		Lines: []figures.Line{{
			Name: "Petrol", Amount: model.AmountOf(volume * 3), Fuel: true,
			Volume: volume, FullTank: full,
		}},
	}
}

// threeRefuels are a full refuel at 10,000 km, 30 L not full at 10,450 km, and 20 L full at
// 10,800 km.
func threeRefuels() []figures.Entry {
	return []figures.Entry{
		refuel("r1", 10000, 40, true, time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)),
		refuel("r2", 10450, 30, false, time.Date(2026, 11, 8, 8, 0, 0, 0, time.UTC)),
		refuel("r3", 10800, 20, true, time.Date(2026, 11, 15, 8, 0, 0, 0, time.UTC)),
	}
}

// DRIVVO-SC-037: Each instalment shares its amount among the types in proportion to the
// line amounts.
func TestAnInstalmentSharesItsAmountAmongTheTypes(t *testing.T) {
	t.Parallel()

	entry := figures.Entry{
		ID:         "m1",
		Kind:       model.KindMaintenance,
		OccurredAt: time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC),
		Amount:     amount(300),
		Lines: []figures.Line{
			{Item: "t1", Name: oilChange, Amount: model.AmountOf(100)},
			{Item: "t2", Name: filter, Amount: model.AmountOf(200)},
		},
		Instalments: []model.Instalment{
			{Number: 1, DueOn: day(2026, 1, 15), Amount: model.AmountOf(150)},
			{Number: 2, DueOn: day(2026, 2, 15), Amount: model.AmountOf(150)},
		},
	}

	overview := figures.Summarize(
		[]figures.Entry{entry},
		period(day(2026, 1, 1), day(2026, 1, 31)),
		model.Tbilisi(),
		day(2026, 11, 25),
		litresPer100,
	)

	assert.Equal(t, []figures.TypeSpend{
		{Kind: model.KindMaintenance, Name: filter, Amount: model.AmountOf(100)},
		{Kind: model.KindMaintenance, Name: oilChange, Amount: model.AmountOf(50)},
	}, overview.Types)
	assert.Equal(t, model.AmountOf(150), overview.Totals.Maintenance)
}

// DRIVVO-SC-038: The spend counts each instalment in the month of its due day, and a month
// with no charge counts zero. The income stays apart from the spend.
func TestTheOverviewCountsEachMonth(t *testing.T) {
	t.Parallel()

	entries := []figures.Entry{
		{
			ID:         "e1",
			Kind:       model.KindExpense,
			OccurredAt: time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC),
			Amount:     amount(600),
			Lines:      []figures.Line{{Item: "x1", Name: insurance, Amount: model.AmountOf(600)}},
			Instalments: []model.Instalment{
				{Number: 1, DueOn: day(2026, 1, 10), Amount: model.AmountOf(200)},
				{Number: 2, DueOn: day(2026, 2, 10), Amount: model.AmountOf(200)},
				{Number: 3, DueOn: day(2026, 3, 10), Amount: model.AmountOf(200)},
			},
		},
		refuel("r1", 10000, 40, true, time.Date(2026, 3, 5, 8, 0, 0, 0, time.UTC)),
		{
			ID:         "i1",
			Kind:       model.KindIncome,
			OccurredAt: time.Date(2026, 3, 20, 8, 0, 0, 0, time.UTC),
			Amount:     amount(50),
		},
	}

	overview := figures.Summarize(
		entries,
		period(day(2026, 1, 1), day(2026, 4, 30)),
		model.Tbilisi(),
		day(2026, 11, 25),
		litresPer100,
	)

	assert.Equal(t, model.AmountOf(120), overview.Totals.Fuel)
	assert.Equal(t, model.AmountOf(600), overview.Totals.Expense)
	assert.Equal(t, model.AmountOf(720), overview.Totals.Spend())
	assert.Equal(t, model.AmountOf(50), overview.Totals.Income)

	two := model.AmountOf(200)
	assert.Equal(t, []figures.Month{
		{Month: day(2026, 1, 1), Expense: two},
		{Month: day(2026, 2, 1), Expense: two},
		{
			Month:   day(2026, 3, 1),
			Expense: two, Fuel: model.AmountOf(120), Income: model.AmountOf(50),
		},
		{Month: day(2026, 4, 1)},
	}, overview.Months)
}

// DRIVVO-SC-038: With all time, the months run from the month of the first charge to the
// month of today.
func TestAllTimeRunsFromTheFirstChargeToToday(t *testing.T) {
	t.Parallel()

	overview := figures.Summarize(
		[]figures.Entry{refuel("r1", 10000, 40, true, time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC))},
		model.Period{},
		model.Tbilisi(),
		day(2026, 11, 25),
		litresPer100,
	)

	require.Len(t, overview.Months, 3)
	assert.Equal(t, day(2026, 9, 1), overview.Months[0].Month)
	assert.Equal(t, day(2026, 11, 1), overview.Months[2].Month)
}

// DRIVVO-SC-039: The distance is the highest minus the lowest odometer of the period, and
// gives the spend per distance. One entry gives no distance and no value per distance.
func TestTheDistanceGivesTheSpendPerDistance(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	fuel := refuel("r1", 10000, 300, false, at)
	service := figures.Entry{
		ID: "m1", Kind: model.KindMaintenance, OccurredAt: at.AddDate(0, 1, 0),
		Odometer: 13000, Amount: amount(600),
	}
	year := period(day(2026, 1, 1), day(2026, 12, 31))

	overview := figures.Summarize(
		[]figures.Entry{fuel, service}, year, model.Tbilisi(), day(2026, 11, 25), litresPer100,
	)

	require.NotNil(t, overview.Distance)
	assert.InDelta(t, 3000, *overview.Distance, 0.001)
	assert.InDelta(t, 0.30, *overview.FuelCostPerDistance, 0.0001)
	assert.InDelta(t, 0.50, *overview.TotalCostPerDistance, 0.0001)

	alone := figures.Summarize(
		[]figures.Entry{fuel}, year, model.Tbilisi(), day(2026, 11, 25), litresPer100,
	)

	assert.Nil(t, alone.Distance)
	assert.Nil(t, alone.FuelCostPerDistance)
	assert.Nil(t, alone.TotalCostPerDistance)
}

// DRIVVO-SC-040: The consumption sums the volume after the first full refuel up to the next
// full one, in the shape of the format. One full refuel gives none.
func TestTheConsumptionRunsFromFullTankToFullTank(t *testing.T) {
	t.Parallel()

	parts := figures.Parts(threeRefuels())
	all := model.Period{}

	perHundred := figures.Consumption(parts, all, model.Tbilisi(), litresPer100)
	require.NotNil(t, perHundred)
	assert.InDelta(t, 6.25, *perHundred, 0.0001)
	assert.Equal(t, model.ShapeVolumePer100, litresPer100.Shape())

	perLitre := figures.Consumption(parts, all, model.Tbilisi(), kmPerLitre)
	require.NotNil(t, perLitre)
	assert.InDelta(t, 16.0, *perLitre, 0.0001)
	assert.Equal(t, model.ShapeDistancePerVolume, kmPerLitre.Shape())

	alone := figures.Parts(threeRefuels()[:1])
	assert.Nil(t, figures.Consumption(alone, all, model.Tbilisi(), litresPer100))
}

// DRIVVO-SC-040: A part counts in the period of its last refuel.
func TestAPartCountsInThePeriodOfItsLastRefuel(t *testing.T) {
	t.Parallel()

	parts := figures.Parts(threeRefuels())

	assert.Nil(t, figures.Consumption(
		parts, period(day(2026, 11, 1), day(2026, 11, 10)), model.Tbilisi(), litresPer100,
	))
	assert.NotNil(t, figures.Consumption(
		parts, period(day(2026, 11, 15), day(2026, 11, 15)), model.Tbilisi(), litresPer100,
	))
}

// DRIVVO-SC-041: A missed refuel skips the part, so the refuel that ends it holds no
// consumption and the average is none.
func TestAMissedRefuelSkipsThePart(t *testing.T) {
	t.Parallel()

	entries := threeRefuels()
	entries[1].MissedRefuel = true

	parts := figures.Parts(entries)

	assert.Empty(t, figures.PartConsumption(parts, litresPer100))
	assert.Nil(t, figures.Consumption(parts, model.Period{}, model.Tbilisi(), litresPer100))

	counted := figures.PartConsumption(figures.Parts(threeRefuels()), litresPer100)
	assert.InDelta(t, 6.25, counted["r3"], 0.0001)
}

// DRIVVO-SC-036: A moment counts on its day in the zone of the member.
func TestAMomentCountsOnItsDayInTheZone(t *testing.T) {
	t.Parallel()

	late := refuel("r1", 10000, 40, true, time.Date(2026, 10, 31, 21, 30, 0, 0, time.UTC))
	november := model.Period{From: new(day(2026, 11, 1))}

	overview := figures.Summarize(
		[]figures.Entry{late}, november, model.Tbilisi(), day(2026, 11, 25), litresPer100,
	)

	assert.Equal(t, model.AmountOf(120), overview.Totals.Fuel)
	assert.Equal(t, day(2026, 11, 1), overview.Months[0].Month)
}

func oilReminder() figures.Reminder {
	return figures.Reminder{
		ID:             "rem1",
		Kind:           model.KindMaintenance,
		Item:           "t1",
		Type:           oilChange,
		DueOn:          new(day(2026, 1, 1)),
		DueOdometer:    new(140000.0),
		Repeats:        true,
		RepeatEvery:    new(int32(12)),
		RepeatUnit:     new(model.RepeatMonth),
		RepeatDistance: new(10000.0),
	}
}

// DRIVVO-SC-048: A reminder that repeats counts from the latest entry that names its type,
// and with no such entry keeps its own due point.
func TestARepeatedReminderCountsFromTheLatestEntryOfItsType(t *testing.T) {
	t.Parallel()

	change := figures.Entry{
		ID: "m1", Kind: model.KindMaintenance,
		OccurredAt: time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC),
		Odometer:   150000, Amount: amount(100),
		Lines: []figures.Line{{Item: "t1", Name: oilChange, Amount: model.AmountOf(100)}},
	}

	due := figures.DueReminders(
		[]figures.Reminder{oilReminder()},
		[]figures.Entry{change},
		150000,
		day(2026, 11, 25),
		model.Tbilisi(),
	)

	require.Len(t, due, 1)
	assert.Equal(t, day(2027, 3, 1), *due[0].DueOn)
	assert.InDelta(t, 160000, *due[0].DueOdometer, 0.001)

	own := figures.DueReminders(
		[]figures.Reminder{oilReminder()}, nil, 0, day(2026, 11, 25), model.Tbilisi(),
	)

	assert.Equal(t, day(2026, 1, 1), *own[0].DueOn)
	assert.InDelta(t, 140000, *own[0].DueOdometer, 0.001)
}

// DRIVVO-SC-049: A reminder is due when the day reaches its due day, or when the latest
// odometer reaches its due odometer.
func TestAReminderIsDueByDayOrByOdometer(t *testing.T) {
	t.Parallel()

	byDistance := figures.Reminder{ID: "a", Type: oilChange, DueOdometer: new(160000.0)}
	byEither := figures.Reminder{
		ID: "b", Type: filter, DueOn: new(day(2027, 1, 4)), DueOdometer: new(159000.0),
	}

	due := figures.DueReminders(
		[]figures.Reminder{byDistance, byEither}, nil, 160250, day(2026, 11, 25), model.Tbilisi(),
	)

	require.Len(t, due, 2)

	for _, reminder := range due {
		assert.True(t, reminder.Due, reminder.ID)
	}

	assert.Equal(t, "b", due[0].ID)
	assert.Equal(t, 40, *due[0].DaysLeft)
	assert.InDelta(t, -250, *due[1].DistanceLeft, 0.001)
}

// DRIVVO-SC-047: A reminder that applies one time keeps its own day, and counts the days
// left.
func TestAOneTimeReminderKeepsItsOwnDay(t *testing.T) {
	t.Parallel()

	once := figures.Reminder{
		ID:    "a",
		Kind:  model.KindExpense,
		Type:  insurance,
		DueOn: new(day(2026, 12, 7)),
	}

	due := figures.DueReminders(
		[]figures.Reminder{once},
		nil,
		0,
		day(2026, 11, 25),
		model.Tbilisi(),
	)

	require.Len(t, due, 1)
	assert.Equal(t, 12, *due[0].DaysLeft)
	assert.False(t, due[0].Due)
	assert.Nil(t, due[0].DistanceLeft)
}

// DRIVVO-SC-050: The due reminders come first, then the others by the days left, then the
// ones with no due day by the distance left.
func TestTheRemindersFollowTheirOrder(t *testing.T) {
	t.Parallel()

	today := day(2026, 11, 25)
	reminders := []figures.Reminder{
		{ID: "odometer", Type: "D", DueOdometer: new(11000.0)},
		{ID: "thirty", Type: "C", DueOn: new(today.AddDate(0, 0, 30))},
		{ID: "due", Type: "A", DueOn: new(today.AddDate(0, 0, -1))},
		{ID: "five", Type: "B", DueOn: new(today.AddDate(0, 0, 5))},
	}

	due := figures.DueReminders(reminders, nil, 10000, today, model.Tbilisi())

	order := make([]string, 0, len(due))
	for _, reminder := range due {
		order = append(order, reminder.ID)
	}

	assert.Equal(t, []string{"due", "five", "thirty", "odometer"}, order)
}

// DRIVVO-SC-049: A reminder is due on its due day, and not the day before.
func TestAReminderIsDueOnItsDueDay(t *testing.T) {
	t.Parallel()

	today := day(2026, 11, 25)
	reminders := []figures.Reminder{
		{ID: "today", Type: "A", DueOn: new(today)},
		{ID: "tomorrow", Type: "B", DueOn: new(today.AddDate(0, 0, 1))},
	}

	due := figures.DueReminders(reminders, nil, 0, today, model.Tbilisi())

	require.Len(t, due, 2)
	assert.True(t, due[0].Due)
	assert.Equal(t, 0, *due[0].DaysLeft)
	assert.False(t, due[1].Due)
}
