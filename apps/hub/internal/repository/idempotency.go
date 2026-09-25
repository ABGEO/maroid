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
	Answer(ctx context.Context, key string) (*model.IdempotencyKey, error)
	Keep(ctx context.Context, entity *model.IdempotencyKey) error
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

// Keep stores the answer of one write. A key that the cache already holds keeps
// the answer it holds, because the first write is the one that a repeat reads.
func (r *Idempotency) Keep(ctx context.Context, entity *model.IdempotencyKey) error {
	query := `
		INSERT INTO public.idempotency_keys (key, request_hash, status, headers, body)
		VALUES (:key, :request_hash, :status, :headers, :body)
		ON CONFLICT (user_id, key) DO NOTHING;`

	if _, err := r.tx.NamedExecContext(ctx, query, entity); err != nil {
		return fmt.Errorf("writing the key cache: %w", err)
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
