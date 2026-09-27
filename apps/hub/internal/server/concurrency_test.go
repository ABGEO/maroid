package server_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/server"
	"github.com/abgeo/maroid/libs/rest"
)

// writeRouter answers one write through the chain, and reports the moment that
// reached the handler. The route stands for every write of the hub and of a
// plugin, because all of them mount under this one chain.
func writeRouter(t *testing.T, reached **time.Time) http.Handler {
	t.Helper()

	cfg := &config.Config{}
	cfg.Server.ExternalURL = externalURL

	router, err := server.NewHTTPRouter(cfg, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	router.Put("/writes", func(_ http.ResponseWriter, r *http.Request) {
		*reached = rest.IfMatchFromContext(r.Context())
	})

	return router
}

// send runs one write with the given validator.
func send(t *testing.T, router http.Handler, validator string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodPut, "/writes", strings.NewReader("{}"),
	)
	if validator != "" {
		request.Header.Set(rest.IfMatchHeader, validator)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

// APIFMT-SC-019: The chain reads the validator once, so every write of the hub
// and of every plugin gets it off the context and parses nothing. A route that
// this chain does not cover loses its guard without a sign, so this pins the
// wiring and not the middleware.
func TestTheChainCarriesTheValidatorToEveryWrite(t *testing.T) {
	t.Parallel()

	var reached *time.Time

	moment := time.Date(2026, time.September, 25, 10, 30, 0, 123456789, time.UTC)
	recorder := send(t, writeRouter(t, &reached), rest.ETag(moment))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, reached, "the chain mounts rest.IfMatch")
	assert.True(t, moment.Equal(*reached))
}

// APIFMT-SC-019: A write that names no validator reaches its handler with no
// moment and lands, because APIFMT-DD-014 keeps the header optional.
func TestTheChainTakesAWriteThatNamesNoValidator(t *testing.T) {
	t.Parallel()

	var reached *time.Time

	recorder := send(t, writeRouter(t, &reached), "")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Nil(t, reached)
}

// APIFMT-SC-019: A validator that this API never answered stops at the chain, so
// no handler of any plugin repeats the check.
func TestTheChainRefusesAValidatorThatMaroidDidNotAnswer(t *testing.T) {
	t.Parallel()

	var reached *time.Time

	recorder := send(t, writeRouter(t, &reached), "*")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Nil(t, reached, "the handler never ran")
}
