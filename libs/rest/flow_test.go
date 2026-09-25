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

// flowOf runs one request through the middleware and returns the value that the
// handler saw, with the recorder that holds the answer.
func flowOf(t *testing.T, sent string) (string, *httptest.ResponseRecorder) {
	t.Helper()

	var seen string

	handler := rest.FlowID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = rest.FlowIDFromContext(r.Context())
	}))

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugins", nil)
	if sent != "" {
		request.Header.Set(rest.FlowIDHeader, sent)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return seen, recorder
}

// APIFMT-SC-018: A request that carries no flow identifier gets a UUID version 7,
// and the answer carries it bare.
func TestFlowIDGivesAUUIDVersion7WhenTheRequestCarriesNone(t *testing.T) {
	t.Parallel()

	seen, recorder := flowOf(t, "")

	parsed, err := uuid.Parse(seen)
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), parsed.Version())
	assert.Equal(t, seen, recorder.Header().Get(rest.FlowIDHeader))
}

// APIFMT-SC-018: A caller picks the value that every record of its request
// carries. ERR-006 asks the hub to read the header, which Z-233 requires.
func TestFlowIDTakesTheValueOfTheRequest(t *testing.T) {
	t.Parallel()

	const chosen = "trace-7f3a/2b+1_c=4"

	seen, recorder := flowOf(t, chosen)

	assert.Equal(t, chosen, seen)
	assert.Equal(t, chosen, recorder.Header().Get(rest.FlowIDHeader))
}

// APIFMT-SC-018: The middleware bounds what a caller sends, because the value
// reaches the log and the body of a problem.
func TestFlowIDBoundsTheValueOfTheCaller(t *testing.T) {
	t.Parallel()

	t.Run("drops a character outside the set", func(t *testing.T) {
		t.Parallel()

		seen, _ := flowOf(t, "ab<script>cd é\n\tef")
		assert.Equal(t, "abscriptcdef", seen)
	})

	t.Run("cuts to 128 characters", func(t *testing.T) {
		t.Parallel()

		seen, _ := flowOf(t, strings.Repeat("a", 300))
		assert.Len(t, seen, 128)
	})

	t.Run("makes its own when nothing remains", func(t *testing.T) {
		t.Parallel()

		seen, _ := flowOf(t, "<<<>>> é ✓")

		parsed, err := uuid.Parse(seen)
		require.NoError(t, err)
		assert.Equal(t, uuid.Version(7), parsed.Version())
	})
}

// APIFMT-SC-018: A problem holds the flow identifier in instance, as the
// relative reference that Z-176 shapes like a type.
func TestWriteFillsTheInstanceWithTheFlow(t *testing.T) {
	t.Parallel()

	var body map[string]any

	handler := rest.FlowID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	assert.Equal(t, "/flows/"+recorder.Header().Get(rest.FlowIDHeader), instance)
}
