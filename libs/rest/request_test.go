package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest"
)

// ERR-006: The hub gives each request an identifier, and it is a UUID version 7.
func TestRequestIDGivesAUUIDVersion7(t *testing.T) {
	t.Parallel()

	var seen string

	handler := rest.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = rest.RequestIDFromContext(r.Context())
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ping", nil),
	)

	parsed, err := uuid.Parse(seen)
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), parsed.Version())
}

// ERR-006: The hub sets the identifier on every response, as the bare UUID.
func TestRequestIDSetsTheHeaderAsTheBareUUID(t *testing.T) {
	t.Parallel()

	handler := rest.RequestID(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ping", nil),
	)

	header := recorder.Header().Get(rest.RequestIDHeader)
	require.NotEmpty(t, header)
	assert.NotContains(t, header, "urn:")

	_, err := uuid.Parse(header)
	require.NoError(t, err)
}

// ERR-006: The hub ignores the header that the request carries, so a caller cannot
// choose the value that every log record holds.
func TestRequestIDIgnoresTheHeaderOfTheRequest(t *testing.T) {
	t.Parallel()

	const chosen = "chosen-by-the-caller"

	var seen string

	handler := rest.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = rest.RequestIDFromContext(r.Context())
	}))

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ping", nil)
	request.Header.Set(rest.RequestIDHeader, chosen)

	handler.ServeHTTP(httptest.NewRecorder(), request)

	assert.NotEqual(t, chosen, seen)
}

// ERR-006: A problem holds the identifier in instance, as a URI.
func TestWriteFillsTheInstanceFromTheRequest(t *testing.T) {
	t.Parallel()

	var body map[string]any

	handler := rest.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest.Write(w, r, rest.NewNotFound())
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugins", nil),
	)

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

	instance, ok := body["instance"].(string)
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(instance, "urn:maroid:request:"))
	assert.Equal(t, recorder.Header().Get(rest.RequestIDHeader),
		strings.TrimPrefix(instance, "urn:maroid:request:"))
}
