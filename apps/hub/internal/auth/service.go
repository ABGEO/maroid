package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

// Service holds the operations that change the identities of a user record.
type Service struct {
	db           *sqlx.DB
	identityRepo repository.IdentityRepository
}

// NewService creates a new Service instance.
func NewService(db *sqlx.DB, identityRepo repository.IdentityRepository) *Service {
	return &Service{db: db, identityRepo: identityRepo}
}

// Attach binds the external account to the user record.
func (s *Service) Attach(
	ctx context.Context,
	userID string,
	provider string,
	providerUserID string,
	profile model.Profile,
) error {
	err := database.WithTx(ctx, s.db, func(tx *sqlx.Tx) error {
		return s.identityRepo.Attach(ctx, tx, userID, provider, providerUserID, profile)
	})

	if errors.Is(err, errs.ErrIdentityTaken) {
		return s.reportConflict(ctx, userID, provider, providerUserID, err)
	}

	if err != nil {
		return fmt.Errorf("attaching the external account: %w", err)
	}

	return nil
}

// Detach removes the identity of the provider from the user record.
func (s *Service) Detach(ctx context.Context, userID string, provider string) error {
	if err := s.identityRepo.Detach(ctx, userID, provider); err != nil {
		return fmt.Errorf("detaching the external account: %w", err)
	}

	return nil
}

// reportConflict tells an account that this record already holds apart from one
// that names another record.
func (s *Service) reportConflict(
	ctx context.Context,
	userID string,
	provider string,
	providerUserID string,
	conflict error,
) error {
	owner, err := s.identityRepo.GetActiveUserByProvider(ctx, provider, providerUserID)
	if err == nil && owner.ID == userID {
		return nil
	}

	return fmt.Errorf("attaching the external account: %w", conflict)
}
