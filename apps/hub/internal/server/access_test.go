package server_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/healthcheck"
	"github.com/abgeo/maroid/apps/hub/internal/server"
)

// ordinaryPath stands for every route that is no probe.
const ordinaryPath = "/other"

// accessPaths returns the path of each access record in the log.
func accessPaths(t *testing.T, buffer *bytes.Buffer) []string {
	t.Helper()

	var paths []string

	scanner := bufio.NewScanner(buffer)
	for scanner.Scan() {
		var record map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &record))

		if record["middleware"] != "access" {
			continue
		}

		path, ok := record["url.path"].(string)
		require.True(t, ok, "an access record carries url.path")

		paths = append(paths, path)
	}

	return paths
}

// HEALTH-SC-013: A probe that succeeds writes no access record. A probe that fails
// and every other request still write one.
func TestTheAccessLogSkipsAProbeThatSucceeds(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer

	cfg := &config.Config{}
	cfg.Server.ExternalURL = externalURL

	router, err := server.NewHTTPRouter(cfg, slog.New(slog.NewJSONHandler(&buffer, nil)))
	require.NoError(t, err)

	answer := func(status int) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }
	}

	router.Get(healthcheck.LivenessPath, answer(http.StatusOK))
	router.Get(healthcheck.ReadinessPath, answer(http.StatusServiceUnavailable))
	router.Get(ordinaryPath, answer(http.StatusOK))

	for _, path := range []string{healthcheck.LivenessPath, healthcheck.ReadinessPath, ordinaryPath} {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		router.ServeHTTP(httptest.NewRecorder(), request)
	}

	assert.ElementsMatch(
		t,
		[]string{healthcheck.ReadinessPath, ordinaryPath},
		accessPaths(t, &buffer),
	)
}
