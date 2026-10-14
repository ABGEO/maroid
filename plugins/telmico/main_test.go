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
	"github.com/abgeo/maroid/plugins/telmico/config"
	"github.com/abgeo/maroid/plugins/telmico/permission"
	"github.com/abgeo/maroid/plugins/telmico/service"
)

// ELECACC-SC-001: The settings hold the username as free text and the password as a
// secret, both of the workspace and both required, and no other field.
func TestTheSettingsHoldTheCredentialsAlone(t *testing.T) {
	t.Parallel()

	model, err := (&TelmicoPlugin{}).SettingsModel()
	require.NoError(t, err)

	settings := reflect.TypeOf(model)
	require.Equal(t, reflect.TypeFor[config.WorkspaceSettings](), settings)
	require.Equal(t, 2, settings.NumField())

	username, found := settings.FieldByName("Username")
	require.True(t, found)
	assert.Equal(t, "username", username.Tag.Get("json"))
	assert.Equal(t, "title=Username,required", username.Tag.Get("jsonschema"))
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

// ELECACC-SC-021: The plugin declares the bill as the notification topic monthly_bill.
func TestTheBillIsANotificationTopic(t *testing.T) {
	t.Parallel()

	topics, err := (&TelmicoPlugin{}).NotificationTopics()

	require.NoError(t, err)
	require.Equal(t, []pluginapi.NotificationTopic{{
		ID:          service.MonthlyBillTopic,
		Name:        "Monthly bill",
		Description: "The electricity bill of the last month, with its document.",
	}}, topics)
}

// ELECACC-SC-031: account.read opens the four reads and the widget to a viewer, and
// account.connect opens the three writes to an editor.
func TestThePermissionsNameTheirLowestRoleAndTheirEntries(t *testing.T) {
	t.Parallel()

	plugin := &TelmicoPlugin{
		config: &config.Config{BaseURL: "https://my.telmico.ge"},
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
		http.MethodGet + " /connections/self":          permission.AccountRead,
		http.MethodPut + " /connections/self":          permission.AccountConnect,
		http.MethodDelete + " /connections/self":       permission.AccountConnect,
		http.MethodPut + " /customer-assignments/self": permission.AccountConnect,
		http.MethodGet + " /transactions":              permission.AccountRead,
		http.MethodGet + " /bills":                     permission.AccountRead,
		http.MethodGet + " /bills/{billId}/document":   permission.AccountRead,
	}, guards)

	widgets, err := plugin.Widgets()
	require.NoError(t, err)
	require.Len(t, widgets, 1)
	require.Equal(t, permission.AccountRead, widgets[0].Meta().Permission)
}

// ELECACC-SC-019: The plugin declares the backfill command, and the migrations of its
// three tables.
func TestThePluginDeclaresTheCommandAndTheMigrations(t *testing.T) {
	t.Parallel()

	plugin := &TelmicoPlugin{logger: slog.New(slog.DiscardHandler)}

	commands := plugin.Commands()
	require.Len(t, commands, 1)
	require.Equal(t, "backfill", commands[0].Name())

	migrations, err := plugin.Migrations()
	require.NoError(t, err)

	entries, err := fs.ReadDir(migrations, ".")
	require.NoError(t, err)
	require.Len(t, entries, 6)
}

// ELECACC-SC-031: The remote exports the account, the transactions, and the bills.
func TestTheUIManifestNamesThePages(t *testing.T) {
	t.Parallel()

	manifest, err := (&TelmicoPlugin{}).UIManifest()
	require.NoError(t, err)

	require.Equal(t, []pluginapi.UIRoute{
		{Path: "/transactions", Label: "Transactions"},
		{Path: "/bills", Label: "Bills"},
		{Path: "/account", Label: "Account"},
	}, manifest.Routes)
}
