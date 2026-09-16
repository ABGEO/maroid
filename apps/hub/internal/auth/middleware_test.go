package auth_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authtest"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	recordID = "01998aa0-1111-7000-8000-000000000001"
	account  = "722183546"
)

// fakeResolver answers with the record that the test gives, or with the error,
// and it reports what the middleware asked for.
type fakeResolver struct {
	user           *model.User
	err            error
	gotProvider    string
	gotProviderUID string
	calls          int
}

var _ auth.IdentityResolver = (*fakeResolver)(nil)

func (f *fakeResolver) ResolveByProvider(
	_ context.Context,
	provider string,
	providerUserID string,
) (*model.User, error) {
	f.calls++
	f.gotProvider = provider
	f.gotProviderUID = providerUserID

	return f.user, f.err
}

func activeRecord() *model.User {
	return &model.User{ID: recordID, Status: model.StatusActive}
}

func verifierFor(t *testing.T, dex *authtest.Provider) auth.TokenVerifier {
	t.Helper()

	cfg := &config.Config{}
	cfg.OIDC.Issuer = dex.URL
	cfg.OIDC.ClientID = authtest.ClientID
	cfg.OIDC.ClientSecret = "secret"
	cfg.OIDC.RedirectURI = "http://hub.maroid.localhost/auth/callback"

	service, err := auth.NewOIDCService(cfg)
	require.NoError(t, err)

	return auth.NewTokenVerifier(service)
}

func serve(
	t *testing.T,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	token string,
	next http.Handler,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugins", nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()

	auth.Middleware(slog.New(slog.DiscardHandler), verifier, resolver)(next).
		ServeHTTP(recorder, request)

	return recorder
}

// refuse fails the test when the middleware lets a request through.
func refuse(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("the handler must not run")
	})
}

func recordActingUser(acting *string) http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		*acting = pluginapi.ActingUserFromContext(r.Context())
	})
}

// EXTID-SC-001: A token whose federated claims name an identity of U resolves to
// U, and the handler runs for that record.
// EXTID-SC-023: The HTTP entry point resolves through the same identity that the
// bot resolves through. See EXTID-DD-012.
func TestATokenOfDexCarriesTheActingUser(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	resolver := &fakeResolver{user: activeRecord()}

	var acting string

	recorder := serve(
		t,
		verifierFor(t, dex),
		resolver,
		dex.Sign(t, auth.ProviderTelegram, account),
		recordActingUser(&acting),
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, recordID, acting)
	require.Equal(t, auth.ProviderTelegram, resolver.gotProvider)
	require.Equal(t, account, resolver.gotProviderUID)
}

// EXTID-SC-003: An identity that names a record which is not active reaches
// nothing, so a block ends a live token within one request. See SEC-004.
func TestABlockedRecordEndsALiveToken(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)

	recorder := serve(
		t,
		verifierFor(t, dex),
		&fakeResolver{err: errs.ErrUserNotFound},
		dex.Sign(t, auth.ProviderTelegram, account),
		refuse(t),
	)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

// EXTID-FR-001: The hub accepts a token that Dex issued and nothing else.
func TestTheMiddlewareRefusesARequestWithNoToken(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)

	recorder := serve(t, verifierFor(t, dex), &fakeResolver{user: activeRecord()}, "", refuse(t))
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

// SEC-002: The verification checks the signature, the issuer, the audience, and
// the expiry. A token that fails any of the four reaches nothing.
func TestTheMiddlewareRefusesATokenThatDexDidNotIssue(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	other := authtest.StartProvider(t)
	verifier := verifierFor(t, dex)

	cases := map[string]func() string{
		"a token of another issuer": func() string {
			return other.Sign(t, auth.ProviderTelegram, account)
		},
		"a token for another audience": func() string {
			claims := dex.Claims(auth.ProviderTelegram, account)
			claims["aud"] = "another-client"

			return dex.SignClaims(t, claims)
		},
		"a token that expired": func() string {
			claims := dex.Claims(auth.ProviderTelegram, account)
			claims["exp"] = time.Now().Add(-time.Hour).Unix()

			return dex.SignClaims(t, claims)
		},
	}

	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			recorder := serve(t, verifier, &fakeResolver{user: activeRecord()}, build(), refuse(t))
			require.Equal(t, http.StatusUnauthorized, recorder.Code)
		})
	}
}

// SEC-003: The hub joins on the federated claims. A token without them names no
// external account, so it reaches nothing.
func TestATokenWithNoFederatedClaimsIsRefused(t *testing.T) {
	t.Parallel()

	dex := authtest.StartProvider(t)
	resolver := &fakeResolver{user: activeRecord()}

	claims := dex.Claims(auth.ProviderTelegram, account)
	delete(claims, "federated_claims")

	recorder := serve(t, verifierFor(t, dex), resolver, dex.SignClaims(t, claims), refuse(t))

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Zero(t, resolver.calls, "the hub asks no question it cannot answer")
}

// EXTID-SC-020: Over 1000 requests that carry a valid token, at most one waits
// for the identity provider. EXTID-NFR-001 gives the limit.
func TestTheVerifierReadsTheKeySetOnce(t *testing.T) {
	t.Parallel()

	const requests = 1000

	dex := authtest.StartProvider(t)
	verifier := verifierFor(t, dex)
	resolver := &fakeResolver{user: activeRecord()}
	token := dex.Sign(t, auth.ProviderTelegram, account)

	var acting string

	for range requests {
		recorder := serve(t, verifier, resolver, token, recordActingUser(&acting))
		require.Equal(t, http.StatusOK, recorder.Code)
	}

	require.Equal(t, requests, resolver.calls, "the record is read on every request")
	require.LessOrEqual(t, dex.KeyRequests(), int64(1), "the key set is read once or never")
}
