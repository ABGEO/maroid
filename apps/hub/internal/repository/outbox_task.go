package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/model"
)

const outboxTaskColumns = `id, type, payload, owner_scope, owner_id, dedupe_key, run_after,
	locked_until, attempts, last_error, created_at`

// OutboxTaskRepository defines the data access contract for the outbox of the hub.
// The table is shared, so a transaction of any scope reaches every task.
type OutboxTaskRepository interface {
	Enqueue(ctx context.Context, drafts ...model.OutboxDraft) error
	Lease(ctx context.Context, limit int, lease time.Duration) ([]model.OutboxTask, error)
	Done(ctx context.Context, id string) error
	Retry(ctx context.Context, id string, delay time.Duration, lastError string) error
	Drop(ctx context.Context, owner model.OutboxOwner, types ...string) error
}

// OutboxTask is a SQL based implementation of OutboxTaskRepository.
type OutboxTask struct {
	tx *sqlx.Tx
}

var _ OutboxTaskRepository = (*OutboxTask)(nil)

// NewOutboxTask creates a new OutboxTask repository instance.
func NewOutboxTask(tx *sqlx.Tx) *OutboxTask {
	return &OutboxTask{tx: tx}
}

// Enqueue writes each task, due after its delay from the clock of the database. A
// draft whose dedupe key a task already holds adds nothing.
func (r *OutboxTask) Enqueue(ctx context.Context, drafts ...model.OutboxDraft) error {
	for _, draft := range drafts {
		payload, err := payloadOf(draft.Payload)
		if err != nil {
			return fmt.Errorf("encoding the payload of an OutboxTask %s: %w", draft.Type, err)
		}

		var scope, owner, dedupe *string

		if draft.Owner != nil {
			held := string(draft.Owner.Scope)
			scope, owner = &held, &draft.Owner.ID
		}

		if draft.DedupeKey != "" {
			dedupe = &draft.DedupeKey
		}

		_, err = r.tx.ExecContext(ctx, `
			INSERT INTO public.outbox_tasks (type, payload, owner_scope, owner_id, dedupe_key, run_after)
			VALUES ($1, $2, $3, $4, $5, now() + make_interval(secs => $6))
			ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING;`,
			draft.Type, payload, scope, owner, dedupe, draft.Delay.Seconds(),
		)
		if err != nil {
			return fmt.Errorf("enqueuing an OutboxTask %s: %w", draft.Type, err)
		}
	}

	return nil
}

// Lease takes at most limit due tasks that no worker holds, in the order they fall
// due, and holds each one for the lease. A task that another transaction locks stays
// with it. Each lease counts as one attempt.
func (r *OutboxTask) Lease(
	ctx context.Context,
	limit int,
	lease time.Duration,
) ([]model.OutboxTask, error) {
	tasks := []model.OutboxTask{}

	err := r.tx.SelectContext(ctx, &tasks, `
		UPDATE public.outbox_tasks
		SET locked_until = now() + make_interval(secs => $2), attempts = attempts + 1
		WHERE id IN (
			SELECT id FROM public.outbox_tasks
			WHERE run_after <= now() AND (locked_until IS NULL OR locked_until <= now())
			ORDER BY run_after
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING `+outboxTaskColumns+`;`, limit, lease.Seconds())
	if err != nil {
		return nil, fmt.Errorf("leasing the OutboxTasks: %w", err)
	}

	return tasks, nil
}

// Done removes the task.
func (r *OutboxTask) Done(ctx context.Context, id string) error {
	if _, err := r.tx.ExecContext(
		ctx,
		`DELETE FROM public.outbox_tasks WHERE id = $1;`,
		id,
	); err != nil {
		return fmt.Errorf("removing an OutboxTask: %w", err)
	}

	return nil
}

// Retry releases the task, due again after the delay, and keeps the error of the attempt.
func (r *OutboxTask) Retry(
	ctx context.Context,
	id string,
	delay time.Duration,
	lastError string,
) error {
	_, err := r.tx.ExecContext(ctx, `
		UPDATE public.outbox_tasks
		SET run_after = now() + make_interval(secs => $2), locked_until = NULL, last_error = $3
		WHERE id = $1;`, id, delay.Seconds(), lastError)
	if err != nil {
		return fmt.Errorf("releasing an OutboxTask: %w", err)
	}

	return nil
}

// Drop removes every task of the owner whose type is one of types, leased or not. A
// leased task that ends later finds no row, so its result changes nothing.
func (r *OutboxTask) Drop(ctx context.Context, owner model.OutboxOwner, types ...string) error {
	_, err := r.tx.ExecContext(ctx, `
		DELETE FROM public.outbox_tasks
		WHERE owner_scope = $1 AND owner_id = $2 AND type = ANY($3::text[]);`,
		owner.Scope, owner.ID, types)
	if err != nil {
		return fmt.Errorf("dropping the OutboxTasks of an owner: %w", err)
	}

	return nil
}

// payloadOf encodes the payload of a draft. A draft with no payload holds an empty
// object, as the column demands.
func payloadOf(payload any) ([]byte, error) {
	if payload == nil {
		return []byte(`{}`), nil
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling: %w", err)
	}

	return encoded, nil
}
