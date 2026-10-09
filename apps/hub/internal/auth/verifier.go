package auth

import (
	"context"
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/model"
)

// FederatedClaims names the connector of Dex and the account at the upstream
// provider.
type FederatedClaims struct {
	ConnectorID string `json:"connector_id"`
	UserID      string `json:"user_id"`
}

// Claims are the claims of a token that Dex issued.
type Claims struct {
	Subject   string          `json:"sub"`
	Name      string          `json:"name"`
	Username  string          `json:"preferred_username"`
	Picture   string          `json:"picture"`
	Federated FederatedClaims `json:"federated_claims"`
}

// Profile answers the profile that the token gives the identity it names. Dex fills
// no preferred_username for a local account and sends the name of its password, which
// the hub sets to the address, so the address stays the username of a local identity.
func (c *Claims) Profile() model.Profile {
	username := c.Username
	if c.Federated.ConnectorID == ProviderLocal {
		username = c.Name
	}

	return model.Profile{
		Username:    username,
		DisplayName: c.Name,
		PictureURL:  c.Picture,
	}
}

// TokenVerifier checks a token that Dex issued.
type TokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (*Claims, error)
}

// Verifier is the Dex backed implementation of TokenVerifier.
type Verifier struct {
	oidcSvc *OIDCService
}

var _ TokenVerifier = (*Verifier)(nil)

// NewTokenVerifier creates a new Verifier instance.
func NewTokenVerifier(oidcSvc *OIDCService) *Verifier {
	return &Verifier{oidcSvc: oidcSvc}
}

// Verify checks the signature, the issuer, the audience, and the expiry, then
// returns the claims.
func (v *Verifier) Verify(ctx context.Context, rawToken string) (*Claims, error) {
	idToken, err := v.oidcSvc.Verifier().Verify(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("verifying the token: %w", err)
	}

	var claims Claims
	if err = idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("reading the claims of the token: %w", err)
	}

	return &claims, nil
}
