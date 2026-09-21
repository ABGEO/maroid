package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/oauth2"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

// ErrRandomGeneration indicates that cryptographic random generation failed.
var ErrRandomGeneration = errors.New("auth: random generation failed")

// OIDCFlow orchestrates the OIDC authorization code flow with PKCE.
//
// The row in public.auth_flows holds the state, the nonce, the
// verifier, and the target. A cookie that carried them is a value that the holder
// of the browser rewrites, and a rewritten value moves an attach to another
// record. The caller of Initiate therefore never sees a secret.
type OIDCFlow struct {
	oidcSvc  *OIDCService
	flowRepo repository.AuthFlowRepository
	flowTTL  time.Duration
}

// Session is the credential that a finished flow produces, and the claims that
// the IdP stated about the person.
type Session struct {
	AccessToken string
	Expiry      time.Time
	Claims      *Claims
}

// NewOIDCFlow creates a new OIDCFlow service.
func NewOIDCFlow(
	oidcSvc *OIDCService,
	flowRepo repository.AuthFlowRepository,
	flowTTL time.Duration,
) *OIDCFlow {
	return &OIDCFlow{
		oidcSvc:  oidcSvc,
		flowRepo: flowRepo,
		flowTTL:  flowTTL,
	}
}

// Initiate writes the authorization flow and returns the address of the provider
// with the binding that the browser must present at the callback.
//
// The caller gives the intent, the target, and the record or the invitation that
// the flow acts for. Initiate fills the state, the nonce, the verifier, and the
// expiry, because those never leave the hub. The binding is the one value that
// the caller puts in a cookie, and the row holds its digest alone.
func (f *OIDCFlow) Initiate(ctx context.Context, flow model.AuthFlow) (string, string, error) {
	const randomBytesCount = 16

	state, err := generateRandomString(randomBytesCount)
	if err != nil {
		return "", "", fmt.Errorf("generating state: %w", ErrRandomGeneration)
	}

	nonce, err := generateRandomString(randomBytesCount)
	if err != nil {
		return "", "", fmt.Errorf("generating nonce: %w", ErrRandomGeneration)
	}

	binding, err := generateRandomString(randomBytesCount)
	if err != nil {
		return "", "", fmt.Errorf("generating binding: %w", ErrRandomGeneration)
	}

	digest := sha256.Sum256([]byte(binding))

	flow.State = state
	flow.Nonce = nonce
	flow.BindingHash = digest[:]
	flow.Verifier = oauth2.GenerateVerifier()
	flow.ExpiresAt = time.Now().Add(f.flowTTL)

	if _, err = f.flowRepo.Create(ctx, flow); err != nil {
		return "", "", fmt.Errorf("writing the authorization flow: %w", err)
	}

	provider := ""
	if flow.Provider != nil {
		provider = *flow.Provider
	}

	return f.oidcSvc.AuthURL(flow.State, flow.Nonce, flow.Verifier, provider), binding, nil
}

// Consume spends the flow that the state names and returns it.
//
// The binding comes from the cookie that Initiate handed the browser. A state
// alone finishes nothing: without this check a person who holds a state finishes
// the flow in any browser, which signs the holder of that browser in as somebody
// else. The mismatch returns no row, because the request may be forged.
//
// An expired flow returns the row beside the error, because the caller reports
// the failure at the target that the row names.
func (f *OIDCFlow) Consume(
	ctx context.Context,
	state string,
	binding string,
) (*model.AuthFlow, error) {
	flow, err := f.flowRepo.ConsumeByState(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("consuming the authorization flow: %w", err)
	}

	digest := sha256.Sum256([]byte(binding))
	if subtle.ConstantTimeCompare(digest[:], flow.BindingHash) != 1 {
		return nil, fmt.Errorf(
			"consuming the authorization flow: %w",
			errs.ErrAuthFlowBindingMismatch,
		)
	}

	if time.Now().After(flow.ExpiresAt) {
		return flow, fmt.Errorf("consuming the authorization flow: %w", errs.ErrAuthFlowExpired)
	}

	return flow, nil
}

// Verify exchanges the authorization code, verifies the identity token, and
// returns the access token beside the claims. The nonce and the verifier come
// from the flow row.
func (f *OIDCFlow) Verify(
	ctx context.Context,
	code string,
	nonce string,
	verifier string,
) (*Session, error) {
	oauth2Token, err := f.oidcSvc.Exchange(ctx, code, verifier)
	if err != nil {
		return nil, fmt.Errorf("exchanging code: %w", err)
	}

	idToken, err := f.oidcSvc.VerifyIDToken(ctx, oauth2Token, nonce)
	if err != nil {
		return nil, fmt.Errorf("verifying id token: %w", err)
	}

	// at_hash binds the access token to the identity token that
	// the hub just verified, so the value that reaches the cookie is the one the
	// IdP minted for this exchange and not a value that arrived another way.
	if err = idToken.VerifyAccessToken(oauth2Token.AccessToken); err != nil {
		return nil, fmt.Errorf("binding the access token to the identity token: %w", err)
	}

	var claims Claims
	if err = idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("extracting claims: %w", err)
	}

	return &Session{
		AccessToken: oauth2Token.AccessToken,
		Expiry:      oauth2Token.Expiry,
		Claims:      &claims,
	}, nil
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
