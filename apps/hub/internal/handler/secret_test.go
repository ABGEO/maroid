package handler_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// IDPROV-SC-023: Every route that takes a password or a client secret answers once,
// with success and with failure. No body and no log line holds the password, its hash,
// or the client secret.
func TestNoAnswerAndNoLogHoldsASecret(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	zura := fixture.administrator(t)

	var lines bytes.Buffer

	fixture.router = loggedAdministration(t, fixture, &lines)

	const (
		clientSecret = "client-secret-value"
		password     = "a password of the test"
		newSecret    = "another-secret-value"
		newPassword  = "another password value"
	)

	body := cloudBody()
	body["client_secret"] = clientSecret

	nina := "/users/" + fixture.nino.id + identitiesRoute

	var answers []string

	answer := func(status int, method string, path string, payload map[string]any) {
		response := fixture.call(t, zura, method, path, payload, "")
		require.Equal(t, status, response.Code, response.Body.String())

		answers = append(answers, response.Body.String())
	}

	answer(http.StatusCreated, http.MethodPost, "/providers", body)
	answer(http.StatusOK, http.MethodPatch, "/providers/abgeo-cloud",
		map[string]any{"client_secret": newSecret})
	answer(http.StatusCreated, http.MethodPost, "/providers", map[string]any{memberPreset: "local"})
	answer(http.StatusCreated, http.MethodPost, nina, localAccountBody(ninoEmail, password))
	answer(http.StatusNoContent, http.MethodPatch, nina+"/local",
		map[string]any{memberPassword: newPassword})

	fixture.idp.Fail(errs.ErrIDPUnavailable)

	answer(http.StatusServiceUnavailable, http.MethodPatch, nina+"/local",
		map[string]any{memberPassword: newPassword})
	answer(http.StatusServiceUnavailable, http.MethodPost,
		"/users/"+fixture.ana.id+identitiesRoute, localAccountBody("ana@home.example", password))

	fixture.idp.Fail(nil)
	hash := string(fixture.idp.Hash(ninoEmail))
	require.NotEmpty(t, hash)
	require.NotEmpty(t, lines.String(), "the failures wrote a log line")

	requireNoSecret(t, append(answers, lines.String()),
		clientSecret, newSecret, password, newPassword, hash)
}

// loggedAdministration answers a router of the routes of an administrator whose log
// lands in the buffer.
func loggedAdministration(
	t *testing.T,
	fixture *workspaceFixture,
	lines *bytes.Buffer,
) *chi.Mux {
	t.Helper()

	router := baseRouter()
	registerAdministration(router, administration{
		logger: slog.New(
			slog.NewJSONHandler(lines, &slog.HandlerOptions{Level: slog.LevelDebug}),
		),
		verifier: workspaceVerifier(t, fixture.provider),
		resolver: auth.NewResolver(fixture.database),
		users:    usersOf(t, fixture.database, fixture.authSvc),
		idp:      fixture.idp,
		database: fixture.database,
	})

	return router
}

func requireNoSecret(t *testing.T, texts []string, secrets ...string) {
	t.Helper()

	for _, secret := range secrets {
		for i, text := range texts {
			assert.NotContains(t, text, secret, "text %d", i)
		}
	}
}
