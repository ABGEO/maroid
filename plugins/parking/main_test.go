package main

import (
	"log/slog"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/permission"
)

// PARKACC-SC-001: The settings hold the username as free text and the password as a
// secret, both of the user and both required, and no token or vehicle.
func TestTheSettingsHoldTheCredentialsAlone(t *testing.T) {
	t.Parallel()

	model, err := (&ParkingPlugin{}).SettingsModel()
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

// PARKACC-SC-022: account.connect has viewer as its lowest role.
func TestAccountConnectIsOpenToAViewer(t *testing.T) {
	t.Parallel()

	permissions, err := (&ParkingPlugin{}).Permissions()
	require.NoError(t, err)

	for _, declared := range permissions {
		if declared.Name == permission.AccountConnect {
			assert.Equal(t, pluginapi.RoleViewer, declared.Lowest)

			return
		}
	}

	t.Fatal("the plugin declares no account.connect")
}

// PARKHIST-SC-001: An empty config gives one job of each user, each hour at minute 15.
func TestTheHistoryCollectorRunsEachHour(t *testing.T) {
	t.Parallel()

	cfg := new(config.Config)
	require.NoError(t, pluginconfig.DecodeAndValidateConfig(map[string]any{}, cfg))

	jobs, err := (&ParkingPlugin{config: cfg, logger: slog.New(slog.DiscardHandler)}).CronJobs()
	require.NoError(t, err)

	require.Len(t, jobs, 1)
	assert.Equal(t, pluginapi.CronJobMeta{
		ID:       "history_collector",
		Schedule: "15 * * * *",
		Scope:    pluginapi.CronScopePerUser,
	}, jobs[0].Meta())
	assert.Equal(t, 5, cfg.CollectorPages)
}
