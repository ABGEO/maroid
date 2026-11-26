package widget_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/drivvo/figures"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/service"
	"github.com/abgeo/maroid/plugins/drivvo/widget"
)

const vehicleID = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e11"

type fixedCar struct {
	summary service.CarSummary
	found   bool
}

func (f fixedCar) Car(context.Context) (service.CarSummary, bool, error) {
	return f.summary, f.found, nil
}

func inTbilisi(t *testing.T) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingTimeZone(t.Context(), model.Tbilisi())
}

func hatchback() service.CarSummary {
	return service.CarSummary{
		Vehicle: model.Vehicle{
			ID:           vehicleID,
			Name:         "Hatchback",
			DistanceUnit: model.DistanceKilometre,
		},
		Settings: model.CarLogSettings{
			VolumeUnit: model.VolumeLitre, ConsumptionFormat: 4, Currency: "GEL",
		},
	}
}

func refuel() *model.EntryRow {
	return &model.EntryRow{
		Kind:       model.KindRefuel,
		OccurredAt: time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC),
		Amount:     new(model.AmountOf(120)),
		Lines: []model.LineRow{{
			Amount: model.AmountOf(120), UnitPrice: new(model.AmountOf(3)), Volume: new(40.0),
		}},
	}
}

// DRIVVO-SC-051: The widget answers the latest refuel, the consumption, the odometer, the
// spend of the month, and the first reminder, each with its link, and links more to the
// overview.
func TestTheWidgetAnswersFiveFacts(t *testing.T) {
	t.Parallel()

	car := hatchback()
	car.LatestRefuel = refuel()
	car.Consumption = new(6.8)
	car.Odometer = new(128450.0)
	car.MonthSpend = new(model.AmountOf(312.40))
	car.Reminder = &figures.DueReminder{Type: "Oil change", DistanceLeft: new(1200.0)}

	answer, err := widget.NewCar(fixedCar{summary: car, found: true}).Load(inTbilisi(t))
	require.NoError(t, err)
	require.NotNil(t, answer.List)

	assert.Equal(t, []pluginapi.ListItem{
		{
			Label: "Latest refuel",
			Meta:  "2026-11-20 · 40.00 L · 120.00 GEL",
			Link:  "/refuels/" + vehicleID,
		},
		{Label: "Consumption", Meta: "6.8 L/100 km", Link: "/overview/" + vehicleID},
		{Label: "Odometer", Meta: "128,450 km", Link: "/overview/" + vehicleID},
		{Label: "This month", Meta: "312.40 GEL", Link: "/overview/" + vehicleID},
		{Label: "Oil change", Meta: "in 1,200 km", Link: "/reminders/" + vehicleID},
	}, answer.List.Items)
	assert.Equal(t, "/overview/"+vehicleID, answer.List.More)
}

// DRIVVO-SC-052: An absent fact drops its item, and a vehicle with no fact answers empty.
func TestTheWidgetDropsAnAbsentFact(t *testing.T) {
	t.Parallel()

	car := hatchback()
	car.LatestRefuel = refuel()
	car.Consumption = new(6.8)
	car.Odometer = new(128450.0)

	answer, err := widget.NewCar(fixedCar{summary: car, found: true}).Load(inTbilisi(t))
	require.NoError(t, err)
	require.NotNil(t, answer.List)

	labels := make([]string, 0, len(answer.List.Items))
	for _, item := range answer.List.Items {
		labels = append(labels, item.Label)
	}

	assert.Equal(t, []string{"Latest refuel", "Consumption", "Odometer"}, labels)

	empty, err := widget.NewCar(fixedCar{summary: hatchback(), found: true}).Load(inTbilisi(t))
	require.NoError(t, err)
	assert.Equal(t, pluginapi.EmptyAnswer("No refuel and no reminder yet."), empty)
}

// DRIVVO-SC-032: With no vehicle, the widget answers not set up with the link to the
// connection page.
func TestTheWidgetWithNoVehicleIsNotSetUp(t *testing.T) {
	t.Parallel()

	answer, err := widget.NewCar(fixedCar{}).Load(inTbilisi(t))
	require.NoError(t, err)
	assert.Equal(t, pluginapi.NotSetUpAnswer("Connect the Drivvo account.", "/account"), answer)
}
