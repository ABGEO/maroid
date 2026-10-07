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
	lookupPath    = "/v1/auth/token/lookup-self"
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

// identityProvider returns a test server that answers the discovery document with
// the status.
func identityProvider(t *testing.T, status int) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != discoveryPath {
			http.NotFound(w, r)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	return server
}

// secretStore returns a test server that answers the health of the store with the
// body, and the lookup of the token of the hub with the status.
func secretStore(t *testing.T, healthBody string, lookupStatus int) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case healthPath:
			_, _ = w.Write([]byte(healthBody))
		case lookupPath:
			w.WriteHeader(lookupStatus)
			_, _ = w.Write([]byte(`{"data": {}}`))
		default:
			http.NotFound(w, r)
		}
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

			idp := identityProvider(t, testCase.status)
			store := secretStore(t, unsealed, http.StatusOK)

			failures := newService(t, idp, store).Readiness(t.Context()).Failures

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

			idp := identityProvider(t, http.StatusOK)
			store := secretStore(t, testCase.body, http.StatusOK)

			failures := newService(t, idp, store).Readiness(t.Context()).Failures

			if testCase.failed {
				assert.Contains(t, failures, "secret-store")
			} else {
				assert.NotContains(t, failures, "secret-store")
			}
		})
	}
}

// SSACCESS-SC-007: A store that refuses the token of the hub protects no secret for
// the hub, so it counts as failed.
func TestTheSecretStoreFailsWhenItRefusesTheToken(t *testing.T) {
	t.Parallel()

	idp := identityProvider(t, http.StatusOK)
	store := secretStore(t, unsealed, http.StatusForbidden)

	check := newService(t, idp, store).Readiness(t.Context())

	assert.Equal(t, health.StatusUnavailable, check.Status)
	assert.Contains(t, check.Failures, "secret-store")
}

// SSACCESS-SC-008: A store that accepts the token of the hub counts as up.
func TestTheSecretStoreIsUpWhenItAcceptsTheToken(t *testing.T) {
	t.Parallel()

	idp := identityProvider(t, http.StatusOK)
	store := secretStore(t, unsealed, http.StatusOK)

	check := newService(t, idp, store).Readiness(t.Context())

	assert.NotContains(t, check.Failures, "secret-store")
}
