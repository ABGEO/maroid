package server_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/logger"
	"github.com/abgeo/maroid/apps/hub/internal/server"
	"github.com/abgeo/maroid/libs/rest"
)

func routerUnderTest(t *testing.T) http.Handler {
	t.Helper()

	router, err := server.NewHTTPRouter(&config.Config{}, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	router.Get("/ping", func(http.ResponseWriter, *http.Request) {})
	router.Get("/boom", func(http.ResponseWriter, *http.Request) {
		panic("the handler gave up")
	})

	return router
}

func answerOf(t *testing.T, method, path string) (*httptest.ResponseRecorder, rest.Problem) {
	t.Helper()

	recorder := httptest.NewRecorder()
	routerUnderTest(t).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), method, path, nil),
	)

	var body rest.Problem

	require.Equal(t, rest.ProblemMediaType, recorder.Header().Get("Content-Type"))
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body), recorder.Body.String())

	return recorder, body
}

// ERR-001: The NotFound handler of the router answers with a problem.
func TestAnUnknownRouteAnswersWithAProblem(t *testing.T) {
	t.Parallel()

	recorder, body := answerOf(t, http.MethodGet, "/nowhere")

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, rest.TypeNotFound, body.Type)
	assert.Equal(t, http.StatusNotFound, body.Status)
}

// ERR-001: The MethodNotAllowed handler of the router answers with a problem.
// RFC 9110 asks that answer for an Allow header that names the methods the route
// does hold.
func TestAMethodThatTheRouteRefusesAnswersWithAProblem(t *testing.T) {
	t.Parallel()

	recorder, body := answerOf(t, http.MethodPost, "/ping")

	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
	assert.Equal(t, rest.TypeMethodNotAllowed, body.Type)
	assert.Equal(t, "GET", recorder.Header().Get("Allow"))
}

// ERR-001: A panic answers with a problem, and ERR-005 keeps the cause in the log.
func TestAPanicAnswersWithAProblemAndLeaksNothing(t *testing.T) {
	t.Parallel()

	recorder, body := answerOf(t, http.MethodGet, "/boom")

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, rest.TypeInternal, body.Type)
	assert.Empty(t, body.Detail)
	assert.NotContains(t, recorder.Body.String(), "the handler gave up")
}

// ERR-006: Every response carries the identifier, and a problem holds it as a URI.
func TestEveryAnswerCarriesTheRequestIdentifier(t *testing.T) {
	t.Parallel()

	recorder, body := answerOf(t, http.MethodGet, "/nowhere")

	identifier := recorder.Header().Get(rest.FlowIDHeader)
	require.NotEmpty(t, identifier)
	assert.Equal(t, rest.Instance(identifier), body.Instance)
}

// LOG-002 and LOG-009: The access record is JSON on stdout, and it carries the
// identifier of the request that a problem also names.
func TestTheAccessRecordCarriesTheRequestIdentifier(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	log := slog.New(logger.WithRequest(slog.NewJSONHandler(buffer, nil)))

	router, err := server.NewHTTPRouter(&config.Config{}, log)
	require.NoError(t, err)

	router.Get("/ping", func(http.ResponseWriter, *http.Request) {})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ping", nil),
	)

	var record map[string]any

	require.NoError(t, json.Unmarshal(buffer.Bytes(), &record), buffer.String())
	assert.Equal(t, recorder.Header().Get(rest.FlowIDHeader), record["flow_id"])
	assert.InDelta(t, float64(http.StatusOK), record["http.response.status_code"], 0)
	assert.Equal(t, http.MethodGet, record["http.request.method"])
	assert.Equal(t, "/ping", record["url.path"])

	// httplog builds this message, and LOG-007 names it as its one exception.
	// The assertion fails if the access log stops going through the library.
	assert.Equal(t, "GET /ping => HTTP 200 (0ms)", msgWithoutDuration(record))
}

// msgWithoutDuration reads the message of the access record with a fixed
// duration, because the real one changes between two runs.
func msgWithoutDuration(record map[string]any) string {
	msg, _ := record["msg"].(string)
	if open := strings.LastIndex(msg, " ("); open != -1 {
		return msg[:open] + " (0ms)"
	}

	return msg
}
