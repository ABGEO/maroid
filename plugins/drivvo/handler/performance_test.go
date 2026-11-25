package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/widget"
)

// largeSnapshot holds "Hatchback" with 2,000 entries, a third of them expenses with three
// instalments each, and its four reminders.
func largeSnapshot() model.Snapshot {
	snapshot := pagesSnapshot()
	snapshot.LogEntries = nil

	start := time.Date(2021, 1, 1, 8, 0, 0, 0, time.UTC)

	for index := range int32(2000) {
		occurredAt := start.Add(time.Duration(index) * 18 * time.Hour)
		odometer := float64(10000 + index*40)

		if index%3 != 0 {
			snapshot.LogEntries = append(snapshot.LogEntries,
				refuelOf(1000+index, 11, occurredAt, odometer, 30, index%2 == 0))

			continue
		}

		day := model.DayOf(occurredAt, time.UTC)
		snapshot.LogEntries = append(snapshot.LogEntries, model.LogEntry{
			Kind: model.KindExpense, SourceID: 1000 + index, VehicleSource: 11,
			OccurredAt: occurredAt, Odometer: odometer, Amount: amountOf(300),
			Lines: []model.Line{{Position: 1, Item: 7, Amount: model.AmountOf(300)}},
			Instalments: []model.Instalment{
				{Number: 1, DueOn: day, Amount: model.AmountOf(100)},
				{Number: 2, DueOn: day.AddDate(0, 1, 0), Amount: model.AmountOf(100)},
				{Number: 3, DueOn: day.AddDate(0, 2, 0), Amount: model.AmountOf(100)},
			},
		})
	}

	return snapshot
}

// DRIVVO-SC-023: With 2,000 entries, each read of the overview, the history, the reminders,
// and the widget answers in less than 1 second.
func TestEachReadAnswersWithinOneSecond(t *testing.T) {
	t.Parallel()

	rig := newPageRig(t, largeSnapshot())
	car := widget.NewCar(rig.figures)

	reads := map[string]func(){
		"widget": func() {
			_, err := car.Load(memberContext(t.Context()))
			require.NoError(t, err)
		},
	}

	for _, path := range []string{
		rig.vehiclePath(hatchback, "/overview"),
		rig.vehiclePath(hatchback, "/log-entries"),
		rig.vehiclePath(hatchback, "/reminders"),
	} {
		reads[path] = func() {
			require.Equal(t, http.StatusOK, rig.get(t, path).Code, path)
		}
	}

	for name, read := range reads {
		for range 10 {
			started := time.Now()

			read()
			assert.Less(t, time.Since(started), time.Second, name)
		}
	}
}
