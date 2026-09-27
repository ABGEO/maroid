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
	"github.com/abgeo/maroid/libs/rest/idempotency"
)

const (
	// Lifetime is how long the cache holds one answer.
	Lifetime = 24 * time.Hour

	// PendingLease is how long a claim holds a key before another request takes it
	// over. It passes the write timeout of the server, so only a claim whose hub
	// died reaches it.
	PendingLease = time.Minute
)

// Store reads and writes the key cache as the acting user.
type Store struct {
	db *sqlx.DB
}

var _ idempotency.Store = (*Store)(nil)

// NewStore creates a Store over the given database.
func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// Reserve claims the key for this request. It answers nil when the request
// holds the key, the answer of a finished write, or idempotency.ErrInProgress
// while another request holds it.
func (s *Store) Reserve(ctx context.Context, key string, hash string) (*idempotency.Answer, error) {
	var (
		claimed bool
		held    *model.IdempotencyKey
	)

	err := database.WithUserTx(ctx, s.db, func(tx *sqlx.Tx) error {
		repo := repository.NewIdempotency(tx)

		var claimErr error

		claimed, claimErr = repo.Claim(ctx, key, hash, PendingLease)
		if claimErr != nil {
			return fmt.Errorf("writing the claim: %w", claimErr)
		}

		if claimed {
			return nil
		}

		var readErr error

		held, readErr = repo.Answer(ctx, key)
		if readErr != nil {
			return fmt.Errorf("reading the held key: %w", readErr)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("claiming the key: %w", err)
	}

	switch {
	case claimed:
		return nil, nil //nolint:nilnil // a claimed key holds no answer yet.
	case held == nil, held.Status == model.IdempotencyPending:
		// No row after a failed claim is a row that its request freed in
		// between. The client sends the write again, and that claim holds.
		return nil, idempotency.ErrInProgress
	}

	return &idempotency.Answer{
		RequestHash: held.RequestHash,
		Status:      held.Status,
		Header:      http.Header(held.Headers),
		Body:        held.Body,
	}, nil
}

// Complete stores the answer of the write that claimed the key.
func (s *Store) Complete(ctx context.Context, key string, answer idempotency.Answer) error {
	err := database.WithUserTx(ctx, s.db, func(tx *sqlx.Tx) error {
		return repository.NewIdempotency(tx).Complete(ctx, &model.IdempotencyKey{
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

// Release frees the key of a write that failed.
func (s *Store) Release(ctx context.Context, key string) error {
	err := database.WithUserTx(ctx, s.db, func(tx *sqlx.Tx) error {
		return repository.NewIdempotency(tx).Release(ctx, key)
	})
	if err != nil {
		return fmt.Errorf("freeing the key: %w", err)
	}

	return nil
}
