// Package idempotency keeps the answer of a write, so that repeating that write
// is safe. It joins the key cache of the database to the middleware of
// libs/rest.
package idempotency

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/rest"
)

// Lifetime is how long the cache holds one answer.
const Lifetime = 24 * time.Hour

// Store reads and writes the key cache as the acting user.
type Store struct {
	db *sqlx.DB
}

var _ rest.IdempotencyStore = (*Store)(nil)

// NewStore creates a Store over the given database.
func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// Answer returns what the key holds, or rest.ErrNoIdempotentAnswer.
func (s *Store) Answer(ctx context.Context, key string) (rest.IdempotentAnswer, error) {
	var held *model.IdempotencyKey

	err := database.WithUserTx(ctx, s.db, func(tx *sqlx.Tx) error {
		var readErr error

		held, readErr = repository.NewIdempotency(tx).Answer(ctx, key)
		if readErr != nil {
			return fmt.Errorf("reading the stored answer: %w", readErr)
		}

		return nil
	})
	if err != nil {
		return rest.IdempotentAnswer{}, fmt.Errorf("reading the key cache: %w", err)
	}

	if held == nil {
		return rest.IdempotentAnswer{}, rest.ErrNoIdempotentAnswer
	}

	return rest.IdempotentAnswer{
		RequestHash: held.RequestHash,
		Status:      held.Status,
		Header:      http.Header(held.Headers),
		Body:        held.Body,
	}, nil
}

// Keep stores the answer of a write under the key.
func (s *Store) Keep(ctx context.Context, key string, answer rest.IdempotentAnswer) error {
	err := database.WithUserTx(ctx, s.db, func(tx *sqlx.Tx) error {
		return repository.NewIdempotency(tx).Keep(ctx, &model.IdempotencyKey{
			Key:         key,
			RequestHash: answer.RequestHash,
			Status:      answer.Status,
			Headers:     model.Headers(answer.Header),
			Body:        answer.Body,
		})
	})
	if err != nil {
		return fmt.Errorf("writing the key cache: %w", err)
	}

	return nil
}
