package main

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/telmico/config"
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
