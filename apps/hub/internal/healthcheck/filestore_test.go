package healthcheck_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/hellofresh/health-go/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/filestore"
	"github.com/abgeo/maroid/apps/hub/internal/teststore"
)

const fileStoreCheck = "file-store"

var errRefused = errors.New("connection refused")

// FILES-SC-031: A file store that does not answer fails its check, and one that
// answers passes it.
func TestTheFileStoreCountsAsADependency(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		store  pinger
		failed bool
	}{
		"answers":         {store: pinger{}},
		"does not answer": {store: pinger{err: errRefused}, failed: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			idp := identityProvider(t, http.StatusOK)
			store := secretStore(t, unsealed, http.StatusOK)

			failures := newServiceWith(
				t,
				idp,
				store,
				"",
				testCase.store,
			).Readiness(t.Context()).
				Failures

			if testCase.failed {
				assert.Contains(t, failures, fileStoreCheck)
			} else {
				assert.NotContains(t, failures, fileStoreCheck)
			}
		})
	}
}

// FILES-SC-031: The readiness fails the check of the file store once RustFS stops.
func TestTheReadinessFailsWhenRustFSStops(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	server := teststore.Start(t)
	idp := identityProvider(t, http.StatusOK)
	store := secretStore(t, unsealed, http.StatusOK)
	service := newServiceWith(t, idp, store, "", filestore.New(server.Config()))

	require.NotContains(t, service.Readiness(t.Context()).Failures, fileStoreCheck)

	server.Stop(t)

	check := service.Readiness(t.Context())
	assert.Equal(t, health.StatusUnavailable, check.Status)
	assert.Contains(t, check.Failures, fileStoreCheck)
}
