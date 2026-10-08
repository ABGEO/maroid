package provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/dex"
	"github.com/abgeo/maroid/apps/hub/internal/dex/dextest"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/provider"
	"github.com/abgeo/maroid/libs/rest/precondition"
)

const (
	botID       = "8514702227"
	cloudIssuer = "https://auth.abgeo.cloud"
	idCloud     = "abgeo-cloud"
	presetOIDC  = `{"maroidPreset":"oidc"}`
	scopeOpenID = "openid"
	testTimeout = 5 * time.Second
)

var errNoDocument = errors.New("no discovery document")

// discoverer answers a fixed result for every issuer.
type discoverer struct{ failure error }

func (d discoverer) Discover(_ context.Context, _ string) error { return d.failure }

func writable(t *testing.T, connectors ...dex.Connector) (*provider.Manager, *dextest.Memory) {
	t.Helper()

	memory := dextest.New(connectors...)

	return provider.NewManager(memory, provider.Settings{
		Issuer:        issuer,
		TelegramBotID: botID,
		Discoverer:    discoverer{},
	}), memory
}

func storedConfig(t *testing.T, memory *dextest.Memory, id string) map[string]any {
	t.Helper()

	connectors, err := memory.ListConnectors(t.Context())
	require.NoError(t, err)

	for _, one := range connectors {
		if one.ID == id {
			var config map[string]any
			require.NoError(t, json.Unmarshal(one.Config, &config))

			return config
		}
	}

	t.Fatalf("Dex holds no connector %s", id)

	return nil
}

func cloudInput() provider.Input {
	return provider.Input{
		Preset:       provider.PresetOIDC,
		ID:           new(idCloud),
		Name:         new("ABGEO.cloud"),
		Issuer:       new(cloudIssuer),
		ClientID:     new("cid"),
		ClientSecret: new("s1"),
	}
}

// IDPROV-SC-002: A generic OIDC provider with no claim and no scopes takes the
// defaults, and Dex holds it with the mark of its preset.
func TestAGenericProviderTakesTheDefaults(t *testing.T) {
	t.Parallel()

	manager, memory := writable(t)

	created, err := manager.Create(t.Context(), cloudInput())
	require.NoError(t, err)

	assert.Equal(t, "sub", created.UserIDKey)
	assert.Equal(t, []string{scopeOpenID, "profile", "email"}, created.Scopes)

	config := storedConfig(t, memory, idCloud)
	assert.Equal(t, "oidc", config[provider.MarkerKey])
	assert.Equal(t, cloudIssuer, config["issuer"])
	assert.Equal(t, "s1", config["clientSecret"])
	assert.Equal(t, issuer+"/callback", config["redirectURI"])
}

// IDPROV-SC-003: A reserved identifier and a malformed one name /id. An identifier
// that Dex holds, static or stored, already exists. Dex holds nothing new.
func TestAnIdentifierIsChecked(t *testing.T) {
	t.Parallel()

	manager, memory := writable(t,
		dextest.Connector("mock", "mockCallback", "Mock", `{}`),
		dextest.Connector(idCloud, "oidc", "Cloud", presetOIDC),
	)

	for _, id := range []string{"telegram", "local", "Bad_Id", "a"} {
		input := cloudInput()
		input.ID = new(id)

		_, err := manager.Create(t.Context(), input)
		requireField(t, err, "/id")
	}

	for _, id := range []string{"mock", idCloud} {
		input := cloudInput()
		input.ID = new(id)

		_, err := manager.Create(t.Context(), input)
		require.ErrorIs(t, err, errs.ErrProviderExists)
	}

	connectors, err := memory.ListConnectors(t.Context())
	require.NoError(t, err)
	assert.Len(t, connectors, 2)
}

// IDPROV-SC-002: A generic OIDC provider needs its name, an https issuer, and its
// client.
func TestAGenericProviderNeedsItsFields(t *testing.T) {
	t.Parallel()

	manager, _ := writable(t)

	cases := map[string]func(*provider.Input){
		"/name":          func(input *provider.Input) { input.Name = new("  ") },
		"/issuer":        func(input *provider.Input) { input.Issuer = new("http://auth.abgeo.cloud") },
		"/client_id":     func(input *provider.Input) { input.ClientID = nil },
		"/client_secret": func(input *provider.Input) { input.ClientSecret = new("") },
		"/scopes":        func(input *provider.Input) { input.Scopes = new([]string{}) },
	}

	for pointer, change := range cases {
		input := cloudInput()
		change(&input)

		_, err := manager.Create(t.Context(), input)
		requireField(t, err, pointer)
	}
}

// IDPROV-SC-004: The Telegram preset fixes its identifier, its name, its issuer, its
// claim, and its scopes. An absent client identifier takes the one of the bot.
func TestTheTelegramPresetFixesItsFields(t *testing.T) {
	t.Parallel()

	manager, memory := writable(t)

	created, err := manager.Create(t.Context(), provider.Input{
		Preset:       provider.PresetTelegram,
		ClientSecret: new("tg-secret"),
	})
	require.NoError(t, err)

	assert.Equal(t, "telegram", created.ID)
	assert.Equal(t, "Telegram", created.Name)
	assert.Equal(t, botID, created.ClientID)

	config := storedConfig(t, memory, "telegram")
	assert.Equal(t, "https://oauth.telegram.org", config["issuer"])
	assert.Equal(t, "id", config["userIDKey"])
	assert.Equal(t, []any{scopeOpenID, "profile"}, config["scopes"])
	assert.Equal(t, "telegram", config[provider.MarkerKey])

	_, err = manager.Create(t.Context(), provider.Input{
		Preset: provider.PresetTelegram, ClientSecret: new("x"), Issuer: new(cloudIssuer),
	})
	requireField(t, err, "/issuer")
}

// IDPROV-SC-005: A second Telegram provider and a second local provider exist already.
func TestThePresetsOfOneKindExistOnce(t *testing.T) {
	t.Parallel()

	manager, _ := writable(t,
		dextest.Connector("telegram", "oidc", "Telegram", `{"maroidPreset":"telegram"}`),
		dextest.Connector("local", "local", "Maroid", `{"maroidPreset":"local"}`),
	)

	_, err := manager.Create(t.Context(), provider.Input{
		Preset: provider.PresetTelegram, ClientSecret: new("x"),
	})
	require.ErrorIs(t, err, errs.ErrProviderExists)

	_, err = manager.Create(t.Context(), provider.Input{Preset: provider.PresetLocal})
	require.ErrorIs(t, err, errs.ErrProviderExists)
}

// IDPROV-SC-006: The local preset takes no input, and Dex holds it named Maroid.
func TestTheLocalPresetTakesNoInput(t *testing.T) {
	t.Parallel()

	manager, memory := writable(t)

	_, err := manager.Create(t.Context(), provider.Input{
		Preset: provider.PresetLocal, Name: new("Local"),
	})
	requireField(t, err, "/name")

	created, err := manager.Create(t.Context(), provider.Input{Preset: provider.PresetLocal})
	require.NoError(t, err)
	assert.Equal(t, "local", created.ID)
	assert.Equal(t, "Maroid", created.Name)

	connectors, err := memory.ListConnectors(t.Context())
	require.NoError(t, err)
	require.Len(t, connectors, 1)
	assert.Equal(t, "local", connectors[0].Type)
	assert.JSONEq(t, `{"maroidPreset":"local"}`, string(connectors[0].Config))
}

// IDPROV-SC-008: An option lands. A key that the preset fixes, a key that Dex lacks,
// and a value of the wrong type change nothing.
func TestTheOptionsOfAChangeAreChecked(t *testing.T) {
	t.Parallel()

	manager, memory := writable(t)
	_, err := manager.Create(t.Context(), cloudInput())
	require.NoError(t, err)

	_, err = manager.Change(t.Context(), idCloud, provider.Change{
		Options: optionsOf(t, `{"getUserInfo": true}`),
	}, nil)
	require.NoError(t, err)

	for text, pointer := range map[string]string{
		`{"userIDKey": "email"}`: "/options/userIDKey",
		`{"getUserinfo": true}`:  "/options/getUserinfo",
		`{"getUserInfo": "yes"}`: "/options/getUserInfo",
	} {
		_, err = manager.Change(t.Context(), idCloud, provider.Change{
			Options: optionsOf(t, text),
		}, nil)
		requireField(t, err, pointer)
	}

	config := storedConfig(t, memory, idCloud)
	assert.Equal(t, true, config["getUserInfo"])
	assert.NotContains(t, config, "getUserinfo")

	_, err = manager.Change(t.Context(), idCloud, provider.Change{
		Options: optionsOf(t, `{"getUserInfo": null}`),
	}, nil)
	require.NoError(t, err)
	assert.NotContains(t, storedConfig(t, memory, idCloud), "getUserInfo",
		"a null member removes the option, as merge patch gives")
}

// IDPROV-SC-009: A rename, a new client, and new scopes land. The Telegram preset
// refuses scopes, and the local preset refuses a client.
func TestAChangeLandsOnTheFieldsThatChange(t *testing.T) {
	t.Parallel()

	manager, memory := writable(t,
		dextest.Connector("telegram", "oidc", "Telegram",
			`{"maroidPreset":"telegram","scopes":["openid","profile"]}`),
		dextest.Connector("local", "local", "Maroid", `{"maroidPreset":"local"}`),
	)
	_, err := manager.Create(t.Context(), cloudInput())
	require.NoError(t, err)

	changed, err := manager.Change(t.Context(), idCloud, provider.Change{
		Name:     new("Cloud"),
		ClientID: new("cid2"),
		Scopes:   new([]string{scopeOpenID}),
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "Cloud", changed.Name)
	assert.Equal(t, "cid2", changed.ClientID)
	assert.Equal(t, []string{scopeOpenID}, changed.Scopes)
	assert.Equal(t, cloudIssuer, storedConfig(t, memory, idCloud)["issuer"])

	_, err = manager.Change(t.Context(), "telegram", provider.Change{
		Scopes: new([]string{scopeOpenID}),
	}, nil)
	requireField(t, err, "/scopes")

	_, err = manager.Change(t.Context(), "local", provider.Change{ClientID: new("x")}, nil)
	requireField(t, err, "/client_id")

	renamed, err := manager.Change(t.Context(), "local", provider.Change{Name: new("Home")}, nil)
	require.NoError(t, err)
	assert.Equal(t, "Home", renamed.Name)
}

// IDPROV-SC-010: A change with no secret keeps the secret, and a change with one
// replaces it.
func TestAChangeKeepsTheSecretItDoesNotName(t *testing.T) {
	t.Parallel()

	manager, memory := writable(t)
	_, err := manager.Create(t.Context(), cloudInput())
	require.NoError(t, err)

	_, err = manager.Change(t.Context(), idCloud, provider.Change{Name: new("Cloud")}, nil)
	require.NoError(t, err)
	assert.Equal(t, "s1", storedConfig(t, memory, idCloud)["clientSecret"])

	changed, err := manager.Change(t.Context(), idCloud, provider.Change{
		ClientSecret: new("s2"),
	}, nil)
	require.NoError(t, err)
	assert.True(t, changed.ClientSecretSet)
	assert.Equal(t, "s2", storedConfig(t, memory, idCloud)["clientSecret"])
}

// IDPROV-SC-011: An issuer with no discovery document names /issuer, and Dex holds
// nothing new.
func TestAnIssuerWithNoDocumentIsRefused(t *testing.T) {
	t.Parallel()

	memory := dextest.New()
	manager := provider.NewManager(memory, provider.Settings{
		Issuer: issuer, Discoverer: discoverer{failure: errNoDocument},
	})

	_, err := manager.Create(t.Context(), cloudInput())
	requireField(t, err, "/issuer")

	connectors, err := memory.ListConnectors(t.Context())
	require.NoError(t, err)
	assert.Empty(t, connectors)
}

// IDPROV-SC-011: The discovery of go-oidc refuses a server that answers 404 and an
// issuer that does not resolve, and accepts a document whose issuer matches.
func TestTheDiscoveryReadsTheDocument(t *testing.T) {
	t.Parallel()

	missing := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(missing.Close)

	var served *httptest.Server

	served = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"issuer":"` + served.URL + `","authorization_endpoint":"` +
			served.URL + `/auth","token_endpoint":"` + served.URL + `/token","jwks_uri":"` +
			served.URL + `/keys"}`))
	}))
	t.Cleanup(served.Close)

	discovery := provider.OIDCDiscovery{Timeout: testTimeout}

	require.Error(t, discovery.Discover(t.Context(), missing.URL))
	require.Error(t, discovery.Discover(t.Context(), "https://nowhere.invalid"))
	require.NoError(t, discovery.Discover(t.Context(), served.URL))
}

// IDPROV-SC-015: A change of a static provider is refused, and Dex keeps it.
func TestAStaticProviderTakesNoChange(t *testing.T) {
	t.Parallel()

	manager, memory := writable(t, dextest.Connector("mock", "mockCallback", "Mock", `{}`))

	_, err := manager.Change(t.Context(), "mock", provider.Change{Name: new("Mock 2")}, nil)
	require.ErrorIs(t, err, errs.ErrProviderStatic)

	connectors, err := memory.ListConnectors(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "Mock", connectors[0].Name)
}

// IDPROV-DD-013: A change that names an older version moves nothing, and one that
// names the current version lands.
func TestAChangeChecksTheVersion(t *testing.T) {
	t.Parallel()

	manager, _ := writable(t)
	created, err := manager.Create(t.Context(), cloudInput())
	require.NoError(t, err)

	stale := created.Version + 1

	_, err = manager.Change(t.Context(), idCloud, provider.Change{Name: new("A")}, &stale)
	require.ErrorIs(t, err, precondition.ErrModified)

	changed, err := manager.Change(t.Context(), idCloud, provider.Change{Name: new("A")},
		&created.Version)
	require.NoError(t, err)
	assert.NotEqual(t, created.Version, changed.Version, "a change moves the version")
	assert.Positive(t, changed.Version)
}
