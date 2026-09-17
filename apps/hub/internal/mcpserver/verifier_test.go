package mcpserver_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authtest"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/mcpserver"
	"github.com/abgeo/maroid/apps/hub/internal/model"
)

const (
	recordID = "01998aa0-1111-7000-8000-000000000001"
	account  = "722183546"
	// mcpClientID is the default that mcp.client_id carries. See CFG-003.
	mcpClientID = "mcp"
)

// stubResolver answers with the record that the test gives, or with the error,
// and it reports what the verifier asked for.
type stubResolver struct {
	user           *model.User
	err            error
	calls          int
	gotProvider    string
	gotProviderUID string
}

var _ auth.IdentityResolver = (*stubResolver)(nil)

func (s *stubResolver) ResolveByProvider(
	_ context.Context,
	provider string,
	providerUserID string,
) (*model.User, error) {
	s.calls++
	s.gotProvider = provider
	s.gotProviderUID = providerUserID

	return s.user, s.err
}

func activeRecord() *model.User {
	first, last := "Temuri", "Takalandze"

	return &model.User{
		ID:        recordID,
		FirstName: &first,
		LastName:  &last,
		Status:    model.StatusActive,
	}
}

// mcpClaims mints the claim set that Dex gives a token of the MCP client.
func mcpClaims(dex *authtest.Provider) jwt.MapClaims {
	claims := dex.Claims(auth.ProviderTelegram, account)
	claims["aud"] = mcpClientID

	return claims
}

func verifierFor(
	t *testing.T,
	dex *authtest.Provider,
	resolver auth.IdentityResolver,
	clientID string,
) mcpauth.TokenVerifier {
	t.Helper()

	cfg := &config.Config{}
	cfg.OIDC.Issuer = dex.URL
	cfg.OIDC.ClientID = authtest.ClientID
	cfg.OIDC.ClientSecret = "secret"
	cfg.OIDC.RedirectURI = "https://hub.maroid.localhost/auth/callback"

	service, err := auth.NewOIDCService(cfg)
	require.NoError(t, err)

	return mcpserver.NewTokenVerifier(service, resolver, clientID)
}

// MCPHUB-SC-003: A token whose identity holds an active user record resolves to
// that record, and the token info carries it. MCPHUB-INV-001 holds: one tool call
// resolves to exactly one acting user.
func TestTheVerifierResolvesTheActingUser(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	resolver := &stubResolver{user: activeRecord()}

	info, err := verifierFor(t, dex, resolver, mcpClientID)(
		t.Context(), dex.SignClaims(t, mcpClaims(dex)), nil,
	)

	require.NoError(t, err)
	require.Equal(t, recordID, info.UserID)
	require.False(t, info.Expiration.IsZero())
	require.Equal(t, activeRecord(), mcpserver.UserFromTokenInfo(info))
	require.Equal(
		t,
		auth.ProviderTelegram,
		mcpserver.ClaimsFromTokenInfo(info).Federated.ConnectorID,
	)
	require.Equal(t, auth.ProviderTelegram, resolver.gotProvider)
	require.Equal(t, account, resolver.gotProviderUID)
}

// MCPHUB-SC-002: A token that fails verification reaches no tool. SEC-002 checks
// the signature, the issuer, the audience, and the expiry, and MCPHUB-DD-001
// scopes the audience to the MCP client of Dex.
func TestTheVerifierRefusesATokenThatDexDidNotMintForTheMCPClient(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	other := authtest.StartProvider(t)
	resolver := &stubResolver{user: activeRecord()}
	verify := verifierFor(t, dex, resolver, mcpClientID)

	cases := map[string]func() string{
		"a token of another issuer": func() string {
			claims := other.Claims(auth.ProviderTelegram, account)
			claims["aud"] = mcpClientID

			return other.SignClaims(t, claims)
		},
		"a token for the audience of the web shell": func() string {
			return dex.Sign(t, auth.ProviderTelegram, account)
		},
		"a token that expired one second ago": func() string {
			claims := mcpClaims(dex)
			claims["exp"] = time.Now().Add(-time.Second).Unix()

			return dex.SignClaims(t, claims)
		},
		"a token with no federated claims": func() string {
			claims := mcpClaims(dex)
			delete(claims, "federated_claims")

			return dex.SignClaims(t, claims)
		},
	}

	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			info, err := verify(t.Context(), build(), nil)

			require.Nil(t, info)
			require.ErrorIs(t, err, mcpauth.ErrInvalidToken, "the middleware answers 401")
		})
	}
}

// MCPHUB-SC-002: The audience that the verifier demands is the one that
// mcp.client_id holds, not a literal in the code. MCPHUB-DD-001 and CFG-003
// give the field, so a deployment whose IdP names that client otherwise needs no
// code change.
func TestTheVerifierDemandsTheConfiguredAudience(t *testing.T) {
	t.Parallel()

	const configured = "maroid-agents"

	dex := authtest.StartProvider(t)
	resolver := &stubResolver{user: activeRecord()}
	verify := verifierFor(t, dex, resolver, configured)

	claims := dex.Claims(auth.ProviderTelegram, account)
	claims["aud"] = configured

	info, err := verify(t.Context(), dex.SignClaims(t, claims), nil)

	require.NoError(t, err)
	require.Equal(t, recordID, info.UserID)

	refused, err := verify(t.Context(), dex.SignClaims(t, mcpClaims(dex)), nil)

	require.Nil(t, refused)
	require.ErrorIs(t, err, mcpauth.ErrInvalidToken, "the default audience no longer passes")
}

// MCPHUB-SC-003: A token whose identity holds no active user record reaches no
// tool, and the refusal reads the same as a failed verification, so a caller
// learns nothing about which account exists. SEC-004 gives the gate.
func TestTheVerifierRefusesAnIdentityWithNoActiveUserRecord(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	verify := verifierFor(t, dex, &stubResolver{err: errs.ErrUserNotFound}, mcpClientID)

	info, unknownErr := verify(t.Context(), dex.SignClaims(t, mcpClaims(dex)), nil)

	require.Nil(t, info)
	require.ErrorIs(t, unknownErr, mcpauth.ErrInvalidToken)

	_, forgedErr := verifierFor(t, dex, &stubResolver{err: errs.ErrUserNotFound}, mcpClientID)(
		t.Context(), "not-a-token", nil,
	)

	require.Equal(t, forgedErr.Error(), unknownErr.Error(), "one message for both refusals")
}

// MCPHUB-SC-007: Over 1000 consecutive calls that carry a valid token, the hub
// reads the key set of Dex once or never. MCPHUB-NFR-001 gives the limit.
func TestTheVerifierReadsTheKeySetOnce(t *testing.T) {
	t.Parallel()

	const calls = 1000

	dex := authtest.StartProvider(t)
	resolver := &stubResolver{user: activeRecord()}
	verify := verifierFor(t, dex, resolver, mcpClientID)
	token := dex.SignClaims(t, mcpClaims(dex))

	for range calls {
		info, err := verify(t.Context(), token, nil)
		require.NoError(t, err)
		require.Equal(t, recordID, info.UserID)
	}

	require.Equal(t, calls, resolver.calls, "the record is read on every call")
	require.LessOrEqual(t, dex.KeyRequests(), int64(1), "the key set is read once or never")
}
