package host_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// secretRecorder is a secret store that keeps the arguments of the last write.
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

// memorySecrets is a secret store in memory, keyed by the plugin, the scope, and the name.
type memorySecrets map[string]string

func (m memorySecrets) ReadSecret(
	_ context.Context,
	pluginID *pluginapi.PluginID,
	scope pluginapi.SecretScope,
	name string,
) (string, error) {
	value, ok := m[m.key(pluginID, scope, name)]
	if !ok {
		return "", pluginapi.ErrSecretAbsent
	}

	return value, nil
}

func (m memorySecrets) WriteSecret(
	_ context.Context,
	pluginID *pluginapi.PluginID,
	scope pluginapi.SecretScope,
	name string,
	value string,
	_ time.Time,
) error {
	m[m.key(pluginID, scope, name)] = value

	return nil
}

func (m memorySecrets) DeleteSecret(
	_ context.Context,
	pluginID *pluginapi.PluginID,
	scope pluginapi.SecretScope,
	name string,
) error {
	delete(m, m.key(pluginID, scope, name))

	return nil
}

func (m memorySecrets) key(
	pluginID *pluginapi.PluginID,
	scope pluginapi.SecretScope,
	name string,
) string {
	return pluginID.String() + "/" + string(scope) + "/" + name
}

func secretsOf(t *testing.T, store host.SecretStore, pluginID string) pluginapi.PluginSecrets {
	t.Helper()

	secrets, err := bind(t, host.Dependencies{Secrets: store}, pluginID).Secrets()
	require.NoError(t, err)

	return secrets
}

// PSECRET-SC-025: A plugin reaches no plugin secret of another plugin. This part
// covers the bind, because each call names the plugin that holds the reader.
func TestPluginSecretsNameThePluginThatHoldsThem(t *testing.T) {
	t.Parallel()

	provider := &secretRecorder{}
	secrets := secretsOf(t, provider, "dev.maroid.parking")

	_, err := secrets.Get(t.Context(), pluginapi.SecretScopeUser, "session")
	require.ErrorIs(t, err, pluginapi.ErrSecretAbsent)
	require.Equal(t, "dev.maroid.parking", provider.plugin.String())

	require.NoError(t, secrets.Delete(t.Context(), pluginapi.SecretScopeUser, "session"))
	require.Equal(t, "dev.maroid.parking", provider.plugin.String())
}

// PLGBIND-SC-011: The plugin secret of one plugin reads as absent for another plugin.
func TestAPluginSecretBelongsToTheBoundPlugin(t *testing.T) {
	t.Parallel()

	store := memorySecrets{}
	gwp := secretsOf(t, store, "dev.maroid.gwp")
	telasi := secretsOf(t, store, "dev.maroid.telasi")

	require.NoError(t, gwp.Put(t.Context(), pluginapi.SecretScopeWorkspace, "token", "gwp-1"))

	_, err := telasi.Get(t.Context(), pluginapi.SecretScopeWorkspace, "token")
	require.ErrorIs(t, err, pluginapi.ErrSecretAbsent)

	value, err := gwp.Get(t.Context(), pluginapi.SecretScopeWorkspace, "token")
	require.NoError(t, err)
	require.Equal(t, "gwp-1", value)
}

// PSECRET-SC-005: A write with Put replaces the expiry with none. This part covers
// the zero time that Put passes for no expiry.
func TestPutStoresNoExpiry(t *testing.T) {
	t.Parallel()

	provider := &secretRecorder{}
	secrets := secretsOf(t, provider, "dev.maroid.parking")

	require.NoError(t, secrets.Put(t.Context(), pluginapi.SecretScopeUser, "session", "token-1"))

	require.Equal(t, pluginapi.SecretScopeUser, provider.scope)
	require.Equal(t, "session", provider.name)
	require.True(t, provider.expiresAt.IsZero())
}

// PSECRET-SC-008: A write with PutUntil passes the expiry that the plugin gives.
func TestPutUntilPassesTheExpiry(t *testing.T) {
	t.Parallel()

	provider := &secretRecorder{}
	secrets := secretsOf(t, provider, "dev.maroid.parking")
	expiresAt := time.Date(2026, 10, 12, 10, 5, 0, 0, time.UTC)

	require.NoError(t, secrets.PutUntil(
		t.Context(), pluginapi.SecretScopeUser, "pending", "uuid-1", expiresAt,
	))

	require.Equal(t, expiresAt, provider.expiresAt)
}
