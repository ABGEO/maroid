package main

import (
	"log/slog"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pensions/config"
	"github.com/abgeo/maroid/plugins/pensions/permission"
)

// PENSION-SC-001: The settings hold the username as free text and the password as a
// secret, both of the user and both required, and no other field.
func TestTheSettingsHoldTheCredentialsOfTheUserAlone(t *testing.T) {
	t.Parallel()

	model, err := (&PensionsPlugin{}).SettingsModel()
	require.NoError(t, err)

	settings := reflect.TypeOf(model)
	require.Equal(t, reflect.TypeFor[config.UserSettings](), settings)
	require.Equal(t, 2, settings.NumField())

	username, found := settings.FieldByName("Username")
	require.True(t, found)
	assert.Equal(t, "username", username.Tag.Get("json"))
	assert.Equal(t, "title=Username,required", username.Tag.Get("jsonschema"))
	assert.Equal(t, "x-maroid-scope=user", username.Tag.Get("jsonschema_extras"))

	password, found := settings.FieldByName("Password")
	require.True(t, found)
	assert.Equal(t, "password", password.Tag.Get("json"))
	assert.Equal(
		t,
		"title=Password,format=password,writeOnly=true,required",
		password.Tag.Get("jsonschema"),
	)
	assert.Equal(t, "x-maroid-scope=user", password.Tag.Get("jsonschema_extras"))
}

// PENSION-SC-013: The plugin declares one job, which runs for each user, and the
// backfill command.
func TestThePluginDeclaresTheJobAndTheCommand(t *testing.T) {
	t.Parallel()

	plugin := &PensionsPlugin{config: &config.Config{}, logger: slog.New(slog.DiscardHandler)}

	jobs, err := plugin.CronJobs()
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "account_collector", jobs[0].Meta().ID)
	assert.Equal(t, pluginapi.CronScopePerUser, jobs[0].Meta().Scope)

	commands := plugin.Commands()
	require.Len(t, commands, 1)
	assert.Equal(t, "backfill", commands[0].Name())

	_, err = plugin.Migrations()
	require.NoError(t, err)
}

// declarations answers the lowest role of each permission, and the permission of each
// route by method and pattern.
func declarations(
	t *testing.T,
	plugin *PensionsPlugin,
) (map[string]pluginapi.Role, map[string]string) {
	t.Helper()

	permissions, err := plugin.Permissions()
	require.NoError(t, err)

	lowest := map[string]pluginapi.Role{}
	for _, declared := range permissions {
		lowest[declared.Name] = declared.Lowest
	}

	routes, err := plugin.Routes()
	require.NoError(t, err)

	guards := map[string]string{}
	for _, route := range routes {
		guards[route.Method+" "+route.Pattern] = route.Permission
	}

	return lowest, guards
}

// PENSION-SC-028: account.connect opens the three routes of the connection to a viewer.
func TestTheConnectionNeedsAPermissionOfAViewer(t *testing.T) {
	t.Parallel()

	lowest, guards := declarations(t, &PensionsPlugin{logger: slog.New(slog.DiscardHandler)})

	assert.Equal(t, pluginapi.RoleViewer, lowest[permission.AccountConnect])

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		assert.Equal(t, permission.AccountConnect, guards[method+" /connections/self"], method)
	}
}

// PENSION-SC-044: account.read opens every read route and the widget to a viewer.
func TestTheReadsNeedAPermissionOfAViewer(t *testing.T) {
	t.Parallel()

	plugin := &PensionsPlugin{logger: slog.New(slog.DiscardHandler)}
	lowest, guards := declarations(t, plugin)

	assert.Equal(t, map[string]pluginapi.Role{
		permission.AccountRead:    pluginapi.RoleViewer,
		permission.AccountConnect: pluginapi.RoleViewer,
	}, lowest)

	for _, path := range []string{
		"/account-summaries/self",
		"/asset-values",
		"/contribution-months",
		"/contributions",
		"/contributions/total",
		"/employers",
		"/portfolio-shares",
		"/portfolio-unit-prices",
	} {
		assert.Equal(t, permission.AccountRead, guards[http.MethodGet+" "+path], path)
	}

	assert.Len(t, guards, 11)

	widgets, err := plugin.Widgets()
	require.NoError(t, err)
	require.Len(t, widgets, 1)
	assert.Equal(t, "asset-value", widgets[0].Meta().Key)
	assert.Equal(t, permission.AccountRead, widgets[0].Meta().Permission)
}
