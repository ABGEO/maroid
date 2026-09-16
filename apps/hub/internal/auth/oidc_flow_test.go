package auth_test

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	flowTTL      = 10 * time.Minute
	flowRedirect = "http://maroid.localhost"
)

// fakeIssuer serves the discovery document that the OIDC provider reads at the
// start. It stands in for Dex, which no unit test may reach.
func fakeIssuer(t *testing.T) string {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	t.Cleanup(server.Close)

	document := fmt.Sprintf(`{
		"issuer": %[1]q,
		"authorization_endpoint": "%[1]s/auth",
		"token_endpoint": "%[1]s/token",
		"jwks_uri": "%[1]s/keys",
		"id_token_signing_alg_values_supported": ["RS256"]
	}`, server.URL)

	discovery := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, document)
	}

	mux.HandleFunc("/.well-known/openid-configuration", discovery)

	return server.URL
}

func flowUnderTest(t *testing.T) (*auth.OIDCFlow, repository.AuthFlowRepository) {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	issuer := fakeIssuer(t)
	cfg := &config.Config{}
	cfg.OIDC.Issuer = issuer
	cfg.OIDC.ClientID = "hub"
	cfg.OIDC.ClientSecret = "secret"
	cfg.OIDC.RedirectURI = "http://hub.maroid.localhost/auth/callback"
	cfg.Auth.FlowTTL = flowTTL

	service, err := auth.NewOIDCService(cfg)
	require.NoError(t, err)

	flowRepo := repository.NewAuthFlow(instance.DB)

	return auth.NewOIDCFlow(service, flowRepo, cfg.Auth.FlowTTL), flowRepo
}

// EXTID-FR-006: The row carries the state, the nonce, and the verifier. Nothing
// travels in a cookie, so the browser cannot rewrite any of them.
// EXTID-DD-001: The caller of Initiate holds no secret.
func TestInitiateWritesTheFlowAndKeepsTheSecrets(t *testing.T) {
	t.Parallel()

	oidcFlow, flowRepo := flowUnderTest(t)
	ctx := t.Context()

	authURL, binding, err := oidcFlow.Initiate(ctx, model.AuthFlow{
		Intent:   model.IntentSignIn,
		Redirect: flowRedirect,
	})
	require.NoError(t, err)
	require.NotEmpty(t, binding, "the browser gets a binding")

	parsed, err := url.Parse(authURL)
	require.NoError(t, err)

	state := parsed.Query().Get("state")
	require.NotEmpty(t, state, "the address carries the state")
	require.NotContains(t, authURL, binding, "the binding never travels in the address")
	require.NotEmpty(t, parsed.Query().Get("nonce"))
	require.NotEmpty(t, parsed.Query().Get("code_challenge"))
	require.NotContains(t, authURL, "code_verifier", "the verifier never leaves the hub")

	stored, err := flowRepo.ConsumeByState(ctx, state)
	require.NoError(t, err)
	require.Equal(t, model.IntentSignIn, stored.Intent)
	require.Equal(t, flowRedirect, stored.Redirect)
	require.NotEmpty(t, stored.Nonce)
	require.NotEmpty(t, stored.Verifier)
	require.True(t, stored.ExpiresAt.After(time.Now()), "the row expires in the future")
	require.Nil(t, stored.UserID, "a sign in names no user record")

	expected := sha256.Sum256([]byte(binding))
	require.Equal(t, expected[:], stored.BindingHash, "the row holds the digest alone")
}

// EXTID-FR-006: One state spends one time, so a replay of the callback reaches
// nothing.
func TestConsumeSpendsTheFlowOneTime(t *testing.T) {
	t.Parallel()

	oidcFlow, _ := flowUnderTest(t)
	ctx := t.Context()

	authURL, binding, err := oidcFlow.Initiate(ctx, model.AuthFlow{
		Intent:   model.IntentSignIn,
		Redirect: flowRedirect,
	})
	require.NoError(t, err)

	parsed, err := url.Parse(authURL)
	require.NoError(t, err)

	state := parsed.Query().Get("state")

	first, err := oidcFlow.Consume(ctx, state, binding)
	require.NoError(t, err)
	require.Equal(t, flowRedirect, first.Redirect)

	_, err = oidcFlow.Consume(ctx, state, binding)
	require.ErrorIs(t, err, errs.ErrAuthFlowNotFound)

	_, err = oidcFlow.Consume(ctx, "a-state-that-nobody-wrote", binding)
	require.ErrorIs(t, err, errs.ErrAuthFlowNotFound)
}

// The hub refuses a flow that outlived auth.flow_ttl, and it reports that reason
// apart from a state that names no row.
func TestConsumeRefusesAnExpiredFlow(t *testing.T) {
	t.Parallel()

	oidcFlow, flowRepo := flowUnderTest(t)
	ctx := t.Context()

	digest := sha256.Sum256([]byte("the-binding"))

	_, err := flowRepo.Create(ctx, model.AuthFlow{
		State:       "state-stale",
		Intent:      model.IntentSignIn,
		BindingHash: digest[:],
		Nonce:       "nonce",
		Verifier:    "verifier",
		Redirect:    flowRedirect,
		ExpiresAt:   time.Now().Add(-time.Minute),
	})
	require.NoError(t, err)

	flow, err := oidcFlow.Consume(ctx, "state-stale", "the-binding")
	require.ErrorIs(t, err, errs.ErrAuthFlowExpired)
	require.NotNil(t, flow, "the caller still needs the target to report the failure")
	require.Equal(t, flowRedirect, flow.Redirect)
}

// A state alone finishes nothing. Without the binding a person who holds a valid
// state completes the flow in another browser, and the holder of that browser is
// then signed in as somebody else. This is login CSRF.
//
// Each case starts its own flow, because a refused callback still spends the row.
func TestConsumeRefusesACallbackFromAnotherBrowser(t *testing.T) {
	t.Parallel()

	oidcFlow, _ := flowUnderTest(t)
	ctx := t.Context()

	start := func(t *testing.T) (string, string) {
		t.Helper()

		authURL, binding, err := oidcFlow.Initiate(ctx, model.AuthFlow{
			Intent:   model.IntentSignIn,
			Redirect: flowRedirect,
		})
		require.NoError(t, err)

		parsed, err := url.Parse(authURL)
		require.NoError(t, err)

		return parsed.Query().Get("state"), binding
	}

	t.Run("a browser that holds no binding is refused", func(t *testing.T) {
		t.Parallel()

		state, _ := start(t)

		flow, err := oidcFlow.Consume(ctx, state, "")
		require.ErrorIs(t, err, errs.ErrAuthFlowBindingMismatch)
		require.Nil(t, flow, "a forged callback gets no target to redirect to")
	})

	t.Run("a browser that holds another binding is refused", func(t *testing.T) {
		t.Parallel()

		state, _ := start(t)

		_, err := oidcFlow.Consume(ctx, state, "a-binding-of-another-browser")
		require.ErrorIs(t, err, errs.ErrAuthFlowBindingMismatch)
	})

	t.Run("the browser that started the flow finishes it", func(t *testing.T) {
		t.Parallel()

		state, binding := start(t)

		flow, err := oidcFlow.Consume(ctx, state, binding)
		require.NoError(t, err)
		require.Equal(t, flowRedirect, flow.Redirect)
	})
}
