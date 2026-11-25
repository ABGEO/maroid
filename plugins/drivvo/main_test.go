package main

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/drivvo/config"
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
