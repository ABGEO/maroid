package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/model"
)

const fileTaskColumns = `id, file_id, scope, owner_id, object_key, action, run_after,
	locked_until, attempts, last_error, created_at`

// FileTaskRepository defines the data access contract for the outbox of the files.
// The table is shared, so a transaction of any scope reaches every task.
type FileTaskRepository interface {
	Enqueue(ctx context.Context, drafts ...model.FileTaskDraft) error
	EnqueueOnce(ctx context.Context, draft model.FileTaskDraft) error
	Lease(ctx context.Context, limit int, lease time.Duration) ([]model.FileTask, error)
	Done(ctx context.Context, id string) error
	Retry(ctx context.Context, id string, delay time.Duration, lastError string) error
}

// FileTask is a SQL based implementation of FileTaskRepository.
type FileTask struct {
	tx *sqlx.Tx
}

var _ FileTaskRepository = (*FileTask)(nil)

// NewFileTask creates a new FileTask repository instance.
func NewFileTask(tx *sqlx.Tx) *FileTask {
	return &FileTask{tx: tx}
}

// Enqueue writes each task, due after its delay from the clock of the database.
func (r *FileTask) Enqueue(ctx context.Context, drafts ...model.FileTaskDraft) error {
	for _, draft := range drafts {
		_, err := r.tx.ExecContext(ctx, `
			INSERT INTO public.file_tasks (file_id, scope, owner_id, object_key, action, run_after)
			VALUES ($1, $2, $3, $4, $5, now() + make_interval(secs => $6));`,
			draft.FileID, draft.Scope, draft.OwnerID, draft.ObjectKey, draft.Action,
			draft.Delay.Seconds(),
		)
		if err != nil {
			return fmt.Errorf("enqueuing a FileTask %s: %w", draft.Action, err)
		}
	}

	return nil
}

// EnqueueOnce writes the task unless a task of the same action already waits for the
// file.
func (r *FileTask) EnqueueOnce(ctx context.Context, draft model.FileTaskDraft) error {
	_, err := r.tx.ExecContext(ctx, `
		INSERT INTO public.file_tasks (file_id, scope, owner_id, object_key, action, run_after)
		SELECT $1::uuid, $2::text, $3::uuid, $4::text, $5::text, now() + make_interval(secs => $6)
		WHERE NOT EXISTS (
			SELECT 1 FROM public.file_tasks WHERE file_id = $1::uuid AND action = $5::text
		);`,
		draft.FileID, draft.Scope, draft.OwnerID, draft.ObjectKey, draft.Action,
		draft.Delay.Seconds(),
	)
	if err != nil {
		return fmt.Errorf("enqueuing a FileTask %s once: %w", draft.Action, err)
	}

	return nil
}

// Lease takes at most limit due tasks that no worker holds, in the order they fall
// due, and holds each one for the lease. A task that another transaction locks stays
// with it. Each lease counts as one attempt.
func (r *FileTask) Lease(
	ctx context.Context,
	limit int,
	lease time.Duration,
) ([]model.FileTask, error) {
	tasks := []model.FileTask{}

	err := r.tx.SelectContext(ctx, &tasks, `
		UPDATE public.file_tasks
		SET locked_until = now() + make_interval(secs => $2), attempts = attempts + 1
		WHERE id IN (
			SELECT id FROM public.file_tasks
			WHERE run_after <= now() AND (locked_until IS NULL OR locked_until <= now())
			ORDER BY run_after
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING `+fileTaskColumns+`;`, limit, lease.Seconds())
	if err != nil {
		return nil, fmt.Errorf("leasing the FileTasks: %w", err)
	}

	return tasks, nil
}

// Done removes the task.
func (r *FileTask) Done(ctx context.Context, id string) error {
	if _, err := r.tx.ExecContext(
		ctx,
		`DELETE FROM public.file_tasks WHERE id = $1;`,
		id,
	); err != nil {
		return fmt.Errorf("removing a FileTask: %w", err)
	}

	return nil
}

// Retry releases the task, due again after the delay, and keeps the error of the attempt.
func (r *FileTask) Retry(
	ctx context.Context,
	id string,
	delay time.Duration,
	lastError string,
) error {
	_, err := r.tx.ExecContext(ctx, `
		UPDATE public.file_tasks
		SET run_after = now() + make_interval(secs => $2), locked_until = NULL, last_error = $3
		WHERE id = $1;`, id, delay.Seconds(), lastError)
	if err != nil {
		return fmt.Errorf("releasing a FileTask: %w", err)
	}

	return nil
}
