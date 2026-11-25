package service

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/repository"
)

// Store runs each operation on the car log copy of the acting workspace in a transaction
// of its own, and creates each repository inside it.
type Store struct {
	db pluginapi.PluginDB
}

// NewStore creates a new Store over the database of the plugin.
func NewStore(db pluginapi.PluginDB) *Store {
	return &Store{db: db}
}

type referenceKey struct {
	kind     model.ReferenceKind
	sourceID int32
}

// replacement holds the identifiers of the rows that one replace stored, by the identifier
// of each item at the car log service.
type replacement struct {
	tx         *sqlx.Tx
	references map[referenceKey]string
	vehicles   map[int32]string
}

// Replace makes the car log copy of the acting workspace hold the snapshot and nothing
// else, in one transaction. A row that the snapshot holds keeps its identifier. A failure
// stores nothing.
func (s *Store) Replace(ctx context.Context, snapshot model.Snapshot) error {
	err := s.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		run := &replacement{
			tx:         tx,
			references: map[referenceKey]string{},
			vehicles:   map[int32]string{},
		}

		steps := []func(context.Context, model.Snapshot) error{
			run.settings,
			run.referenceItems,
			run.vehicleRows,
			run.logEntries,
			run.reminders,
		}

		for _, step := range steps {
			if err := step(ctx, snapshot); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("replacing the car log copy: %w", err)
	}

	return nil
}

func (r *replacement) settings(ctx context.Context, snapshot model.Snapshot) error {
	if err := repository.NewSettings(r.tx).Put(ctx, snapshot.Settings); err != nil {
		return fmt.Errorf("storing the settings: %w", err)
	}

	return nil
}

func (r *replacement) referenceItems(ctx context.Context, snapshot model.Snapshot) error {
	references := repository.NewReferences(r.tx)
	kept := map[model.ReferenceKind][]int32{}

	for _, item := range snapshot.ReferenceItems {
		id, err := references.Upsert(ctx, item)
		if err != nil {
			return fmt.Errorf("storing the reference items: %w", err)
		}

		r.references[referenceKey{item.Kind, item.SourceID}] = id
		kept[item.Kind] = append(kept[item.Kind], item.SourceID)
	}

	for _, kind := range referenceKinds() {
		if err := references.DeleteAbsent(ctx, kind, nonNil(kept[kind])); err != nil {
			return fmt.Errorf("storing the reference items: %w", err)
		}
	}

	return nil
}

func (r *replacement) vehicleRows(ctx context.Context, snapshot model.Snapshot) error {
	vehicles := repository.NewVehicles(r.tx)
	kept := make([]int32, 0, len(snapshot.Vehicles))

	for _, vehicle := range snapshot.Vehicles {
		id, err := vehicles.Upsert(ctx, vehicle)
		if err != nil {
			return fmt.Errorf("storing the vehicles: %w", err)
		}

		r.vehicles[vehicle.SourceID] = id
		kept = append(kept, vehicle.SourceID)
	}

	if err := vehicles.DeleteAbsent(ctx, kept); err != nil {
		return fmt.Errorf("storing the vehicles: %w", err)
	}

	return nil
}

// logEntries stores the entries, deletes the absent ones, and writes every line and every
// instalment again. A line and an instalment have no address, so a fresh insert loses
// nothing.
func (r *replacement) logEntries(ctx context.Context, snapshot model.Snapshot) error {
	entryIDs, err := r.upsertEntries(ctx, snapshot.LogEntries)
	if err != nil {
		return err
	}

	lines, instalments := repository.NewLines(r.tx), repository.NewInstalments(r.tx)

	if err = lines.DeleteAll(ctx); err != nil {
		return fmt.Errorf("storing the lines: %w", err)
	}

	if err = instalments.DeleteAll(ctx); err != nil {
		return fmt.Errorf("storing the instalments: %w", err)
	}

	for index, entry := range snapshot.LogEntries {
		for _, line := range entry.Lines {
			item := r.reference(lineItemKind(entry.Kind), line.Item)
			if err = lines.Insert(ctx, entryIDs[index], line, item); err != nil {
				return fmt.Errorf(
					"storing the lines of the %s %d: %w",
					entry.Kind,
					entry.SourceID,
					err,
				)
			}
		}

		for _, instalment := range entry.Instalments {
			if err = instalments.Insert(ctx, entryIDs[index], instalment); err != nil {
				return fmt.Errorf(
					"storing the instalments of the %s %d: %w",
					entry.Kind,
					entry.SourceID,
					err,
				)
			}
		}
	}

	return nil
}

// upsertEntries stores the entries, deletes the absent ones, and answers the identifier of
// the row of each entry, in the order of the entries.
func (r *replacement) upsertEntries(
	ctx context.Context,
	logEntries []model.LogEntry,
) ([]string, error) {
	var (
		entries  = repository.NewEntries(r.tx)
		kept     = map[model.Kind][]int32{}
		entryIDs = make([]string, len(logEntries))
	)

	for index, entry := range logEntries {
		vehicleID, err := r.vehicle(entry.VehicleSource)
		if err != nil {
			return nil, fmt.Errorf("storing the %s %d: %w", entry.Kind, entry.SourceID, err)
		}

		entryIDs[index], err = entries.Upsert(ctx, vehicleID, entry, r.entryReferences(entry))
		if err != nil {
			return nil, fmt.Errorf("storing the log entries: %w", err)
		}

		kept[entry.Kind] = append(kept[entry.Kind], entry.SourceID)
	}

	for _, kind := range entryKinds() {
		if err := entries.DeleteAbsent(ctx, kind, nonNil(kept[kind])); err != nil {
			return nil, fmt.Errorf("storing the log entries: %w", err)
		}
	}

	return entryIDs, nil
}

func (r *replacement) reminders(ctx context.Context, snapshot model.Snapshot) error {
	reminders := repository.NewReminders(r.tx)
	kept := make([]int32, 0, len(snapshot.Reminders))

	for _, reminder := range snapshot.Reminders {
		vehicleID, err := r.vehicle(reminder.VehicleSource)
		if err != nil {
			return fmt.Errorf("storing the reminder %d: %w", reminder.SourceID, err)
		}

		item := r.reference(lineItemKind(reminder.Kind), reminder.Item)
		if err = reminders.Upsert(ctx, vehicleID, reminder, item); err != nil {
			return fmt.Errorf("storing the reminders: %w", err)
		}

		kept = append(kept, reminder.SourceID)
	}

	if err := reminders.DeleteAbsent(ctx, kept); err != nil {
		return fmt.Errorf("storing the reminders: %w", err)
	}

	return nil
}

// vehicle answers the row of the vehicle that the snapshot holds. A log entry or a reminder
// of a vehicle that the snapshot does not hold fails the replace.
func (r *replacement) vehicle(sourceID int32) (string, error) {
	id, found := r.vehicles[sourceID]
	if !found {
		return "", fmt.Errorf(
			"%w: the snapshot holds no vehicle %d",
			errs.ErrRequestFailed,
			sourceID,
		)
	}

	return id, nil
}

// reference answers the row of the reference item, or nil when the snapshot holds none.
func (r *replacement) reference(kind model.ReferenceKind, sourceID int32) *string {
	id, found := r.references[referenceKey{kind, sourceID}]
	if !found {
		return nil
	}

	return &id
}

func (r *replacement) entryReferences(entry model.LogEntry) repository.EntryReferences {
	return repository.EntryReferences{
		Station:       r.reference(model.ReferenceStation, entry.Station),
		Place:         r.reference(model.ReferencePlace, entry.Place),
		PaymentMethod: r.reference(model.ReferencePaymentMethod, entry.PaymentMethod),
		Reason:        r.reference(model.ReferenceReason, entry.Reason),
		IncomeType:    r.reference(model.ReferenceIncomeType, entry.IncomeType),
	}
}

// lineItemKind answers the kind of the reference item that a line or a reminder of the kind
// names. An income and an odometer entry hold no line.
func lineItemKind(kind model.Kind) model.ReferenceKind {
	return map[model.Kind]model.ReferenceKind{
		model.KindRefuel:      model.ReferenceFuel,
		model.KindMaintenance: model.ReferenceMaintenanceType,
		model.KindExpense:     model.ReferenceExpenseType,
	}[kind]
}

func referenceKinds() []model.ReferenceKind {
	return []model.ReferenceKind{
		model.ReferenceFuel,
		model.ReferenceMaintenanceType,
		model.ReferenceExpenseType,
		model.ReferenceIncomeType,
		model.ReferenceReason,
		model.ReferencePaymentMethod,
		model.ReferenceStation,
		model.ReferencePlace,
	}
}

func entryKinds() []model.Kind {
	return []model.Kind{
		model.KindRefuel,
		model.KindMaintenance,
		model.KindExpense,
		model.KindIncome,
		model.KindOdometer,
	}
}

// nonNil answers an empty list in place of nil, so the array parameter is empty, not null.
func nonNil(kept []int32) []int32 {
	if kept == nil {
		return []int32{}
	}

	return kept
}
