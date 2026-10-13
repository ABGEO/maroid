package main

import (
	"log/slog"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/config"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/permission"
	"github.com/abgeo/maroid/plugins/tbilisi-energy/service"
)

// GASACC-SC-001: The settings hold the username as free text and the password as a
// secret, both of the workspace and both required, and no other field.
func TestTheSettingsHoldTheCredentialsAlone(t *testing.T) {
	t.Parallel()

	model, err := (&TbilisiEnergyPlugin{}).SettingsModel()
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

// GASACC-SC-022: The plugin declares the bill as the notification topic monthly_bill.
func TestTheBillIsANotificationTopic(t *testing.T) {
	t.Parallel()

	topics, err := (&TbilisiEnergyPlugin{}).NotificationTopics()

	require.NoError(t, err)
	require.Equal(t, []pluginapi.NotificationTopic{{
		ID:          service.MonthlyBillTopic,
		Name:        "Monthly bill",
		Description: "The gas bill of the last month, with its receipt.",
	}}, topics)
}

// GASACC-SC-031: account.read opens the two reads and the widget to a viewer, and
// account.connect opens the three writes to an editor.
func TestThePermissionsNameTheirLowestRoleAndTheirEntries(t *testing.T) {
	t.Parallel()

	plugin := &TbilisiEnergyPlugin{
		config: &config.Config{BaseURL: "https://my.te.ge"},
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
	}, guards)

	widgets, err := plugin.Widgets()
	require.NoError(t, err)
	require.Len(t, widgets, 1)
	require.Equal(t, permission.AccountRead, widgets[0].Meta().Permission)
}
