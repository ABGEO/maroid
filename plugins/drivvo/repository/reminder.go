package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// Reminders reads and writes the reminders of the acting workspace within one transaction.
type Reminders struct {
	tx *sqlx.Tx
}

// NewReminders creates a new Reminders over the transaction.
func NewReminders(tx *sqlx.Tx) *Reminders {
	return &Reminders{tx: tx}
}

// Upsert stores the reminder of the vehicle by its identifier at the car log service.
// itemID names the row of its type, nil for none.
func (r *Reminders) Upsert(
	ctx context.Context,
	vehicleID string,
	reminder model.Reminder,
	itemID *string,
) error {
	const query = `
		INSERT INTO reminders (
			vehicle_id, source_id, kind, item_id, due_on, due_odometer, repeats,
			repeat_every, repeat_unit, repeat_distance, note
		)
		VALUES ($1, $2, $3, $4, $5::date, $6, $7, $8, $9, $10, NULLIF($11, ''))
		ON CONFLICT ON CONSTRAINT reminders_source_key DO UPDATE
		SET vehicle_id = excluded.vehicle_id,
		    kind = excluded.kind,
		    item_id = excluded.item_id,
		    due_on = excluded.due_on,
		    due_odometer = excluded.due_odometer,
		    repeats = excluded.repeats,
		    repeat_every = excluded.repeat_every,
		    repeat_unit = excluded.repeat_unit,
		    repeat_distance = excluded.repeat_distance,
		    note = excluded.note,
		    collected_at = now();`

	var dueOn, repeatUnit *string

	if reminder.DueOn != nil {
		day := reminder.DueOn.Format(time.DateOnly)
		dueOn = &day
	}

	if reminder.RepeatUnit != nil {
		unit := string(*reminder.RepeatUnit)
		repeatUnit = &unit
	}

	_, err := r.tx.ExecContext(
		ctx,
		query,
		vehicleID,
		reminder.SourceID,
		string(reminder.Kind),
		itemID,
		dueOn,
		reminder.DueOdometer,
		reminder.Repeats,
		reminder.RepeatEvery,
		repeatUnit,
		reminder.RepeatDistance,
		reminder.Note,
	)
	if err != nil {
		return fmt.Errorf("upserting the reminder %d: %w", reminder.SourceID, err)
	}

	return nil
}

// DeleteAbsent deletes each reminder whose identifier at the car log service is not kept.
func (r *Reminders) DeleteAbsent(ctx context.Context, kept []int32) error {
	const query = `DELETE FROM reminders WHERE source_id <> ALL($1::integer[]);`

	if _, err := r.tx.ExecContext(ctx, query, kept); err != nil {
		return fmt.Errorf("deleting the absent reminders: %w", err)
	}

	return nil
}
