package handler_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hellofresh/health-go/v5"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/openbao/openbao/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/healthcheck"
	"github.com/abgeo/maroid/apps/hub/internal/logger"
	"github.com/abgeo/maroid/apps/hub/internal/server"
	"github.com/abgeo/maroid/libs/rest/flow"
	"github.com/abgeo/maroid/libs/rest/problem"
)

const (
	probeFlowID       = "01a0c611-c3d9-710d-84de-7df920aa9a5f"
	failedCheckRecord = "dependency check failed"
	refusedDatabase   = "pinging the database: dial tcp 10.0.3.7:5432: connect: connection refused"
	hubComponent      = "maroid-hub"
	hubVersion        = "0.1.0"
	database          = "database"
	idp               = "idp"
	timedOut          = "Timeout during health check"
)

// fakeChecker answers the measurements that a test gives it, and counts each
// readiness that the handler asks for.
type fakeChecker struct {
	readiness health.Check
	draining  bool
	measured  atomic.Int32
}

func (f *fakeChecker) Liveness(context.Context) health.Check {
	return health.Check{
		Status:    health.StatusOK,
		Timestamp: time.Now(),
		Component: health.Component{Name: hubComponent, Version: hubVersion},
	}
}

func (f *fakeChecker) Readiness(context.Context) health.Check {
	f.measured.Add(1)

	return f.readiness
}

func (f *fakeChecker) Draining() bool {
	return f.draining
}

// failing returns a checker whose readiness fails each dependency with its text.
func failing(failures map[string]string) *fakeChecker {
	return &fakeChecker{readiness: health.Check{
		Status:    health.StatusUnavailable,
		Timestamp: time.Now(),
		Failures:  failures,
		Component: health.Component{Name: hubComponent, Version: hubVersion},
	}}
}

// probe sends one probe through a router that holds the health handler, and
// returns the answer and every log record that the handler wrote.
func probe(
	t *testing.T,
	checker healthcheck.Checker,
	path string,
) (*httptest.ResponseRecorder, []map[string]any) {
	t.Helper()

	var buffer bytes.Buffer

	log := slog.New(logger.WithFlowID(slog.NewJSONHandler(&buffer, nil)))

	router := chi.NewRouter()
	handler.NewHealth(log, checker).Register(router)

	request := httptest.NewRequestWithContext(
		flow.WithID(t.Context(), probeFlowID), http.MethodGet, path, nil,
	)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	var records []map[string]any

	scanner := bufio.NewScanner(&buffer)
	for scanner.Scan() {
		var record map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &record))

		if record["msg"] == failedCheckRecord {
			records = append(records, record)
		}
	}

	return recorder, records
}

func bodyOf(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body), recorder.Body.String())

	return body
}

// HEALTH-SC-001: A failed dependency does not reach the liveness, so the
// orchestrator never restarts the hub for it.
func TestLivenessSucceedsWhileADependencyFails(t *testing.T) {
	t.Parallel()

	recorder, _ := probe(t, failing(map[string]string{database: refusedDatabase}), "/livez")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "OK", bodyOf(t, recorder)["status"])
}

// HEALTH-SC-003: A success answers the measurement of the library, with the
// moment in UTC.
func TestReadinessAnswersTheMeasurement(t *testing.T) {
	t.Parallel()

	moment := time.Date(2026, time.September, 29, 14, 30, 0, 0, time.FixedZone("+04", 4*60*60))
	checker := &fakeChecker{readiness: health.Check{
		Status:    health.StatusOK,
		Timestamp: moment,
		Component: health.Component{Name: hubComponent, Version: hubVersion},
	}}

	recorder, _ := probe(t, checker, "/readyz")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "application/json")

	body := bodyOf(t, recorder)
	assert.Equal(t, "OK", body["status"])
	assert.Equal(t, "2026-09-29T10:30:00Z", body["timestamp"])
	assert.Equal(t, map[string]any{"name": hubComponent, "version": hubVersion}, body["component"])
}

// HEALTH-SC-004: A failed dependency answers 503 with a problem.
func TestReadinessFailsWhenADependencyFails(t *testing.T) {
	t.Parallel()

	recorder, _ := probe(t, failing(map[string]string{database: refusedDatabase}), "/readyz")

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	assert.Equal(t, problem.MediaType, recorder.Header().Get("Content-Type"))
}

// HEALTH-SC-008: The problem names each failed dependency, sorted.
func TestReadinessNamesEachFailedDependency(t *testing.T) {
	t.Parallel()

	checker := failing(map[string]string{idp: timedOut, database: refusedDatabase})

	recorder, _ := probe(t, checker, "/readyz")

	body := bodyOf(t, recorder)
	assert.Equal(t, problems.TypeNotReady, body["type"])
	assert.InDelta(t, float64(http.StatusServiceUnavailable), body["status"], 0)
	assert.Equal(t, flow.Instance(probeFlowID), body["instance"])
	assert.Equal(t, []any{database, idp}, body["dependencies"])
}

// HEALTH-SC-009: The answer holds no error text of a dependency. The log holds it,
// with the name and the flow identifier of the request.
func TestReadinessLogsTheErrorTextAndAnswersNone(t *testing.T) {
	t.Parallel()

	recorder, records := probe(t, failing(map[string]string{database: refusedDatabase}), "/readyz")

	assert.NotContains(t, recorder.Body.String(), "10.0.3.7")

	require.Len(t, records, 1)
	assert.Equal(t, "WARN", records[0]["level"])
	assert.Equal(t, database, records[0]["dependency"])
	assert.Equal(t, refusedDatabase, records[0]["error"])
	assert.Equal(t, probeFlowID, records[0]["flow_id"])
}

// HEALTH-SC-010: The chain mounts no access check, so an orchestrator with no
// credential reads both probes.
func TestTheProbesTakeNoCredential(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Server.ExternalURL = "https://hub.example.com"

	log := slog.New(slog.DiscardHandler)

	router, err := server.NewHTTPRouter(cfg, log)
	require.NoError(t, err)

	handler.NewHealth(log, failing(map[string]string{database: refusedDatabase})).Register(router)

	for _, path := range []string{"/livez", "/readyz"} {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		assert.NotEqual(t, http.StatusUnauthorized, recorder.Code, path)
	}
}

// HEALTH-SC-011: A drain answers 503 with the base problem and measures nothing.
func TestReadinessFailsFromTheStartOfAShutdown(t *testing.T) {
	t.Parallel()

	checker := &fakeChecker{
		readiness: health.Check{Status: health.StatusOK, Timestamp: time.Now()},
		draining:  true,
	}

	recorder, _ := probe(t, checker, "/readyz")

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)

	body := bodyOf(t, recorder)
	assert.Equal(t, problems.TypeNotReady, body["type"])
	assert.Equal(t, "The hub is shutting down.", body["detail"])
	assert.NotContains(t, body, "dependencies")
	assert.Zero(t, checker.measured.Load())
}

// HEALTH-SC-014: The liveness answers in less than 100 milliseconds while no
// dependency answers.
func TestLivenessAnswersInTime(t *testing.T) {
	t.Parallel()

	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()

	cfg := &config.Config{}
	cfg.OIDC.Issuer = closed.URL

	db, err := sqlx.Open("pgx", "postgres://maroid@127.0.0.1:1/maroid?connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	storeConfig := api.DefaultConfig()
	storeConfig.Address = closed.URL

	store, err := api.NewClient(storeConfig)
	require.NoError(t, err)

	service, err := healthcheck.New(cfg, db, store, http.DefaultClient)
	require.NoError(t, err)

	for range 100 {
		started := time.Now()
		recorder, _ := probe(t, service, "/livez")

		require.Equal(t, http.StatusOK, recorder.Code)
		require.Less(t, time.Since(started), 100*time.Millisecond)
	}
}

// HEALTH-SC-018: Each failed check writes one record, a timeout among them, and a
// drain writes none.
func TestEachFailedCheckWritesOneRecord(t *testing.T) {
	t.Parallel()

	checker := failing(map[string]string{
		idp:      timedOut,
		database: "pinging the database: connection refused",
	})

	_, records := probe(t, checker, "/readyz")

	errorsByDependency := map[any]any{}

	for _, record := range records {
		assert.Equal(t, "WARN", record["level"])
		errorsByDependency[record["dependency"]] = record["error"]
	}

	assert.Equal(t, map[any]any{
		idp:      timedOut,
		database: "pinging the database: connection refused",
	}, errorsByDependency)

	checker.draining = true
	_, drained := probe(t, checker, "/readyz")

	assert.Empty(t, drained)
}
