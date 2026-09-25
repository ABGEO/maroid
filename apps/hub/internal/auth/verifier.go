package auth

import (
	"context"
	"fmt"
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
