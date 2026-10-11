package main

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/jasmine/config"
)

// GARDEN-SC-028: A configuration with no channel schedules no reminder. A
// configuration with a channel schedules one reminder for each workspace.
func TestTheReminderNeedsAChannel(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	silent := &JasminePlugin{config: &config.Config{}, logger: logger}

	jobs, err := silent.CronJobs()
	require.NoError(t, err)
	assert.Empty(t, jobs)

	configured := &JasminePlugin{
		config: &config.Config{
			Notification: config.Notification{CareReminderChannel: "garden"},
		},
		logger: logger,
	}

	jobs, err = configured.CronJobs()
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "care_reminder", jobs[0].Meta().ID)
	assert.Equal(t, pluginapi.CronScopePerWorkspace, jobs[0].Meta().Scope)
}
