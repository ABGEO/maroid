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

// upsertEntryQuery stores a log entry, and keeps the identifier of a row that exists.
const upsertEntryQuery = `
	INSERT INTO log_entries (
		vehicle_id, kind, source_id, occurred_at, odometer, amount, title, note,
		station_id, place_id, payment_method_id, reason_id, income_type_id,
		missed_refuel, battery_start_pct, battery_end_pct, charge_minutes
	)
	VALUES (
		$1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''),
		$9, $10, $11, $12, $13, $14, $15, $16, $17
	)
	ON CONFLICT ON CONSTRAINT log_entries_source_key DO UPDATE
	SET vehicle_id = excluded.vehicle_id,
	    occurred_at = excluded.occurred_at,
	    odometer = excluded.odometer,
	    amount = excluded.amount,
	    title = excluded.title,
	    note = excluded.note,
	    station_id = excluded.station_id,
	    place_id = excluded.place_id,
	    payment_method_id = excluded.payment_method_id,
	    reason_id = excluded.reason_id,
	    income_type_id = excluded.income_type_id,
	    missed_refuel = excluded.missed_refuel,
	    battery_start_pct = excluded.battery_start_pct,
	    battery_end_pct = excluded.battery_end_pct,
	    charge_minutes = excluded.charge_minutes,
	    collected_at = now()
	RETURNING id;`

// Entries reads and writes the log entries of the acting workspace within one transaction.
type Entries struct {
	tx *sqlx.Tx
}

// NewEntries creates a new Entries over the transaction.
func NewEntries(tx *sqlx.Tx) *Entries {
	return &Entries{tx: tx}
}

// Upsert stores the entry of the vehicle by its kind and its identifier at the car log
// service, and answers the identifier of its row. A row that exists keeps its identifier.
func (r *Entries) Upsert(
	ctx context.Context,
	vehicleID string,
	entry model.LogEntry,
	refs EntryReferences,
) (string, error) {
	var id string

	err := r.tx.GetContext(
		ctx,
		&id,
		upsertEntryQuery,
		vehicleID,
		string(entry.Kind),
		entry.SourceID,
		entry.OccurredAt,
		entry.Odometer,
		entry.Amount,
		entry.Title,
		entry.Note,
		refs.Station,
		refs.Place,
		refs.PaymentMethod,
		refs.Reason,
		refs.IncomeType,
		entry.MissedRefuel,
		entry.BatteryStartPct,
		entry.BatteryEndPct,
		entry.ChargeMinutes,
	)
	if err != nil {
		return "", fmt.Errorf("upserting the %s %d: %w", entry.Kind, entry.SourceID, err)
	}

	return id, nil
}

// DeleteAbsent deletes each entry of the kind whose identifier at the car log service is
// not kept, with its lines and its instalments.
func (r *Entries) DeleteAbsent(ctx context.Context, kind model.Kind, kept []int32) error {
	const query = `DELETE FROM log_entries WHERE kind = $1 AND source_id <> ALL($2::integer[]);`

	if _, err := r.tx.ExecContext(ctx, query, string(kind), kept); err != nil {
		return fmt.Errorf("deleting the absent entries of kind %s: %w", kind, err)
	}

	return nil
}

// entryColumns names the columns of a read of an entry, with the names of its references.
const entryColumns = `
	t.id, t.vehicle_id, t.kind, t.occurred_at, t.odometer, t.amount, t.title, t.note,
	st.name AS station, pl.name AS place, pm.name AS payment_method, re.name AS reason,
	it.name AS income_type, t.missed_refuel, t.battery_start_pct, t.battery_end_pct,
	t.charge_minutes`

// entryJoins joins the reference items that an entry names.
const entryJoins = `
	LEFT JOIN reference_items st ON st.id = t.station_id
	LEFT JOIN reference_items pl ON pl.id = t.place_id
	LEFT JOIN reference_items pm ON pm.id = t.payment_method_id
	LEFT JOIN reference_items re ON re.id = t.reason_id
	LEFT JOIN reference_items it ON it.id = t.income_type_id`

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
		FROM log_entries t` + entryJoins + `
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
		FROM log_entries t` + entryJoins + `
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
		FROM log_entries t` + entryJoins + `
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
