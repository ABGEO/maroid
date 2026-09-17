package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

// ScopeFederatedID makes Dex put the connector and the upstream user identifier
// in the `federated_claims` claim.
const ScopeFederatedID = "federated:id"

var (
	// ErrNonceMismatch indicates that the OIDC nonce does not match the expected value.
	ErrNonceMismatch = errors.New("oidc: nonce mismatch")
	// ErrMissingIDToken indicates that the OAuth2 token response did not contain an id_token.
	ErrMissingIDToken = errors.New("oidc: missing id token")
)

// OIDCService handles OIDC authentication flows, including generating auth URLs,
// exchanging auth codes for tokens, and verifying ID tokens.
type OIDCService struct {
	cfg *config.Config

	provider     *oidc.Provider
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

// NewOIDCService creates a new OIDCService by initializing the OIDC provider,
// OAuth2 configuration, and ID token verifier based on the provided configuration.
func NewOIDCService(cfg *config.Config) (*OIDCService, error) {
	const timeout = 10 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, cfg.OIDC.Issuer)
	if err != nil {
		return nil, fmt.Errorf("creating OIDC provider: %w", err)
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.OIDC.ClientID,
		ClientSecret: cfg.OIDC.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.OIDC.RedirectURI,
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile, ScopeFederatedID},
	}
	oidcConfig := &oidc.Config{
		ClientID: cfg.OIDC.ClientID,
	}

	return &OIDCService{
		cfg: cfg,

		provider:     provider,
		oauth2Config: oauth2Config,
		verifier:     provider.Verifier(oidcConfig),
	}, nil
}

// Verifier returns the verifier of the provider. It holds the key set in memory.
func (s *OIDCService) Verifier() *oidc.IDTokenVerifier {
	return s.verifier
}

// VerifierForClient returns a verifier that accepts a token whose audience names
// the given client of Dex.
func (s *OIDCService) VerifierForClient(clientID string) *oidc.IDTokenVerifier {
	return s.provider.Verifier(&oidc.Config{ClientID: clientID})
}

// AuthURL generates the OIDC authentication URL with the specified state, nonce,
// and PKCE verifier.
//
// A provider names the connector of Dex. A request that names none reaches the
// connector list of Dex, which is the page that a person picks from.
func (s *OIDCService) AuthURL(state string, nonce string, verifier string, provider string) string {
	options := []oauth2.AuthCodeOption{
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("nonce", nonce),
	}

	if provider != "" {
		options = append(options, oauth2.SetAuthURLParam("connector_id", provider))
	}

	return s.oauth2Config.AuthCodeURL(state, options...)
}

// Exchange exchanges the authorization code for an OAuth2 token, using the provided PKCE verifier.
func (s *OIDCService) Exchange(
	ctx context.Context,
	code string,
	verifier string,
) (*oauth2.Token, error) {
	token, err := s.oauth2Config.Exchange(
		ctx,
		code,
		oauth2.VerifierOption(verifier),
	)
	if err != nil {
		return nil, fmt.Errorf("exchanging token: %w", err)
	}

	return token, nil
}

// VerifyIDToken extracts and verifies the ID token from an OAuth2 token response,
// and checks that the nonce matches the expected value.
func (s *OIDCService) VerifyIDToken(
	ctx context.Context,
	token *oauth2.Token,
	nonce string,
) (*oidc.IDToken, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, ErrMissingIDToken
	}

	idToken, err := s.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verifying ID token: %w", err)
	}

	if idToken.Nonce != nonce {
		return nil, ErrNonceMismatch
	}

	return idToken, nil
}
