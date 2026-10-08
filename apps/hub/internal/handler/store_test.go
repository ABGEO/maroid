package handler_test

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	hubdatabase "github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

// identityStore reads the identities that a test checks. Each call runs in a
// transaction of its own, as a service runs a repository.
type identityStore struct{ db *sqlx.DB }

func (s identityStore) ListByUser(ctx context.Context, userID string) ([]model.Identity, error) {
	identities, err := hubdatabase.FetchTx(ctx, s.db, func(tx *sqlx.Tx) ([]model.Identity, error) {
		return repository.NewIdentity(tx).ListByUser(ctx, userID)
	})
	if err != nil {
		return nil, fmt.Errorf("reading the identities in the test: %w", err)
	}

	return identities, nil
}
