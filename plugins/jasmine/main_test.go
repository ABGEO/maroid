package main

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/jasmine/config"
)

// GARDEN-SC-027: The plugin schedules one reminder that runs for each workspace.
func TestTheReminderRunsForEachWorkspace(t *testing.T) {
	t.Parallel()

	plugin := &JasminePlugin{config: &config.Config{}, logger: slog.New(slog.DiscardHandler)}

	jobs, err := plugin.CronJobs()
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "care_reminder", jobs[0].Meta().ID)
	assert.Equal(t, pluginapi.CronScopePerWorkspace, jobs[0].Meta().Scope)
}

// GARDEN-SC-034: The plugin declares the reminder as one notification topic.
func TestThePluginDeclaresTheReminderTopic(t *testing.T) {
	t.Parallel()

	topics, err := (&JasminePlugin{}).NotificationTopics()
	require.NoError(t, err)
	require.Len(t, topics, 1)
	assert.Equal(t, "care_reminder", topics[0].ID)
	assert.Equal(t, "Care reminder", topics[0].Name)
}

// GARDEN-SC-037: The plugin declares the command care, which needs the permission to
// read the plants.
func TestThePluginDeclaresTheCareCommand(t *testing.T) {
	t.Parallel()

	commands, err := (&JasminePlugin{}).TelegramCommands()
	require.NoError(t, err)
	require.Len(t, commands, 1)
	assert.Equal(t, "care", commands[0].Meta().Command)
	assert.Equal(t, "Plants that need care", commands[0].Meta().Description)
	assert.Equal(t, "plants.read", commands[0].Meta().Permission)
}
