package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/oauth2"
)

// ErrRandomGeneration indicates that cryptographic random generation failed.
var ErrRandomGeneration = errors.New("auth: random generation failed")

// InitiateResult holds the values generated during OIDC flow initiation.
type InitiateResult struct {
	AuthURL  string
	State    string
	Nonce    string
	Verifier string
}

// IDTokenClaims represents claims extracted from an OIDC ID token.
//
//nolint:tagliatelle
type IDTokenClaims struct {
	Subject  string `json:"sub"`
	ID       string `json:"id"`
	Username string `json:"preferred_username"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
}

// OIDCFlow orchestrates the OIDC authorization code + PKCE flow.
// It is a stateless service, secrets are generated per call and returned
// for the caller to persist.
type OIDCFlow struct {
	oidcSvc *OIDCService
}

// NewOIDCFlow creates a new OIDCFlow service.
func NewOIDCFlow(oidcSvc *OIDCService) *OIDCFlow {
	return &OIDCFlow{
		oidcSvc: oidcSvc,
	}
}

// Initiate generates PKCE parameters, state, and nonce, then builds the
// authorization URL. The caller is responsible for persisting the returned
// values (typically in HTTP-only cookies).
func (f *OIDCFlow) Initiate() (*InitiateResult, error) {
	const randomBytesCount = 16

	state, err := generateRandomString(randomBytesCount)
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", ErrRandomGeneration)
	}

	nonce, err := generateRandomString(randomBytesCount)
	if err != nil {
		return nil, fmt.Errorf("generating nonce: %w", ErrRandomGeneration)
	}

	verifier := oauth2.GenerateVerifier()
	authURL := f.oidcSvc.AuthURL(state, nonce, verifier)

	return &InitiateResult{
		AuthURL:  authURL,
		State:    state,
		Nonce:    nonce,
		Verifier: verifier,
	}, nil
}

// Verify exchanges the authorization code for an OAuth2 token, verifies the
// ID token, and extracts claims. The nonce and verifier must match those
// generated during Initiate.
func (f *OIDCFlow) Verify(
	ctx context.Context,
	code string,
	nonce string,
	verifier string,
) (*IDTokenClaims, error) {
	oauth2Token, err := f.oidcSvc.Exchange(ctx, code, verifier)
	if err != nil {
		return nil, fmt.Errorf("exchanging code: %w", err)
	}

	idToken, err := f.oidcSvc.VerifyIDToken(ctx, oauth2Token, nonce)
	if err != nil {
		return nil, fmt.Errorf("verifying ID token: %w", err)
	}

	var claims IDTokenClaims
	if err = idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("extracting claims: %w", err)
	}

	return &claims, nil
}

func generateRandomString(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}
