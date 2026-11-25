package service

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/repository"
)

// defaultSettings stand in for the settings before the first collection stores them.
func defaultSettings() model.CarLogSettings {
	return model.CarLogSettings{
		VolumeUnit:        model.VolumeLitre,
		ConsumptionFormat: model.FirstConsumptionFormat,
		Currency:          "USD",
	}
}

// Vehicles answers every vehicle of the car log copy, active first, then by name, with the
// car log settings.
func (s *Store) Vehicles(ctx context.Context) ([]model.Vehicle, model.CarLogSettings, error) {
	var (
		vehicles []model.Vehicle
		settings model.CarLogSettings
	)

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		if settings, err = settingsIn(ctx, tx); err != nil {
			return err
		}

		vehicles, err = repository.NewVehicles(tx).List(ctx)

		return err //nolint:wrapcheck // the repository names the read.
	})
	if err != nil {
		return nil, model.CarLogSettings{}, fmt.Errorf("reading the vehicles: %w", err)
	}

	return vehicles, settings, nil
}

// VehicleRows answers the vehicle, the settings, and every entry, with its lines and
// instalments, and every reminder of the vehicle. It answers errs.ErrVehicleNotFound for a
// vehicle that the copy does not hold.
func (s *Store) VehicleRows(ctx context.Context, vehicleID string) (model.VehicleRows, error) {
	var rows model.VehicleRows

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		if rows.Vehicle, rows.Settings, err = vehicleIn(ctx, tx, vehicleID); err != nil {
			return err
		}

		if rows.Entries, err = repository.NewEntries(tx).OfVehicle(ctx, vehicleID); err != nil {
			return err //nolint:wrapcheck // the repository names the read.
		}

		lines, err := repository.NewLines(tx).OfVehicle(ctx, vehicleID)
		if err != nil {
			return err //nolint:wrapcheck // the repository names the read.
		}

		instalments, err := repository.NewInstalments(tx).OfVehicle(ctx, vehicleID)
		if err != nil {
			return err //nolint:wrapcheck // the repository names the read.
		}

		attach(rows.Entries, lines, instalments)

		rows.Reminders, err = repository.NewReminders(tx).OfVehicle(ctx, vehicleID)

		return err //nolint:wrapcheck // the repository names the read.
	})
	if err != nil {
		return model.VehicleRows{}, fmt.Errorf("reading the rows of the vehicle: %w", err)
	}

	return rows, nil
}

// LogEntries answers one page of the entries of the vehicle that the filter names, each
// with its lines, and the vehicle and the settings.
func (s *Store) LogEntries(
	ctx context.Context,
	vehicleID string,
	filter model.EntryFilter,
	asked page.Request,
	order repository.Order,
) (page.Window[model.EntryRow], model.VehicleRows, error) {
	var (
		window page.Window[model.EntryRow]
		held   model.VehicleRows
	)

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		if held.Vehicle, held.Settings, err = vehicleIn(ctx, tx, vehicleID); err != nil {
			return err
		}

		entries := repository.NewEntries(tx)
		fetch := func(ctx context.Context, seek page.Seek) ([]model.EntryRow, error) {
			return entries.List(ctx, seek, vehicleID, filter, order)
		}

		window, err = page.Read(
			ctx,
			asked,
			fetch,
			func(row model.EntryRow) string { return row.ID },
		)
		if err != nil {
			return fmt.Errorf("reading the keyset: %w", err)
		}

		lines, err := repository.NewLines(tx).OfEntries(ctx, entryIDs(window.Items))
		if err != nil {
			return err //nolint:wrapcheck // the repository names the read.
		}

		attach(window.Items, lines, nil)

		return nil
	})
	if err != nil {
		return page.Window[model.EntryRow]{}, model.VehicleRows{}, fmt.Errorf(
			"reading the log entries: %w",
			err,
		)
	}

	return window, held, nil
}

// LogEntryTotals sums the entries of the vehicle that the filter names.
func (s *Store) LogEntryTotals(
	ctx context.Context,
	vehicleID string,
	filter model.EntryFilter,
) (model.EntryTotals, model.CarLogSettings, error) {
	var (
		totals   model.EntryTotals
		settings model.CarLogSettings
	)

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		if _, settings, err = vehicleIn(ctx, tx, vehicleID); err != nil {
			return err
		}

		totals, err = repository.NewEntries(tx).Totals(ctx, vehicleID, filter)

		return err //nolint:wrapcheck // the repository names the read.
	})
	if err != nil {
		return model.EntryTotals{}, model.CarLogSettings{}, fmt.Errorf(
			"summing the log entries: %w",
			err,
		)
	}

	return totals, settings, nil
}

// LogEntry answers the entry of the vehicle with its lines and instalments, and the
// settings. It answers errs.ErrLogEntryNotFound for an entry that the vehicle does not hold.
func (s *Store) LogEntry(
	ctx context.Context,
	vehicleID, entryID string,
) (model.EntryRow, model.CarLogSettings, error) {
	var (
		entry    model.EntryRow
		settings model.CarLogSettings
	)

	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		if _, settings, err = vehicleIn(ctx, tx, vehicleID); err != nil {
			return err
		}

		if entry, err = repository.NewEntries(tx).Get(ctx, vehicleID, entryID); err != nil {
			return err //nolint:wrapcheck // the repository names the read.
		}

		ids := []string{entry.ID}

		lines, err := repository.NewLines(tx).OfEntries(ctx, ids)
		if err != nil {
			return err //nolint:wrapcheck // the repository names the read.
		}

		instalments, err := repository.NewInstalments(tx).OfEntries(ctx, ids)
		if err != nil {
			return err //nolint:wrapcheck // the repository names the read.
		}

		entries := []model.EntryRow{entry}
		attach(entries, lines, instalments)
		entry = entries[0]

		return nil
	})
	if err != nil {
		return model.EntryRow{}, model.CarLogSettings{}, fmt.Errorf(
			"reading the log entry: %w",
			err,
		)
	}

	return entry, settings, nil
}

func settingsIn(ctx context.Context, tx *sqlx.Tx) (model.CarLogSettings, error) {
	settings, found, err := repository.NewSettings(tx).Get(ctx)
	if err != nil {
		return model.CarLogSettings{}, err //nolint:wrapcheck // the repository names the read.
	}

	if !found {
		return defaultSettings(), nil
	}

	return settings, nil
}

func vehicleIn(
	ctx context.Context,
	tx *sqlx.Tx,
	vehicleID string,
) (model.Vehicle, model.CarLogSettings, error) {
	vehicle, err := repository.NewVehicles(tx).Get(ctx, vehicleID)
	if err != nil {
		return model.Vehicle{}, model.CarLogSettings{}, err //nolint:wrapcheck // the repository names the read.
	}

	settings, err := settingsIn(ctx, tx)

	return vehicle, settings, err
}

// attach puts each line and each instalment on its entry.
func attach(entries []model.EntryRow, lines []model.LineRow, instalments []model.InstalmentRow) {
	index := make(map[string]int, len(entries))
	for position, entry := range entries {
		index[entry.ID] = position
	}

	for _, line := range lines {
		if position, found := index[line.EntryID]; found {
			entries[position].Lines = append(entries[position].Lines, line)
		}
	}

	for _, instalment := range instalments {
		if position, found := index[instalment.EntryID]; found {
			entries[position].Instalments = append(entries[position].Instalments, instalment)
		}
	}
}

func entryIDs(entries []model.EntryRow) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}

	return ids
}
