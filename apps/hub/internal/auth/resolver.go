package auth

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

// The connectors of Dex whose identifier the hub fixes.
const (
	// ProviderTelegram is the connector of Dex that federates Telegram.
	ProviderTelegram = "telegram"
	// ProviderLocal is the connector of Dex that holds the local accounts.
	ProviderLocal = "local"
)

// IdentityResolver reads the user record that one external account names.
type IdentityResolver interface {
	ResolveByProvider(
		ctx context.Context,
		provider string,
		providerUserID string,
	) (*model.User, error)
}

// Resolver is the identity backed implementation of IdentityResolver.
type Resolver struct {
	db *sqlx.DB
}

var _ IdentityResolver = (*Resolver)(nil)

// NewResolver creates a new Resolver instance.
func NewResolver(db *sqlx.DB) *Resolver {
	return &Resolver{db: db}
}

// ResolveByProvider returns the active user record that the external account names.
func (r *Resolver) ResolveByProvider(
	ctx context.Context,
	provider string,
	providerUserID string,
) (*model.User, error) {
	user, err := database.FetchTx(ctx, r.db, func(tx *sqlx.Tx) (*model.User, error) {
		return repository.NewIdentity(tx).GetActiveUserByProvider(ctx, provider, providerUserID)
	})
	if err != nil {
		return nil, fmt.Errorf("resolving the acting user: %w", err)
	}

	return user, nil
}
