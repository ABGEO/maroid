package service

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/drivvo/figures"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/repository"
)

// widgetMonthsBack is the count of months before the current one that the consumption of
// the widget covers, so it covers 12 months.
const widgetMonthsBack = 11

// CarLogReader reads the car log copy of the acting workspace.
type CarLogReader interface {
	Vehicles(ctx context.Context) ([]model.Vehicle, model.CarLogSettings, error)
	VehicleRows(ctx context.Context, vehicleID string) (model.VehicleRows, error)
	LogEntries(
		ctx context.Context,
		vehicleID string,
		filter model.EntryFilter,
		asked page.Request,
		order repository.Order,
	) (page.Window[model.EntryRow], model.VehicleRows, error)
	LogEntryTotals(
		ctx context.Context,
		vehicleID string,
		filter model.EntryFilter,
	) (model.EntryTotals, model.CarLogSettings, error)
	LogEntry(
		ctx context.Context,
		vehicleID, entryID string,
	) (model.EntryRow, model.CarLogSettings, error)
}

var _ CarLogReader = (*Store)(nil)

// VehicleList holds every vehicle of the copy, the one that a page with no vehicle opens,
// and the settings.
type VehicleList struct {
	Vehicles []model.Vehicle
	Default  string
	Settings model.CarLogSettings
}

// VehicleOverview holds the figures of one vehicle in one period.
type VehicleOverview struct {
	figures.Overview

	Period   model.Period
	Vehicle  model.Vehicle
	Settings model.CarLogSettings
}

// EntryPage holds one page of log entries, the consumption of each refuel that ends a part
// with a value, and the vehicle and the settings.
type EntryPage struct {
	Window      page.Window[model.EntryRow]
	Consumption map[string]float64
	Vehicle     model.Vehicle
	Settings    model.CarLogSettings
}

// VehicleReminders holds the reminders of one vehicle with their due points.
type VehicleReminders struct {
	Reminders []figures.DueReminder
	Vehicle   model.Vehicle
	Settings  model.CarLogSettings
}

// CarSummary holds the facts of the widget for the default vehicle. A fact that the copy
// cannot give is nil.
type CarSummary struct {
	Vehicle      model.Vehicle
	Settings     model.CarLogSettings
	LatestRefuel *model.EntryRow
	Consumption  *float64
	Reminder     *figures.DueReminder
}

// Figures answers the pages and the widget from the car log copy. It calls no service.
type Figures struct {
	store CarLogReader
	now   func() time.Time
}

// NewFigures creates a new Figures. now gives today.
func NewFigures(store CarLogReader, now func() time.Time) *Figures {
	return &Figures{store: store, now: now}
}

// Vehicles answers every vehicle, active first, then by name, and marks the default one:
// the principal vehicle, else the first active one, else the first one.
func (f *Figures) Vehicles(ctx context.Context) (VehicleList, error) {
	vehicles, settings, err := f.store.Vehicles(ctx)
	if err != nil {
		return VehicleList{}, err //nolint:wrapcheck // the store names the read.
	}

	return VehicleList{
		Vehicles: vehicles,
		Default:  defaultVehicle(vehicles),
		Settings: settings,
	}, nil
}

// Overview answers the figures of the vehicle in the period, in the zone of the member.
func (f *Figures) Overview(
	ctx context.Context,
	vehicleID string,
	period model.Period,
) (VehicleOverview, error) {
	rows, err := f.store.VehicleRows(ctx, vehicleID)
	if err != nil {
		return VehicleOverview{}, err //nolint:wrapcheck // the store names the read.
	}

	loc := pluginapi.ActingTimeZone(ctx)

	return VehicleOverview{
		Overview: figures.Summarize(
			FigureEntries(rows.Entries),
			period,
			loc,
			model.DayOf(f.now(), loc),
			rows.Settings.ConsumptionFormat,
		),
		Period:   period,
		Vehicle:  rows.Vehicle,
		Settings: rows.Settings,
	}, nil
}

// LogEntries answers one page of the entries of the vehicle that the filter names. A page
// that holds a refuel reads every refuel of the vehicle for the consumption.
func (f *Figures) LogEntries(
	ctx context.Context,
	vehicleID string,
	filter model.EntryFilter,
	asked page.Request,
	order repository.Order,
) (EntryPage, error) {
	window, held, err := f.store.LogEntries(ctx, vehicleID, filter, asked, order)
	if err != nil {
		return EntryPage{}, err //nolint:wrapcheck // the store names the read.
	}

	answer := EntryPage{Window: window, Vehicle: held.Vehicle, Settings: held.Settings}

	if !holdsRefuel(window.Items) {
		return answer, nil
	}

	rows, err := f.store.VehicleRows(ctx, vehicleID)
	if err != nil {
		return EntryPage{}, err //nolint:wrapcheck // the store names the read.
	}

	answer.Consumption = figures.PartConsumption(
		figures.Parts(FigureEntries(rows.Entries)),
		rows.Settings.ConsumptionFormat,
	)

	return answer, nil
}

// LogEntryTotals sums the entries of the vehicle that the filter names.
func (f *Figures) LogEntryTotals(
	ctx context.Context,
	vehicleID string,
	filter model.EntryFilter,
) (model.EntryTotals, model.CarLogSettings, error) {
	totals, settings, err := f.store.LogEntryTotals(ctx, vehicleID, filter)
	if err != nil {
		return model.EntryTotals{}, model.CarLogSettings{}, fmt.Errorf(
			"reading the totals: %w",
			err,
		)
	}

	return totals, settings, nil
}

// LogEntry answers one entry of the vehicle with its lines and instalments.
func (f *Figures) LogEntry(
	ctx context.Context,
	vehicleID, entryID string,
) (model.EntryRow, model.CarLogSettings, error) {
	entry, settings, err := f.store.LogEntry(ctx, vehicleID, entryID)
	if err != nil {
		return model.EntryRow{}, model.CarLogSettings{}, fmt.Errorf("reading the entry: %w", err)
	}

	return entry, settings, nil
}

// Reminders answers the reminders of the vehicle with their due points, in their order.
func (f *Figures) Reminders(ctx context.Context, vehicleID string) (VehicleReminders, error) {
	rows, err := f.store.VehicleRows(ctx, vehicleID)
	if err != nil {
		return VehicleReminders{}, err //nolint:wrapcheck // the store names the read.
	}

	return VehicleReminders{
		Reminders: f.dueReminders(ctx, rows),
		Vehicle:   rows.Vehicle,
		Settings:  rows.Settings,
	}, nil
}

// Car answers the facts of the widget for the default vehicle, and false when the copy
// holds no vehicle.
func (f *Figures) Car(ctx context.Context) (CarSummary, bool, error) {
	list, err := f.Vehicles(ctx)
	if err != nil {
		return CarSummary{}, false, err
	}

	if list.Default == "" {
		return CarSummary{}, false, nil
	}

	rows, err := f.store.VehicleRows(ctx, list.Default)
	if err != nil {
		return CarSummary{}, false, fmt.Errorf("reading the default vehicle: %w", err)
	}

	loc := pluginapi.ActingTimeZone(ctx)
	today := model.DayOf(f.now(), loc)
	from := model.MonthOf(today).AddDate(0, -widgetMonthsBack, 0)
	entries := FigureEntries(rows.Entries)

	summary := CarSummary{
		Vehicle:      rows.Vehicle,
		Settings:     rows.Settings,
		LatestRefuel: latestRefuel(rows.Entries),
		Consumption: figures.Consumption(
			figures.Parts(entries),
			model.Period{From: &from, To: &today},
			loc,
			rows.Settings.ConsumptionFormat,
		),
	}

	if due := f.dueReminders(ctx, rows); len(due) > 0 {
		summary.Reminder = &due[0]
	}

	return summary, true, nil
}

func (f *Figures) dueReminders(ctx context.Context, rows model.VehicleRows) []figures.DueReminder {
	loc := pluginapi.ActingTimeZone(ctx)
	entries := FigureEntries(rows.Entries)

	return figures.DueReminders(
		figureReminders(rows.Reminders),
		entries,
		figures.LatestOdometer(entries),
		model.DayOf(f.now(), loc),
		loc,
	)
}

func defaultVehicle(vehicles []model.Vehicle) string {
	for _, vehicle := range vehicles {
		if vehicle.Principal {
			return vehicle.ID
		}
	}

	for _, vehicle := range vehicles {
		if vehicle.Active {
			return vehicle.ID
		}
	}

	if len(vehicles) > 0 {
		return vehicles[0].ID
	}

	return ""
}

func holdsRefuel(entries []model.EntryRow) bool {
	for _, entry := range entries {
		if entry.Kind == model.KindRefuel {
			return true
		}
	}

	return false
}

func latestRefuel(entries []model.EntryRow) *model.EntryRow {
	var latest *model.EntryRow

	for index := range entries {
		entry := &entries[index]
		if entry.Kind == model.KindRefuel &&
			(latest == nil || entry.OccurredAt.After(latest.OccurredAt)) {
			latest = entry
		}
	}

	return latest
}

// FigureEntries converts the rows of the copy to the entries of the figures.
func FigureEntries(rows []model.EntryRow) []figures.Entry {
	entries := make([]figures.Entry, 0, len(rows))

	for _, row := range rows {
		entry := figures.Entry{
			ID:           row.ID,
			Kind:         row.Kind,
			OccurredAt:   row.OccurredAt,
			Odometer:     row.Odometer,
			Amount:       row.Amount,
			MissedRefuel: row.MissedRefuel,
		}

		for _, line := range row.Lines {
			entry.Lines = append(entry.Lines, figures.Line{
				Item:     valueOf(line.ItemID),
				Name:     valueOf(line.Item),
				Amount:   line.Amount,
				Fuel:     line.Fuel(),
				Volume:   valueOf(line.Volume),
				FullTank: valueOf(line.FullTank),
				Free:     valueOf(line.Free),
			})
		}

		for _, instalment := range row.Instalments {
			entry.Instalments = append(entry.Instalments, model.Instalment{
				Number: instalment.Number,
				DueOn:  instalment.DueOn,
				Amount: instalment.Amount,
			})
		}

		entries = append(entries, entry)
	}

	return entries
}

func figureReminders(rows []model.ReminderRow) []figures.Reminder {
	reminders := make([]figures.Reminder, 0, len(rows))

	for _, row := range rows {
		reminders = append(reminders, figures.Reminder{
			ID:             row.ID,
			Kind:           row.Kind,
			Item:           valueOf(row.ItemID),
			Type:           valueOf(row.Type),
			DueOn:          row.DueOn,
			DueOdometer:    row.DueOdometer,
			Repeats:        row.Repeats,
			RepeatEvery:    row.RepeatEvery,
			RepeatUnit:     row.RepeatUnit,
			RepeatDistance: row.RepeatDistance,
			Note:           valueOf(row.Note),
		})
	}

	return reminders
}

func valueOf[T any](pointer *T) T {
	var zero T
	if pointer == nil {
		return zero
	}

	return *pointer
}
