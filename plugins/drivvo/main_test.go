package main

import (
	"log/slog"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/drivvo/config"
	"github.com/abgeo/maroid/plugins/drivvo/permission"
)

// DRIVVO-SC-001: The settings hold the email address and the password as a secret, both
// of the workspace and both required, and no other field.
func TestTheSettingsHoldTheCredentialsAlone(t *testing.T) {
	t.Parallel()

	model, err := (&DrivvoPlugin{}).SettingsModel()
	require.NoError(t, err)

	settings := reflect.TypeOf(model)
	require.Equal(t, reflect.TypeFor[config.WorkspaceSettings](), settings)
	require.Equal(t, 2, settings.NumField())

	email, found := settings.FieldByName("Email")
	require.True(t, found)
	assert.Equal(t, "email", email.Tag.Get("json"))
	assert.Equal(t, "title=Email address,format=email,required", email.Tag.Get("jsonschema"))
	assert.Equal(t, "x-maroid-scope=workspace", email.Tag.Get("jsonschema_extras"))

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

// DRIVVO-SC-021: account.read opens every read and the widget to a viewer, and
// account.connect opens the connect and the end of the connection to an editor.
func TestThePermissionsNameTheirLowestRoleAndTheirRoutes(t *testing.T) {
	t.Parallel()

	plugin := &DrivvoPlugin{logger: slog.New(slog.DiscardHandler)}

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
		http.MethodGet + " /connections/self":                              permission.AccountRead,
		http.MethodPut + " /connections/self":                              permission.AccountConnect,
		http.MethodDelete + " /connections/self":                           permission.AccountConnect,
		http.MethodGet + " /vehicles":                                      permission.AccountRead,
		http.MethodGet + " /vehicles/{vehicleId}/overview":                 permission.AccountRead,
		http.MethodGet + " /vehicles/{vehicleId}/log-entries":              permission.AccountRead,
		http.MethodGet + " /vehicles/{vehicleId}/log-entry-totals":         permission.AccountRead,
		http.MethodGet + " /vehicles/{vehicleId}/log-entries/{logEntryId}": permission.AccountRead,
		http.MethodGet + " /vehicles/{vehicleId}/reminders":                permission.AccountRead,
	}, guards)

	widgets, err := plugin.Widgets()
	require.NoError(t, err)
	require.Len(t, widgets, 1)
	require.Equal(t, permission.AccountRead, widgets[0].Meta().Permission)
}
