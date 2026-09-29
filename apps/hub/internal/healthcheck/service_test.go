package healthcheck_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hellofresh/health-go/v5"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/openbao/openbao/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/healthcheck"
)

const (
	discoveryPath = "/.well-known/openid-configuration"
	healthPath    = "/v1/sys/health"
	unsealed      = `{"initialized": true, "sealed": false}`
	hangTime      = 10 * time.Second
)

// newService builds a Service whose database refuses every connection, and whose
// IdP and secret store are the given test servers.
func newService(t *testing.T, idp, secretStore *httptest.Server) *healthcheck.Service {
	t.Helper()

	cfg := &config.Config{}
	cfg.OIDC.Issuer = idp.URL

	db, err := sqlx.Open("pgx", "postgres://maroid@127.0.0.1:1/maroid?connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	storeConfig := api.DefaultConfig()
	storeConfig.Address = secretStore.URL

	store, err := api.NewClient(storeConfig)
	require.NoError(t, err)

	service, err := healthcheck.New(cfg, db, store, idp.Client())
	require.NoError(t, err)

	return service
}

// answer returns a test server that answers the path with the status and the body.
func answer(t *testing.T, path string, status int, body string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

// hang returns a test server that answers after hangTime, and the moment at which
// each of its requests ended.
func hang(t *testing.T) (*httptest.Server, chan time.Time) {
	t.Helper()

	ended := make(chan time.Time, 1)

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(hangTime):
		}

		ended <- time.Now()
	}))
	t.Cleanup(server.Close)

	return server, ended
}

// HEALTH-SC-002: The liveness reads no dependency, so a failed dependency never
// restarts the process.
func TestLivenessReadsNoDependency(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	counting := func(next http.Handler) *httptest.Server {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			next.ServeHTTP(w, r)
		}))
		t.Cleanup(server.Close)

		return server
	}

	idp := counting(http.NotFoundHandler())
	secretStore := counting(http.NotFoundHandler())
	service := newService(t, idp, secretStore)

	for range 10 {
		assert.Equal(t, health.StatusOK, service.Liveness(t.Context()).Status)
	}

	assert.Zero(t, calls.Load())
}

// HEALTH-SC-005: A dependency that hangs fails its check at the limit, and the
// check cancels its call, so the answer arrives in less than 3 seconds.
func TestReadinessAnswersInTimeWhenDependenciesHang(t *testing.T) {
	t.Parallel()

	idp, idpEnded := hang(t)
	secretStore, storeEnded := hang(t)
	service := newService(t, idp, secretStore)

	started := time.Now()
	check := service.Readiness(t.Context())

	assert.Less(t, time.Since(started), 3*time.Second)
	assert.Equal(t, health.StatusUnavailable, check.Status)
	assert.Contains(t, check.Failures, "idp")
	assert.Contains(t, check.Failures, "secret-store")

	for name, ended := range map[string]chan time.Time{"idp": idpEnded, "secret-store": storeEnded} {
		select {
		case moment := <-ended:
			assert.Less(t, moment.Sub(started), 3*time.Second, name)
		case <-time.After(3 * time.Second):
			assert.Fail(t, "the request stayed open", name)
		}
	}
}

// HEALTH-SC-006: The IdP counts as up on 200 only, because a proxy in front of it
// answers 404 or 503 and a sign in fails in each case.
func TestTheIdentityProviderNeedsTheDiscoveryDocument(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		status int
		failed bool
	}{
		"200": {http.StatusOK, false},
		"404": {http.StatusNotFound, true},
		"503": {http.StatusServiceUnavailable, true},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			idp := answer(t, discoveryPath, testCase.status, `{}`)
			secretStore := answer(t, healthPath, http.StatusOK, unsealed)

			failures := newService(t, idp, secretStore).Readiness(t.Context()).Failures

			if testCase.failed {
				assert.Contains(t, failures, "idp")
			} else {
				assert.NotContains(t, failures, "idp")
			}
		})
	}
}

// HEALTH-SC-007: A sealed or uninitialized secret store answers the network and
// protects no secret, so it counts as failed.
func TestTheSecretStoreMustBeInitializedAndUnsealed(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		body   string
		failed bool
	}{
		"sealed":        {`{"initialized": true, "sealed": true}`, true},
		"uninitialized": {`{"initialized": false, "sealed": true}`, true},
		"unsealed":      {unsealed, false},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			idp := answer(t, discoveryPath, http.StatusOK, `{}`)
			secretStore := answer(t, healthPath, http.StatusOK, testCase.body)

			failures := newService(t, idp, secretStore).Readiness(t.Context()).Failures

			if testCase.failed {
				assert.Contains(t, failures, "secret-store")
			} else {
				assert.NotContains(t, failures, "secret-store")
			}
		})
	}
}
