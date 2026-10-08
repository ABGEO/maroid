package auth_test

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

// identityStore reads the identities that a test checks. Each call runs in a
// transaction of its own, as a service runs a repository.
type identityStore struct{ db *sqlx.DB }

func (s identityStore) ListByUser(ctx context.Context, userID string) ([]model.Identity, error) {
	return inTx(ctx, s.db, func(tx *sqlx.Tx) ([]model.Identity, error) {
		return repository.NewIdentity(tx).ListByUser(ctx, userID)
	})
}

func (s identityStore) GetActiveUserByProvider(
	ctx context.Context,
	provider string,
	providerUserID string,
) (*model.User, error) {
	return inTx(ctx, s.db, func(tx *sqlx.Tx) (*model.User, error) {
		return repository.NewIdentity(tx).GetActiveUserByProvider(ctx, provider, providerUserID)
	})
}

// invitationStore reads the invitations that a test checks.
type invitationStore struct{ db *sqlx.DB }

func (s invitationStore) GetValidByTokenHash(
	ctx context.Context,
	tokenHash []byte,
) (*model.Invitation, error) {
	return inTx(ctx, s.db, func(tx *sqlx.Tx) (*model.Invitation, error) {
		return repository.NewInvitation(tx).GetValidByTokenHash(ctx, tokenHash)
	})
}

// flowStore reads and writes the authorization flows that a test checks.
type flowStore struct{ db *sqlx.DB }

func (s flowStore) Create(ctx context.Context, flow model.AuthFlow) (*model.AuthFlow, error) {
	return inTx(ctx, s.db, func(tx *sqlx.Tx) (*model.AuthFlow, error) {
		return repository.NewAuthFlow(tx).Create(ctx, flow)
	})
}

func (s flowStore) ConsumeByState(ctx context.Context, state string) (*model.AuthFlow, error) {
	return inTx(ctx, s.db, func(tx *sqlx.Tx) (*model.AuthFlow, error) {
		return repository.NewAuthFlow(tx).ConsumeByState(ctx, state)
	})
}

func inTx[T any](ctx context.Context, db *sqlx.DB, load func(*sqlx.Tx) (T, error)) (T, error) {
	result, err := database.FetchTx(ctx, db, load)
	if err != nil {
		return result, fmt.Errorf("reading in the test: %w", err)
	}

	return result, nil
}
