package registry_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	pluginGas         = "dev.maroid.tbilisi-energy"
	pluginElectricity = "dev.maroid.telmico"
	sharedJobID       = "transactions_collector"
)

// namedJob is a cron job that names itself and does nothing.
type namedJob struct {
	id string
}

func (j namedJob) Meta() pluginapi.CronJobMeta {
	return pluginapi.CronJobMeta{ID: j.id, Schedule: "0 10 1 * *"}
}

func (j namedJob) Run(context.Context) error { return nil }

// PLG-010: Two plugins that name a job alike both load, because the identifier of a job
// is unique within its plugin. Each entry keeps its plugin.
func TestTwoPluginsNameAJobAlike(t *testing.T) {
	t.Parallel()

	cron := registry.NewCronRegistry()

	require.NoError(t, cron.Register(namedJob{id: sharedJobID}))
	require.NoError(t, cron.RegisterOf(pluginGas, namedJob{id: sharedJobID}))
	require.NoError(t, cron.RegisterOf(pluginElectricity, namedJob{id: sharedJobID}))

	entries := cron.Entries()
	require.Len(t, entries, 3)

	assert.Empty(t, entries[0].PluginID)
	assert.Equal(t, pluginGas, entries[1].PluginID)
	assert.Equal(t, pluginElectricity, entries[2].PluginID)

	for _, entry := range entries {
		assert.Equal(t, sharedJobID, entry.Job.Meta().ID)
	}
}

// JOB-008: A plugin that names two of its jobs alike is refused, and the hub too.
func TestAPluginNamesEachJobOnce(t *testing.T) {
	t.Parallel()

	cron := registry.NewCronRegistry()

	require.NoError(t, cron.RegisterOf(pluginGas, namedJob{id: sharedJobID}))
	require.ErrorIs(
		t,
		cron.RegisterOf(pluginGas, namedJob{id: sharedJobID}),
		errs.ErrCronAlreadyRegistered,
	)

	require.NoError(t, cron.Register(namedJob{id: "sweep"}))
	require.ErrorIs(t, cron.Register(namedJob{id: "sweep"}), errs.ErrCronAlreadyRegistered)

	assert.Len(t, cron.Entries(), 2)
}
