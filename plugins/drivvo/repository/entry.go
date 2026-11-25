package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

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
