package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// EntryReferences holds the identifiers of the rows of the reference items that a log
// entry names. A nil value names none.
type EntryReferences struct {
	Station       *string
	Place         *string
	PaymentMethod *string
	Reason        *string
	IncomeType    *string
}

// The upsert of each kind stores the entry, and keeps the identifier of a row that exists.
const (
	upsertRefuelQuery = `
		INSERT INTO refuels (
			vehicle_id, source_id, occurred_at, odometer, amount, note, station_id,
			payment_method_id, reason_id, missed_refuel, battery_start_pct, battery_end_pct,
			charge_minutes
		)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT ON CONSTRAINT refuels_source_key DO UPDATE
		SET vehicle_id = excluded.vehicle_id,
		    occurred_at = excluded.occurred_at,
		    odometer = excluded.odometer,
		    amount = excluded.amount,
		    note = excluded.note,
		    station_id = excluded.station_id,
		    payment_method_id = excluded.payment_method_id,
		    reason_id = excluded.reason_id,
		    missed_refuel = excluded.missed_refuel,
		    battery_start_pct = excluded.battery_start_pct,
		    battery_end_pct = excluded.battery_end_pct,
		    charge_minutes = excluded.charge_minutes,
		    collected_at = now()
		RETURNING id;`

	upsertMaintenanceQuery = `
		INSERT INTO maintenances (
			vehicle_id, source_id, occurred_at, odometer, amount, note, title, place_id,
			payment_method_id
		)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8, $9)
		ON CONFLICT ON CONSTRAINT maintenances_source_key DO UPDATE
		SET vehicle_id = excluded.vehicle_id,
		    occurred_at = excluded.occurred_at,
		    odometer = excluded.odometer,
		    amount = excluded.amount,
		    note = excluded.note,
		    title = excluded.title,
		    place_id = excluded.place_id,
		    payment_method_id = excluded.payment_method_id,
		    collected_at = now()
		RETURNING id;`

	upsertExpenseQuery = `
		INSERT INTO expenses (
			vehicle_id, source_id, occurred_at, odometer, amount, note, title, place_id,
			payment_method_id, reason_id
		)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8, $9, $10)
		ON CONFLICT ON CONSTRAINT expenses_source_key DO UPDATE
		SET vehicle_id = excluded.vehicle_id,
		    occurred_at = excluded.occurred_at,
		    odometer = excluded.odometer,
		    amount = excluded.amount,
		    note = excluded.note,
		    title = excluded.title,
		    place_id = excluded.place_id,
		    payment_method_id = excluded.payment_method_id,
		    reason_id = excluded.reason_id,
		    collected_at = now()
		RETURNING id;`

	upsertIncomeQuery = `
		INSERT INTO incomes (
			vehicle_id, source_id, occurred_at, odometer, amount, note, income_type_id
		)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7)
		ON CONFLICT ON CONSTRAINT incomes_source_key DO UPDATE
		SET vehicle_id = excluded.vehicle_id,
		    occurred_at = excluded.occurred_at,
		    odometer = excluded.odometer,
		    amount = excluded.amount,
		    note = excluded.note,
		    income_type_id = excluded.income_type_id,
		    collected_at = now()
		RETURNING id;`

	upsertOdometerQuery = `
		INSERT INTO odometer_entries (vehicle_id, source_id, occurred_at, odometer, note)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		ON CONFLICT ON CONSTRAINT odometer_entries_source_key DO UPDATE
		SET vehicle_id = excluded.vehicle_id,
		    occurred_at = excluded.occurred_at,
		    odometer = excluded.odometer,
		    note = excluded.note,
		    collected_at = now()
		RETURNING id;`
)

// Entries reads and writes the log entries of the acting workspace within one transaction.
// Each kind has a table of its own. A read names the view that joins them.
type Entries struct {
	tx *sqlx.Tx
}

// NewEntries creates a new Entries over the transaction.
func NewEntries(tx *sqlx.Tx) *Entries {
	return &Entries{tx: tx}
}

// Upsert stores the entry of the vehicle in the table of its kind by its identifier at the
// car log service, and answers the identifier of its row. A row that exists keeps its
// identifier.
func (r *Entries) Upsert(
	ctx context.Context,
	vehicleID string,
	entry model.LogEntry,
	refs EntryReferences,
) (string, error) {
	query, arguments, err := upsertOf(vehicleID, entry, refs)
	if err != nil {
		return "", err
	}

	var id string
	if err = r.tx.GetContext(ctx, &id, query, arguments...); err != nil {
		return "", fmt.Errorf("upserting the %s %d: %w", entry.Kind, entry.SourceID, err)
	}

	return id, nil
}

// DeleteAbsent deletes each entry of the kind whose identifier at the car log service is
// not kept, with its lines and its instalments.
func (r *Entries) DeleteAbsent(ctx context.Context, kind model.Kind, kept []int32) error {
	table, err := entryTable(kind)
	if err != nil {
		return err
	}

	query := `DELETE FROM ` + table + ` WHERE source_id <> ALL($1::integer[]);`

	if _, err = r.tx.ExecContext(ctx, query, kept); err != nil {
		return fmt.Errorf("deleting the absent entries of kind %s: %w", kind, err)
	}

	return nil
}

func upsertOf(
	vehicleID string,
	entry model.LogEntry,
	refs EntryReferences,
) (string, []any, error) {
	common := []any{vehicleID, entry.SourceID, entry.OccurredAt, entry.Odometer}

	switch entry.Kind {
	case model.KindRefuel:
		return upsertRefuelQuery, append(
			common,
			entry.Amount,
			entry.Note,
			refs.Station,
			refs.PaymentMethod,
			refs.Reason,
			entry.MissedRefuel,
			entry.BatteryStartPct,
			entry.BatteryEndPct,
			entry.ChargeMinutes,
		), nil
	case model.KindMaintenance:
		return upsertMaintenanceQuery, append(
			common,
			entry.Amount,
			entry.Note,
			entry.Title,
			refs.Place,
			refs.PaymentMethod,
		), nil
	case model.KindExpense:
		return upsertExpenseQuery, append(
			common,
			entry.Amount,
			entry.Note,
			entry.Title,
			refs.Place,
			refs.PaymentMethod,
			refs.Reason,
		), nil
	case model.KindIncome:
		return upsertIncomeQuery, append(common, entry.Amount, entry.Note, refs.IncomeType), nil
	case model.KindOdometer:
		return upsertOdometerQuery, append(common, entry.Note), nil
	}

	return "", nil, fmt.Errorf("%w: %q", errs.ErrUnknownKind, entry.Kind)
}

func entryTable(kind model.Kind) (string, error) {
	switch kind {
	case model.KindRefuel:
		return "refuels", nil
	case model.KindMaintenance:
		return "maintenances", nil
	case model.KindExpense:
		return "expenses", nil
	case model.KindIncome:
		return "incomes", nil
	case model.KindOdometer:
		return "odometer_entries", nil
	}

	return "", fmt.Errorf("%w: %q", errs.ErrUnknownKind, kind)
}

// entryColumns names the columns of a read of an entry from the view log_entries, which
// answers the names of its references.
const entryColumns = `
	t.id, t.vehicle_id, t.kind, t.occurred_at, t.odometer, t.amount, t.title, t.note,
	t.station, t.place, t.payment_method, t.reason, t.income_type, t.missed_refuel,
	t.battery_start_pct, t.battery_end_pct, t.charge_minutes`

// entryFilter names the entries of the vehicle of the kinds in the period. An empty list
// of kinds names every kind.
const entryFilter = `
	t.vehicle_id = CAST(:vehicle_id AS uuid)
	AND (cardinality(CAST(:kinds AS text[])) = 0 OR t.kind = ANY(CAST(:kinds AS text[])))
	AND (CAST(:start AS timestamptz) IS NULL OR t.occurred_at >= CAST(:start AS timestamptz))
	AND (CAST(:end AS timestamptz) IS NULL OR t.occurred_at < CAST(:end AS timestamptz))`

// List answers one keyset read of the entries of the vehicle that the filter names, by
// moment.
func (r *Entries) List(
	ctx context.Context,
	seek page.Seek,
	vehicleID string,
	filter model.EntryFilter,
	order Order,
) ([]model.EntryRow, error) {
	condition, sorting := keyset(seek, "log_entries", "occurred_at", order)
	query := `
		SELECT ` + entryColumns + `
		FROM log_entries t
		WHERE ` + entryFilter + `
		  AND ` + condition + `
		ORDER BY ` + sorting + `
		LIMIT :limit;`

	arguments := filterArguments(vehicleID, filter)
	arguments["boundary"] = boundaryOf(seek)
	arguments["limit"] = seek.Limit

	entries, err := selectNamed[model.EntryRow](ctx, r.tx, query, arguments)
	if err != nil {
		return nil, fmt.Errorf("listing the log entries: %w", err)
	}

	return inOrder(seek, entries), nil
}

// Totals sums the entries of the vehicle that the filter names, and the volume of their
// lines.
func (r *Entries) Totals(
	ctx context.Context,
	vehicleID string,
	filter model.EntryFilter,
) (model.EntryTotals, error) {
	query := `
		WITH chosen AS (
			SELECT t.id, t.amount FROM log_entries t WHERE ` + entryFilter + `
		)
		SELECT (SELECT count(*) FROM chosen) AS count,
		       (SELECT COALESCE(sum(amount), 0) FROM chosen) AS amount,
		       (SELECT COALESCE(sum(l.volume), 0) FROM log_entry_lines l
		        WHERE l.log_entry_id IN (SELECT id FROM chosen)) AS volume;`

	totals, err := selectNamed[model.EntryTotals](
		ctx,
		r.tx,
		query,
		filterArguments(vehicleID, filter),
	)
	if err != nil {
		return model.EntryTotals{}, fmt.Errorf("summing the log entries: %w", err)
	}

	return totals[0], nil
}

// OfVehicle answers every entry of the vehicle, the earliest first.
func (r *Entries) OfVehicle(ctx context.Context, vehicleID string) ([]model.EntryRow, error) {
	const query = `
		SELECT ` + entryColumns + `
		FROM log_entries t
		WHERE t.vehicle_id = $1
		ORDER BY t.occurred_at, t.id;`

	var entries []model.EntryRow
	if err := r.tx.SelectContext(ctx, &entries, query, vehicleID); err != nil {
		return nil, fmt.Errorf("listing the log entries of the vehicle: %w", err)
	}

	return entries, nil
}

// Get answers the entry of the vehicle, or errs.ErrLogEntryNotFound when the vehicle holds
// no such entry.
func (r *Entries) Get(ctx context.Context, vehicleID, id string) (model.EntryRow, error) {
	const query = `
		SELECT ` + entryColumns + `
		FROM log_entries t
		WHERE t.vehicle_id = $1 AND t.id = $2;`

	var entry model.EntryRow

	err := r.tx.GetContext(ctx, &entry, query, vehicleID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.EntryRow{}, errs.ErrLogEntryNotFound
	}

	if err != nil {
		return model.EntryRow{}, fmt.Errorf("reading the log entry: %w", err)
	}

	return entry, nil
}

func filterArguments(vehicleID string, filter model.EntryFilter) map[string]any {
	kinds := make([]string, 0, len(filter.Kinds))
	for _, kind := range filter.Kinds {
		kinds = append(kinds, string(kind))
	}

	return map[string]any{
		"vehicle_id": vehicleID,
		"kinds":      kinds,
		"start":      filter.Start,
		"end":        filter.End,
	}
}
