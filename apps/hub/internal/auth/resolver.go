package auth

import (
	"context"
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

// ProviderTelegram is the connector of Dex that federates Telegram.
const ProviderTelegram = "telegram"

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
	identityRepo repository.IdentityRepository
}

var _ IdentityResolver = (*Resolver)(nil)

// NewResolver creates a new Resolver instance.
func NewResolver(identityRepo repository.IdentityRepository) *Resolver {
	return &Resolver{identityRepo: identityRepo}
}

// ResolveByProvider returns the active user record that the external account names.
func (r *Resolver) ResolveByProvider(
	ctx context.Context,
	provider string,
	providerUserID string,
) (*model.User, error) {
	user, err := r.identityRepo.GetActiveUserByProvider(ctx, provider, providerUserID)
	if err != nil {
		return nil, fmt.Errorf("resolving the acting user: %w", err)
	}

	return user, nil
}
