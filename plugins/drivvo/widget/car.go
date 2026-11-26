package widget

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/drivvo/figures"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/permission"
	"github.com/abgeo/maroid/plugins/drivvo/service"
)

const (
	accountPage = "/account"
	separator   = " · "
	// tenths rounds a consumption to one decimal.
	tenths = 10
)

// CarReader reads the facts of the widget for the default vehicle.
type CarReader interface {
	Car(ctx context.Context) (service.CarSummary, bool, error)
}

// NewCar builds the widget of the default vehicle: the latest refuel, the consumption of
// the last 12 months, the latest odometer, the spend of the current month, and the first
// reminder.
func NewCar(cars CarReader) *pluginapi.FuncWidget {
	return pluginapi.NewWidget(
		pluginapi.WidgetMeta{
			Key:        "car",
			Title:      "Drivvo",
			Kind:       pluginapi.WidgetKindList,
			Size:       pluginapi.WidgetSizeM,
			Permission: permission.AccountRead,
		},
		func(ctx context.Context) (pluginapi.WidgetAnswer, error) {
			car, found, err := cars.Car(ctx)
			if err != nil {
				return pluginapi.WidgetAnswer{}, fmt.Errorf("reading the default vehicle: %w", err)
			}

			if !found {
				return pluginapi.NotSetUpAnswer("Connect the Drivvo account.", accountPage), nil
			}

			items := items(ctx, car)
			if len(items) == 0 {
				return pluginapi.EmptyAnswer("No refuel and no reminder yet."), nil
			}

			return pluginapi.ListAnswer(pluginapi.ListData{
				Items: items,
				More:  vehiclePage(car.Vehicle, "overview"),
			}), nil
		},
	)
}

func items(ctx context.Context, car service.CarSummary) []pluginapi.ListItem {
	var answer []pluginapi.ListItem

	if refuel := car.LatestRefuel; refuel != nil {
		answer = append(answer, pluginapi.ListItem{
			Label: "Latest refuel",
			Meta:  refuelMeta(ctx, *refuel, car.Settings),
			Link:  vehiclePage(car.Vehicle, "refuels"),
		})
	}

	if car.Consumption != nil {
		answer = append(answer, pluginapi.ListItem{
			Label: "Consumption",
			Meta: strconv.FormatFloat(
				math.Round(*car.Consumption*tenths)/tenths,
				'f',
				1,
				64,
			) + " " +
				ConsumptionUnit(
					car.Vehicle.DistanceUnit,
					car.Settings,
				),
			Link: vehiclePage(car.Vehicle, "overview"),
		})
	}

	if car.Odometer != nil {
		answer = append(answer, pluginapi.ListItem{
			Label: "Odometer",
			Meta:  grouped(*car.Odometer) + " " + string(car.Vehicle.DistanceUnit),
			Link:  vehiclePage(car.Vehicle, "overview"),
		})
	}

	if car.MonthSpend != nil {
		answer = append(answer, pluginapi.ListItem{
			Label: "This month",
			Meta:  money(*car.MonthSpend, car.Settings.Currency),
			Link:  vehiclePage(car.Vehicle, "overview"),
		})
	}

	if reminder := car.Reminder; reminder != nil {
		label := reminder.Type
		if label == "" {
			label = "Reminder"
		}

		answer = append(answer, pluginapi.ListItem{
			Label: label,
			Meta:  reminderMeta(*reminder, car.Vehicle.DistanceUnit),
			Link:  vehiclePage(car.Vehicle, "reminders"),
		})
	}

	return answer
}

func refuelMeta(ctx context.Context, refuel model.EntryRow, settings model.CarLogSettings) string {
	var volume float64

	for _, line := range refuel.Lines {
		if line.Fuel() && line.Volume != nil {
			volume += *line.Volume
		}
	}

	parts := []string{
		model.FormatDay(model.DayOf(refuel.OccurredAt, pluginapi.ActingTimeZone(ctx))),
		strconv.FormatFloat(volume, 'f', 2, 64) + " " + VolumeLabel(settings.VolumeUnit),
	}

	if refuel.Amount != nil {
		parts = append(parts, money(*refuel.Amount, settings.Currency))
	}

	return strings.Join(parts, separator)
}

func money(amount model.Amount, currency string) string {
	return strconv.FormatFloat(amount.Float(), 'f', 2, 64) + " " + currency
}

func reminderMeta(reminder figures.DueReminder, unit model.DistanceUnit) string {
	switch {
	case reminder.Due:
		return "due"
	case reminder.DaysLeft != nil && *reminder.DaysLeft == 1:
		return "in 1 day"
	case reminder.DaysLeft != nil:
		return "in " + strconv.Itoa(*reminder.DaysLeft) + " days"
	case reminder.DistanceLeft != nil:
		return "in " + grouped(*reminder.DistanceLeft) + " " + string(unit)
	default:
		return ""
	}
}

// VolumeLabel answers the label of the volume unit.
func VolumeLabel(unit model.VolumeUnit) string {
	switch unit {
	case model.VolumeGallonUS:
		return "US gal"
	case model.VolumeGallonImperial:
		return "imp gal"
	case model.VolumeLitre:
		return "L"
	default:
		return string(unit)
	}
}

// ConsumptionUnit answers the label of a consumption in the shape of the settings.
func ConsumptionUnit(distance model.DistanceUnit, settings model.CarLogSettings) string {
	volume := VolumeLabel(settings.VolumeUnit)
	if settings.ConsumptionFormat.Shape() == model.ShapeVolumePer100 {
		return volume + "/100 " + string(distance)
	}

	return string(distance) + "/" + volume
}

func vehiclePage(vehicle model.Vehicle, page string) string {
	return "/" + page + "/" + vehicle.ID
}

// grouped writes the whole part of the value with a comma between each group of three
// digits, as 1,200.
func grouped(value float64) string {
	whole := int64(math.Round(value))
	sign := ""

	if whole < 0 {
		sign, whole = "-", -whole
	}

	digits := strconv.FormatInt(whole, 10)

	var builder strings.Builder

	for index, digit := range digits {
		if index > 0 && (len(digits)-index)%3 == 0 {
			builder.WriteByte(',')
		}

		builder.WriteRune(digit)
	}

	return sign + builder.String()
}
