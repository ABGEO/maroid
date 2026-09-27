package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/model"
)

// IdempotencyRepository defines the data access contract for the key cache.
type IdempotencyRepository interface {
	Claim(ctx context.Context, key string, hash string, lease time.Duration) (bool, error)
	Answer(ctx context.Context, key string) (*model.IdempotencyKey, error)
	Complete(ctx context.Context, entity *model.IdempotencyKey) error
	Release(ctx context.Context, key string) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

// Idempotency is a SQL based implementation of IdempotencyRepository.
type Idempotency struct {
	tx *sqlx.Tx
}

var _ IdempotencyRepository = (*Idempotency)(nil)

// NewIdempotency creates a new Idempotency repository instance.
func NewIdempotency(tx *sqlx.Tx) *Idempotency {
	return &Idempotency{tx: tx}
}

// Claim writes a pending row for the key, and answers whether this request
// holds it. A pending row whose write stopped longer than lease ago is taken
// over, because a hub that died between the claim and the answer never frees
// its key. A row that another request holds, or that holds an answer, stays.
func (r *Idempotency) Claim(
	ctx context.Context,
	key string,
	hash string,
	lease time.Duration,
) (bool, error) {
	query := `
		INSERT INTO public.idempotency_keys (key, request_hash, status, body)
		VALUES ($1, $2, $3, ''::bytea)
		ON CONFLICT (user_id, key) DO UPDATE
			SET request_hash = EXCLUDED.request_hash, created_at = now()
			WHERE public.idempotency_keys.status = $3
			  AND public.idempotency_keys.updated_at < now() - $4 * interval '1 second'
		RETURNING id;`

	var id string

	err := r.tx.GetContext(ctx, &id, query, key, hash, model.IdempotencyPending, lease.Seconds())
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("claiming the key: %w", err)
	}

	return true, nil
}

// Answer returns what the acting user stored under the key, or a nil entity when
// the cache holds nothing.
func (r *Idempotency) Answer(ctx context.Context, key string) (*model.IdempotencyKey, error) {
	var entity model.IdempotencyKey

	query := `
		SELECT id, user_id, key, request_hash, status, headers, body, created_at, updated_at
		FROM public.idempotency_keys
		WHERE key = $1;`

	if err := r.tx.GetContext(ctx, &entity, query, key); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // an absent key is not a failure.
		}

		return nil, fmt.Errorf("reading the key cache: %w", err)
	}

	return &entity, nil
}

// Complete writes the answer onto the pending row that the same request
// claimed. A row that another request took over keeps what that request writes.
func (r *Idempotency) Complete(ctx context.Context, entity *model.IdempotencyKey) error {
	query := `
		UPDATE public.idempotency_keys
		SET status = :status, headers = :headers, body = :body
		WHERE key = :key AND request_hash = :request_hash AND status = :pending;`

	if _, err := r.tx.NamedExecContext(ctx, query, map[string]any{
		"key":          entity.Key,
		"request_hash": entity.RequestHash,
		"status":       entity.Status,
		"headers":      entity.Headers,
		"body":         entity.Body,
		"pending":      model.IdempotencyPending,
	}); err != nil {
		return fmt.Errorf("writing the key cache: %w", err)
	}

	return nil
}

// Release removes the pending row of a write that failed, so that a repeat
// runs it again. A row that holds an answer stays.
func (r *Idempotency) Release(ctx context.Context, key string) error {
	query := `DELETE FROM public.idempotency_keys WHERE key = $1 AND status = $2;`

	if _, err := r.tx.ExecContext(ctx, query, key, model.IdempotencyPending); err != nil {
		return fmt.Errorf("freeing the key: %w", err)
	}

	return nil
}

// DeleteExpired removes every row that the cache no longer needs.
func (r *Idempotency) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.tx.ExecContext(
		ctx, `DELETE FROM public.idempotency_keys WHERE created_at < $1;`, before,
	)
	if err != nil {
		return 0, fmt.Errorf("removing the expired keys: %w", err)
	}

	removed, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("counting the removed keys: %w", err)
	}

	return removed, nil
}
