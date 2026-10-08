package provider_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/dex"
	"github.com/abgeo/maroid/apps/hub/internal/dex/dextest"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/provider"
)

const issuer = "https://auth.maroid.localhost"

func managerOf(t *testing.T, connectors ...dex.Connector) *provider.Manager {
	t.Helper()

	return provider.NewManager(
		dextest.New(connectors...),
		migrated(t).DB,
		provider.Settings{Issuer: issuer},
	)
}

// IDPROV-SC-001: A connector with no maroidPreset is static and carries no preset.
// A connector that the hub stored carries the preset that its config names.
func TestTheListMarksAStaticProvider(t *testing.T) {
	t.Parallel()

	manager := managerOf(t,
		dextest.Connector("telegram", "oidc", "Telegram",
			`{"issuer":"https://oauth.telegram.org","clientID":"1","clientSecret":"s",`+
				`"userIDKey":"id","maroidPreset":"telegram"}`),
		dextest.Connector("mock", "mockCallback", "Mock", `{}`),
	)

	providers, err := manager.List(t.Context())
	require.NoError(t, err)
	require.Len(t, providers, 2)

	assert.Equal(t, "mock", providers[0].ID, "the list is ordered by the identifier")
	assert.True(t, providers[0].Static)
	assert.Empty(t, providers[0].Preset)
	assert.Equal(t, "Mock", providers[0].Name)

	assert.Equal(t, "telegram", providers[1].ID)
	assert.False(t, providers[1].Static)
	assert.Equal(t, provider.PresetTelegram, providers[1].Preset)
}

// IDPROV-SC-001: A config that is not a JSON object carries no mark, so it is static.
func TestAConfigThatIsNoObjectIsStatic(t *testing.T) {
	t.Parallel()

	providers, err := managerOf(
		t,
		dextest.Connector("odd", "github", "Odd", `[]`),
	).List(t.Context())
	require.NoError(t, err)
	require.Len(t, providers, 1)
	assert.True(t, providers[0].Static)
}

// IDPROV-SC-007: Every provider carries the redirect address of Dex, so the form of a
// new provider reads it from the list.
func TestEveryProviderCarriesTheRedirectAddress(t *testing.T) {
	t.Parallel()

	manager := managerOf(t,
		dextest.Connector("abgeo-cloud", "oidc", "ABGEO.cloud",
			`{"issuer":"https://auth.abgeo.cloud","maroidPreset":"oidc"}`),
		dextest.Connector("local", "local", "Email", `{"maroidPreset":"local"}`),
	)

	cloud, err := manager.Get(t.Context(), "abgeo-cloud")
	require.NoError(t, err)
	assert.Equal(t, "https://auth.maroid.localhost/callback", cloud.RedirectURI)

	local, err := manager.Get(t.Context(), "local")
	require.NoError(t, err)
	assert.Equal(t, provider.PresetLocal, local.Preset)
	assert.Equal(t, "https://auth.maroid.localhost/callback", local.RedirectURI)
}

// IDPROV-SC-010: The answer says that a secret exists and never carries it. The
// options hold every key that no preset field covers.
func TestTheProviderSplitsItsConfig(t *testing.T) {
	t.Parallel()

	manager := managerOf(t, dextest.Connector("abgeo-cloud", "oidc", "ABGEO.cloud", `{
		"issuer": "https://auth.abgeo.cloud",
		"clientID": "cid",
		"clientSecret": "s1",
		"redirectURI": "https://auth.maroid.localhost/callback",
		"userIDKey": "sub",
		"scopes": ["openid", "email"],
		"getUserInfo": true,
		"maroidPreset": "oidc"
	}`))

	cloud, err := manager.Get(t.Context(), "abgeo-cloud")
	require.NoError(t, err)

	assert.Equal(t, provider.PresetOIDC, cloud.Preset)
	assert.Equal(t, "https://auth.abgeo.cloud", cloud.Issuer)
	assert.Equal(t, "cid", cloud.ClientID)
	assert.True(t, cloud.ClientSecretSet)
	assert.Equal(t, "sub", cloud.UserIDKey)
	assert.Equal(t, []string{scopeOpenID, "email"}, cloud.Scopes)
	assert.Equal(
		t,
		map[string]json.RawMessage{"getUserInfo": json.RawMessage("true")},
		cloud.Options,
	)
}

// IDPROV-SC-004: The Telegram preset fixes its scopes, so the answer carries none.
func TestTheTelegramProviderCarriesNoScopes(t *testing.T) {
	t.Parallel()

	telegram, err := managerOf(t, dextest.Connector("telegram", "oidc", "Telegram",
		`{"scopes":["openid","profile"],"userIDKey":"id","maroidPreset":"telegram"}`),
	).Get(t.Context(), "telegram")
	require.NoError(t, err)

	assert.Equal(t, "id", telegram.UserIDKey)
	assert.Nil(t, telegram.Scopes)
	assert.Empty(t, telegram.Options)
}

// IDPROV-SC-001: An identifier that Dex does not hold is not found.
func TestAProviderThatDexLacksIsNotFound(t *testing.T) {
	t.Parallel()

	_, err := managerOf(t).Get(t.Context(), "nope")
	require.ErrorIs(t, err, errs.ErrProviderNotFound)
}

// IDPROV-SC-029: A Dex that does not answer reaches the caller as unavailable.
func TestAnUnavailableDexReachesTheCaller(t *testing.T) {
	t.Parallel()

	memory := dextest.New()
	memory.Fail(errs.ErrIDPUnavailable)

	manager := provider.NewManager(memory, migrated(t).DB, provider.Settings{Issuer: issuer})

	_, err := manager.List(t.Context())
	require.ErrorIs(t, err, errs.ErrIDPUnavailable)
}
