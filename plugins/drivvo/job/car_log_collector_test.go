package job_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/drivvo/config"
	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/internal/fakes"
	"github.com/abgeo/maroid/plugins/drivvo/internal/fakeservice"
	"github.com/abgeo/maroid/plugins/drivvo/job"
	"github.com/abgeo/maroid/plugins/drivvo/service"
)

type jobRig struct {
	fake  *fakeservice.Service
	store *fakes.Store
	logs  *bytes.Buffer
	job   *job.CarLogCollector
}

// newJobRig answers the job over the fake service, with a session for workspace W alone.
func newJobRig(t *testing.T) *jobRig {
	t.Helper()

	cfg := new(config.Config)
	require.NoError(t, pluginconfig.DecodeAndValidateConfig(map[string]any{}, cfg))

	fake := fakeservice.New(t)
	cfg.BaseURL = fake.URL()

	clock := func() time.Time { return time.Date(2026, 11, 25, 6, 0, 0, 0, time.UTC) }
	client := service.NewAPIClient(cfg, clock)
	secrets := fakes.NewSecrets().
		WithSession(fakes.WorkspaceW, fakeservice.Token, clock().Add(time.Hour))
	store := fakes.NewStore()
	account := service.NewAccount(client, fakes.Credentials(), secrets, clock)

	logs := new(bytes.Buffer)
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelInfo}))

	return &jobRig{
		fake:  fake,
		store: store,
		logs:  logs,
		job:   job.NewCarLogCollector(cfg, logger, service.NewCollector(account, client, store)),
	}
}

// DRIVVO-SC-012: The job collects a connected workspace, and skips a workspace with no
// connection with a log at info and no call to the service.
func TestTheJobCollectsAConnectedWorkspaceAndSkipsTheOther(t *testing.T) {
	t.Parallel()

	rig := newJobRig(t)

	require.NoError(t, rig.job.Run(fakes.Acting(t.Context(), fakes.WorkspaceW)))

	callsOfW := len(rig.fake.Calls())

	require.NoError(t, rig.job.Run(fakes.Acting(t.Context(), fakes.WorkspaceX)))

	assert.Len(t, rig.store.Snapshots(fakes.WorkspaceW), 1)
	assert.Empty(t, rig.store.Snapshots(fakes.WorkspaceX))
	assert.Len(t, rig.fake.Calls(), callsOfW)
	assert.Contains(t, rig.logs.String(), "level=INFO msg=\"skipping the workspace")
	assert.Contains(t, rig.logs.String(), "workspace_id="+fakes.WorkspaceX)
}

// DRIVVO-SC-013: The job runs for each workspace on the schedule of the configuration.
func TestTheJobRunsForEachWorkspaceEachHour(t *testing.T) {
	t.Parallel()

	rig := newJobRig(t)

	assert.Equal(t, pluginapi.CronJobMeta{
		ID:       "car_log_collector",
		Schedule: "0 * * * *",
		Scope:    pluginapi.CronScopePerWorkspace,
	}, rig.job.Meta())
}

// DRIVVO-SC-017: A failed collection answers an error that names the path, for the
// scheduler to log with the workspace, and the job logs no secret.
func TestAFailedCollectionAnswersItsReason(t *testing.T) {
	t.Parallel()

	rig := newJobRig(t)
	rig.fake.Set(func(state *fakeservice.Service) {
		state.Status["/veiculo/12/despesa/completo"] = http.StatusInternalServerError
	})

	err := rig.job.Run(fakes.Acting(t.Context(), fakes.WorkspaceW))
	require.ErrorIs(t, err, errs.ErrRequestFailed)

	assert.Contains(t, err.Error(), "/veiculo/12/despesa/completo")

	for _, secret := range []string{fakeservice.Token, fakes.Email, "secret"} {
		assert.NotContains(t, err.Error(), secret)
		assert.NotContains(t, rig.logs.String(), secret)
	}
}
