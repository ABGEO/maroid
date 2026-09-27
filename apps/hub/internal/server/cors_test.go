package server_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/server"
)

// deckOrigin stands for the web shell, which runs on its own origin.
const deckOrigin = "https://maroid.localhost"

// corsRouter answers one route through the chain, with CORS as a deployment
// configures it for the deck.
func corsRouter(t *testing.T) http.Handler {
	t.Helper()

	cfg := &config.Config{}
	cfg.Server.ExternalURL = externalURL
	cfg.CORS.Enabled = true
	cfg.CORS.AllowCredentials = true
	cfg.CORS.AllowOrigins = []string{deckOrigin}
	cfg.CORS.AllowMethods = []string{"GET", "POST", "PUT", "DELETE"}
	cfg.CORS.AllowHeaders = []string{"*"}
	cfg.CORS.ExposeHeaders = []string{"ETag", "X-Flow-ID"}

	router, err := server.NewHTTPRouter(cfg, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	router.Get("/reads", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"1790332200123456789"`)
	})

	return router
}

// APIFMT-SC-019: A browser reads the entity tag off the answer. A credentialed
// request takes no wildcard, so the answer names every header that a client
// reads, or `getTagged` answers no tag and every write goes unconditional.
func TestABrowserReadsTheEntityTagAcrossOrigins(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/reads", nil)
	request.Header.Set("Origin", deckOrigin)

	recorder := httptest.NewRecorder()
	corsRouter(t).ServeHTTP(recorder, request)

	// The library canonicalizes each name, and RFC 9110 makes a header name
	// case insensitive, so a browser matches whatever case reaches it.
	exposed := strings.ToLower(recorder.Header().Get("Access-Control-Expose-Headers"))

	assert.Contains(t, exposed, "etag", "a browser reads no header that this list omits")
	assert.Contains(t, exposed, "x-flow-id", "a person reports the flow identifier")
	assert.NotEqual(t, "*", strings.TrimSpace(exposed),
		"the Fetch standard reads a wildcard here as a header of that name")
	assert.Equal(t, "true", recorder.Header().Get("Access-Control-Allow-Credentials"))
}

// APIFMT-SC-019: The preflight of a conditional write allows If-Match, and the
// preflight of a repeated write allows Idempotency-Key.
func TestThePreflightAllowsTheWriteHeaders(t *testing.T) {
	t.Parallel()

	for _, header := range []string{"If-Match", "Idempotency-Key"} {
		t.Run(header, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodOptions, "/reads", nil,
			)
			request.Header.Set("Origin", deckOrigin)
			request.Header.Set("Access-Control-Request-Method", http.MethodPut)
			request.Header.Set("Access-Control-Request-Headers", header)

			recorder := httptest.NewRecorder()
			corsRouter(t).ServeHTTP(recorder, request)

			assert.Contains(t, recorder.Header().Get("Access-Control-Allow-Headers"), header)
		})
	}
}
