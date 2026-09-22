package middleware_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/middleware"
)

const telegramNetwork = "149.154.160.0/20"

// answerOf sends one request through the client address middleware that the
// trusted proxies imply, then through the allowlist.
func answerOf(t *testing.T, trustedProxies []string, peer, forwarded string) int {
	t.Helper()

	allowlist, err := middleware.AllowedNetworks(
		slog.New(slog.DiscardHandler),
		[]string{telegramNetwork},
	)
	require.NoError(t, err)

	reached := false
	handler := allowlist(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached = true
	}))

	if len(trustedProxies) == 0 {
		handler = chimiddleware.ClientIPFromRemoteAddr(handler)
	} else {
		handler = chimiddleware.ClientIPFromXFF(trustedProxies...)(handler)
	}

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/telegram/webhook",
		nil,
	)
	request.RemoteAddr = peer

	if forwarded != "" {
		request.Header.Set("X-Forwarded-For", forwarded)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assert.Equal(t, recorder.Code == http.StatusOK, reached)

	return recorder.Code
}

// SEC-009: With no proxy, a caller cannot forge the address that the allowlist
// reads. The header names a Telegram address and the peer does not.
func TestAForgedHeaderReachesNothing(t *testing.T) {
	t.Parallel()

	assert.Equal(t, http.StatusForbidden, answerOf(t, nil, "203.0.113.9:4567", "149.154.167.220"))
}

// SEC-009: With no proxy, the peer of the connection decides.
func TestThePeerOfTheConnectionDecides(t *testing.T) {
	t.Parallel()

	assert.Equal(t, http.StatusOK, answerOf(t, nil, "149.154.167.220:443", ""))
}

// SEC-009: A trusted proxy names the caller, and the allowlist reads it.
func TestATrustedProxyNamesTheCaller(t *testing.T) {
	t.Parallel()

	code := answerOf(t, []string{"10.0.0.0/8"}, "10.1.2.3:80", "149.154.167.220")

	assert.Equal(t, http.StatusOK, code)
}

// SEC-009: Behind a trusted proxy, a caller that prepends its own address
// reaches nothing. The walk runs right to left past the trusted hop.
func TestAPrependedAddressReachesNothing(t *testing.T) {
	t.Parallel()

	code := answerOf(t, []string{"10.0.0.0/8"}, "10.1.2.3:80", "149.154.167.220, 203.0.113.9")

	assert.Equal(t, http.StatusForbidden, code)
}

// SEC-009: The guard fails closed when no middleware resolved an address.
func TestNoClientAddressIsRefused(t *testing.T) {
	t.Parallel()

	allowlist, err := middleware.AllowedNetworks(
		slog.New(slog.DiscardHandler),
		[]string{telegramNetwork},
	)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	allowlist(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("the handler ran without a client address")
	})).ServeHTTP(recorder, httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/telegram/webhook", nil,
	))

	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

// A CIDR that does not parse stops the hub at the start.
func TestABadNetworkIsRefused(t *testing.T) {
	t.Parallel()

	_, err := middleware.AllowedNetworks(slog.New(slog.DiscardHandler), []string{"not-a-cidr"})

	require.Error(t, err)
}
