package main

import (
	"io/fs"
	"log/slog"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/magti/config"
	"github.com/abgeo/maroid/plugins/magti/permission"
)

// MAGTIACC-SC-001: The settings hold the username as free text and the password as a
// secret, both of the workspace and both required, and no other field.
func TestTheSettingsHoldTheCredentialsAlone(t *testing.T) {
	t.Parallel()

	model, err := (&MagtiPlugin{}).SettingsModel()
	require.NoError(t, err)

	settings := reflect.TypeOf(model)
	require.Equal(t, reflect.TypeFor[config.WorkspaceSettings](), settings)
	require.Equal(t, 2, settings.NumField())

	username, found := settings.FieldByName("Username")
	require.True(t, found)
	assert.Equal(t, "username", username.Tag.Get("json"))
	assert.Equal(t, "title=Phone number or username,required", username.Tag.Get("jsonschema"))
	assert.Equal(t, "x-maroid-scope=workspace", username.Tag.Get("jsonschema_extras"))

	password, found := settings.FieldByName("Password")
	require.True(t, found)
	assert.Equal(t, "password", password.Tag.Get("json"))
	assert.Equal(
		t,
		"title=Password,format=password,writeOnly=true,required",
		password.Tag.Get("jsonschema"),
	)
	assert.Equal(t, "x-maroid-scope=workspace", password.Tag.Get("jsonschema_extras"))
}

// MAGTIACC-SC-027: account.read opens every read and the widget to a viewer, and
// account.connect opens every write to an editor.
func TestThePermissionsNameTheirLowestRoleAndTheirEntries(t *testing.T) {
	t.Parallel()

	plugin := &MagtiPlugin{
		config: &config.Config{BaseURL: "https://www.magticom.ge"},
		logger: slog.New(slog.DiscardHandler),
	}

	permissions, err := plugin.Permissions()
	require.NoError(t, err)

	lowest := map[string]pluginapi.Role{}
	for _, declared := range permissions {
		lowest[declared.Name] = declared.Lowest
	}

	require.Equal(t, map[string]pluginapi.Role{
		permission.AccountRead:    pluginapi.RoleViewer,
		permission.AccountConnect: pluginapi.RoleEditor,
	}, lowest)

	routes, err := plugin.Routes()
	require.NoError(t, err)

	guards := map[string]string{}
	for _, route := range routes {
		guards[route.Method+" "+route.Pattern] = route.Permission
	}

	require.Equal(t, map[string]string{
		http.MethodGet + " /connections/self":            permission.AccountRead,
		http.MethodPut + " /connections/self":            permission.AccountConnect,
		http.MethodDelete + " /connections/self":         permission.AccountConnect,
		http.MethodPut + " /subscriber-assignments/self": permission.AccountConnect,
		http.MethodGet + " /subscriber-states/self":      permission.AccountRead,
		http.MethodGet + " /traffic-days":                permission.AccountRead,
		http.MethodGet + " /money-entries":               permission.AccountRead,
		http.MethodGet + " /monthly-money-totals":        permission.AccountRead,
	}, guards)

	widgets, err := plugin.Widgets()
	require.NoError(t, err)
	require.Len(t, widgets, 1)
	require.Equal(t, permission.AccountRead, widgets[0].Meta().Permission)
}

// MAGTIACC-SC-019: The plugin declares the backfill command, and the migrations of its
// three tables.
func TestThePluginDeclaresTheCommandAndTheMigrations(t *testing.T) {
	t.Parallel()

	plugin := &MagtiPlugin{logger: slog.New(slog.DiscardHandler)}

	commands := plugin.Commands()
	require.Len(t, commands, 1)
	require.Equal(t, "backfill", commands[0].Name())

	migrations, err := plugin.Migrations()
	require.NoError(t, err)

	entries, err := fs.ReadDir(migrations, ".")
	require.NoError(t, err)

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	require.Equal(t, []string{
		"20261124120000_table_subscriber_assignments_create.down.sql",
		"20261124120000_table_subscriber_assignments_create.up.sql",
		"20261124120100_table_traffic_days_create.down.sql",
		"20261124120100_table_traffic_days_create.up.sql",
		"20261124120200_table_money_entries_create.down.sql",
		"20261124120200_table_money_entries_create.up.sql",
	}, names)
}

// MAGTIACC-SC-013: The job of the collection runs on the schedule of the configuration.
func TestThePluginDeclaresTheJob(t *testing.T) {
	t.Parallel()

	plugin := &MagtiPlugin{
		config: &config.Config{CronSchedule: config.CronSchedule{TrafficCollector: "0 9 * * *"}},
		logger: slog.New(slog.DiscardHandler),
	}

	jobs, err := plugin.CronJobs()
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "traffic_collector", jobs[0].Meta().ID)
	assert.Equal(t, pluginapi.CronScopePerWorkspace, jobs[0].Meta().Scope)
	assert.Equal(t, "0 9 * * *", jobs[0].Meta().Schedule)
}

// MAGTIACC-SC-028: A configuration with state_timeout above 4 seconds fails New.
func TestNewRefusesALongStateTimeout(t *testing.T) {
	t.Parallel()

	_, err := New(nil, map[string]any{"state_timeout": "6s"})
	require.Error(t, err)
}

// MAGTIACC-SC-027: The remote exports the subscriber, the traffic, the money, and the
// account.
func TestTheUIManifestNamesThePages(t *testing.T) {
	t.Parallel()

	manifest, err := (&MagtiPlugin{}).UIManifest()
	require.NoError(t, err)

	require.Equal(t, []pluginapi.UIRoute{
		{Path: "/subscriber", Label: "Subscriber"},
		{Path: "/traffic", Label: "Traffic"},
		{Path: "/money", Label: "Money"},
		{Path: "/account", Label: "Account"},
	}, manifest.Routes)
}
