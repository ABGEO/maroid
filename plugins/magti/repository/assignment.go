package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
)

// SubscriberAssignmentRepository defines the data access contract for the assigned
// subscriber of the acting workspace.
type SubscriberAssignmentRepository interface {
	Get(ctx context.Context) (model.Assignment, error)
	Put(
		ctx context.Context,
		number string,
		kind model.SubscriberKind,
		ifMatch *time.Time,
	) (model.Assignment, error)
	Delete(ctx context.Context) error
}

// SubscriberAssignment reads and writes the assigned subscriber of the acting workspace
// within one transaction.
type SubscriberAssignment struct {
	tx *sqlx.Tx
}

var _ SubscriberAssignmentRepository = (*SubscriberAssignment)(nil)

// NewSubscriberAssignment creates a new SubscriberAssignment over the transaction.
func NewSubscriberAssignment(tx *sqlx.Tx) *SubscriberAssignment {
	return &SubscriberAssignment{tx: tx}
}

const selectAssignment = `SELECT subscriber_number, kind, assigned_at FROM subscriber_assignments`

// Get answers the assignment, or errs.ErrNotAssigned when the workspace holds none.
func (r *SubscriberAssignment) Get(ctx context.Context) (model.Assignment, error) {
	var assignment model.Assignment

	err := r.tx.GetContext(ctx, &assignment, selectAssignment+`;`)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Assignment{}, errs.ErrNotAssigned
	}

	if err != nil {
		return model.Assignment{}, fmt.Errorf("selecting the subscriber assignment: %w", err)
	}

	return assignment, nil
}

// Put assigns the subscriber to the workspace, and replaces the assignment that exists.
// A non-nil ifMatch names the moment of the assignment that the client read, and a
// workspace whose assignment moved since then, or holds none, answers
// precondition.ErrModified and changes nothing.
func (r *SubscriberAssignment) Put(
	ctx context.Context,
	number string,
	kind model.SubscriberKind,
	ifMatch *time.Time,
) (model.Assignment, error) {
	if ifMatch != nil {
		if err := r.checkMoment(ctx, *ifMatch); err != nil {
			return model.Assignment{}, err
		}
	}

	const query = `
		INSERT INTO subscriber_assignments (subscriber_number, kind)
		VALUES (:subscriber_number, :kind)
		ON CONFLICT (workspace_id) DO UPDATE
		SET subscriber_number = EXCLUDED.subscriber_number,
		    kind              = EXCLUDED.kind,
		    assigned_at       = now()
		RETURNING subscriber_number, kind, assigned_at;`

	bound, values, err := sqlx.Named(query, map[string]any{
		paramSubscriberNumber: number,
		"kind":                string(kind),
	})
	if err != nil {
		return model.Assignment{}, fmt.Errorf("binding the subscriber assignment: %w", err)
	}

	var assignment model.Assignment
	if err = r.tx.GetContext(ctx, &assignment, r.tx.Rebind(bound), values...); err != nil {
		return model.Assignment{}, fmt.Errorf("upserting the subscriber assignment: %w", err)
	}

	return assignment, nil
}

// Delete removes the assignment. A workspace that holds none is left as it is.
func (r *SubscriberAssignment) Delete(ctx context.Context) error {
	if _, err := r.tx.ExecContext(ctx, `DELETE FROM subscriber_assignments;`); err != nil {
		return fmt.Errorf("deleting the subscriber assignment: %w", err)
	}

	return nil
}

// checkMoment locks the assignment, and answers precondition.ErrModified when its moment
// is not the one that the client read.
func (r *SubscriberAssignment) checkMoment(ctx context.Context, ifMatch time.Time) error {
	var assignment model.Assignment

	err := r.tx.GetContext(ctx, &assignment, selectAssignment+` FOR UPDATE;`)
	if errors.Is(err, sql.ErrNoRows) {
		return precondition.ErrModified
	}

	if err != nil {
		return fmt.Errorf("locking the subscriber assignment: %w", err)
	}

	if !assignment.AssignedAt.Equal(ifMatch) {
		return precondition.ErrModified
	}

	return nil
}
