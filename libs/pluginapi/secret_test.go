package pluginapi_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// secretRecorder is a SecretProvider that keeps the arguments of the last write.
type secretRecorder struct {
	plugin    *pluginapi.PluginID
	scope     pluginapi.SecretScope
	name      string
	expiresAt time.Time
}

func (r *secretRecorder) ReadSecret(
	_ context.Context,
	pluginID *pluginapi.PluginID,
	_ pluginapi.SecretScope,
	_ string,
) (string, error) {
	r.plugin = pluginID

	return "", pluginapi.ErrSecretAbsent
}

func (r *secretRecorder) WriteSecret(
	_ context.Context,
	pluginID *pluginapi.PluginID,
	scope pluginapi.SecretScope,
	name string,
	_ string,
	expiresAt time.Time,
) error {
	r.plugin, r.scope, r.name, r.expiresAt = pluginID, scope, name, expiresAt

	return nil
}

func (r *secretRecorder) DeleteSecret(
	_ context.Context,
	pluginID *pluginapi.PluginID,
	_ pluginapi.SecretScope,
	_ string,
) error {
	r.plugin = pluginID

	return nil
}

// PSECRET-SC-025: A plugin reaches no plugin secret of another plugin. This part
// covers the bind, because each call names the plugin that holds the reader.
func TestPluginSecretsNameThePluginThatHoldsThem(t *testing.T) {
	t.Parallel()

	provider := &secretRecorder{}
	secrets := pluginapi.NewPluginSecrets(provider, pluginapi.ParsePluginID("dev.maroid.parking"))

	_, err := secrets.Get(t.Context(), pluginapi.SecretScopeUser, "session")
	require.ErrorIs(t, err, pluginapi.ErrSecretAbsent)
	require.Equal(t, "dev.maroid.parking", provider.plugin.String())

	require.NoError(t, secrets.Delete(t.Context(), pluginapi.SecretScopeUser, "session"))
	require.Equal(t, "dev.maroid.parking", provider.plugin.String())
}

// PSECRET-SC-005: A write with Put replaces the expiry with none. This part covers
// the zero time that Put passes for no expiry.
func TestPutStoresNoExpiry(t *testing.T) {
	t.Parallel()

	provider := &secretRecorder{}
	secrets := pluginapi.NewPluginSecrets(provider, pluginapi.ParsePluginID("dev.maroid.parking"))

	require.NoError(t, secrets.Put(t.Context(), pluginapi.SecretScopeUser, "session", "token-1"))

	require.Equal(t, pluginapi.SecretScopeUser, provider.scope)
	require.Equal(t, "session", provider.name)
	require.True(t, provider.expiresAt.IsZero())
}

// PSECRET-SC-008: A write with PutUntil passes the expiry that the plugin gives.
func TestPutUntilPassesTheExpiry(t *testing.T) {
	t.Parallel()

	provider := &secretRecorder{}
	secrets := pluginapi.NewPluginSecrets(provider, pluginapi.ParsePluginID("dev.maroid.parking"))
	expiresAt := time.Date(2026, 10, 12, 10, 5, 0, 0, time.UTC)

	require.NoError(t, secrets.PutUntil(
		t.Context(), pluginapi.SecretScopeUser, "pending", "uuid-1", expiresAt,
	))

	require.Equal(t, expiresAt, provider.expiresAt)
}
