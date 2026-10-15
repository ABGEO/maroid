package loader

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"plugin"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	"github.com/abgeo/maroid/apps/hub/internal/plugin/registrar"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	probePath   = "/plugins/probe.so"
	testDeckURL = "https://deck.example"
	workspaceID = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e78"
)

var errNoSymbol = errors.New("symbol not found")

// symbols is a symbolLookup over a map.
type symbols map[string]plugin.Symbol

func (s symbols) Lookup(name string) (plugin.Symbol, error) {
	symbol, ok := s[name]
	if !ok {
		return nil, errNoSymbol
	}

	return symbol, nil
}

// migratingPlugin declares one migration.
type migratingPlugin struct{}

func (migratingPlugin) Migrations() (fs.FS, error) {
	return fstest.MapFS{"1_init.up.sql": {Data: []byte("SELECT 1;")}}, nil
}

// spy is a constructor that keeps the host that it receives.
type spy struct {
	calls int
	host  pluginapi.Host
}

func (s *spy) constructor() pluginapi.Constructor {
	return func(host pluginapi.Host, _ map[string]any) (pluginapi.Plugin, error) {
		s.calls++
		s.host = host

		return migratingPlugin{}, nil
	}
}

type harness struct {
	logs       *bytes.Buffer
	loader     *Loader
	plugins    *registry.PluginRegistry
	migrations *registry.MigrationRegistry
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	logs := new(bytes.Buffer)

	binder, err := host.New(host.Dependencies{
		Config: &config.Config{Auth: config.Auth{DeckURL: testDeckURL}},
		Logger: slog.New(slog.NewTextHandler(logs, nil)),
	})
	require.NoError(t, err)

	plugins := registry.NewPluginRegistry()
	migrations := registry.NewMigrationRegistry()

	return &harness{
		logs: logs,
		loader: &Loader{
			binder:  binder,
			plugins: plugins,
			registrars: []registrar.Registrar{
				registrar.NewPluginRegistrar(plugins),
				registrar.NewMigrationRegistrar(migrations, registry.NewCapabilityRegistry()),
			},
		},
		plugins:    plugins,
		migrations: migrations,
	}
}

func declared(pluginID string) *pluginapi.Metadata {
	return &pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID(pluginID),
		Name:       "Probe",
		Version:    "1.0.0",
		APIVersion: pluginapi.APIVersion,
	}
}

func pageOf(t *testing.T, bound pluginapi.Host) string {
	t.Helper()

	pages, err := bound.Pages()
	require.NoError(t, err)

	ctx := pluginapi.ContextWithActingWorkspace(context.Background(), workspaceID)

	address, err := pages.PageURL(ctx, "/")
	require.NoError(t, err)

	return address
}

// PLGBIND-SC-001: The loader reads the metadata first, and the constructor receives a
// host bound to its identifier.
func TestTheConstructorReceivesAHostOfTheDeclaredPlugin(t *testing.T) {
	t.Parallel()

	built := &spy{}
	meta, constructor, err := openSymbols(probePath, symbols{
		MetadataSymbol:    declared("dev.maroid.probe"),
		ConstructorSymbol: new(built.constructor()),
	})
	require.NoError(t, err)

	require.NoError(t, newHarness(t).loader.load(probePath, meta, constructor, nil))

	require.Equal(t, 1, built.calls)
	assert.Contains(t, pageOf(t, built.host), "/plugins/dev.maroid.probe/")
}

// PLGBIND-SC-002: An object with no metadata, or with metadata of a wrong type, is
// refused, and the error names the file.
func TestAnObjectWithNoMetadataIsRefused(t *testing.T) {
	t.Parallel()

	built := &spy{}
	constructor := new(built.constructor())

	_, _, err := openSymbols(probePath, symbols{ConstructorSymbol: constructor})
	require.ErrorIs(t, err, errNoSymbol)
	assert.Contains(t, err.Error(), probePath)

	_, _, err = openSymbols(probePath, symbols{
		MetadataSymbol:    new("dev.maroid.probe"),
		ConstructorSymbol: constructor,
	})
	require.ErrorIs(t, err, errs.ErrUnexpectedPluginSymbolType)
	assert.Contains(t, err.Error(), probePath)

	assert.Zero(t, built.calls)
}

// PLGBIND-SC-003: A missing identifier, or one that does not parse, is refused before
// the constructor runs.
func TestAnInvalidIdentifierIsRefused(t *testing.T) {
	t.Parallel()

	missing := declared("dev.maroid.probe")
	missing.ID = nil

	handBuilt := declared("dev.maroid.probe")
	handBuilt.ID = &pluginapi.PluginID{Name: "GWP"}

	for label, meta := range map[string]*pluginapi.Metadata{
		"a missing identifier":    missing,
		"a hand built identifier": handBuilt,
	} {
		t.Run(label, func(t *testing.T) {
			t.Parallel()

			built := &spy{}
			err := newHarness(t).loader.load(probePath, meta, built.constructor(), nil)

			require.ErrorIs(t, err, errs.ErrInvalidPluginID)
			assert.Contains(t, err.Error(), probePath)
			assert.Zero(t, built.calls)
		})
	}
}

// PLGBIND-SC-004: A different API version is refused before the constructor runs.
func TestADifferentAPIVersionIsRefused(t *testing.T) {
	t.Parallel()

	meta := declared("dev.maroid.probe")
	meta.APIVersion = "v0"

	built := &spy{}
	err := newHarness(t).loader.load(probePath, meta, built.constructor(), nil)

	require.ErrorIs(t, err, errs.ErrIncompatiblePluginAPIVersion)
	assert.Zero(t, built.calls)
}

// PLGBIND-SC-006: A second plugin with a loaded identifier is refused before its
// constructor runs.
func TestASecondPluginWithALoadedIdentifierIsRefused(t *testing.T) {
	t.Parallel()

	scene := newHarness(t)
	first, second := &spy{}, &spy{}

	require.NoError(
		t,
		scene.loader.load(probePath, declared("dev.maroid.gwp"), first.constructor(), nil),
	)

	err := scene.loader.load(
		"/plugins/copy.so",
		declared("dev.maroid.gwp"),
		second.constructor(),
		nil,
	)

	require.ErrorIs(t, err, errs.ErrPluginAlreadyRegistered)
	assert.Contains(t, err.Error(), "/plugins/copy.so")
	assert.Zero(t, second.calls)
}

// PLGBIND-SC-008: The migrations of a plugin register under its declared identifier.
func TestTheMigrationsRegisterUnderTheDeclaredIdentifier(t *testing.T) {
	t.Parallel()

	scene := newHarness(t)

	require.NoError(
		t,
		scene.loader.load(probePath, declared("dev.maroid.gwp"), (&spy{}).constructor(), nil),
	)

	assert.Contains(t, scene.migrations.All(), "dev.maroid.gwp")
}

// PLGBIND-SC-009: A later write to the metadata that the plugin exports reaches neither
// the plugin registry, nor the bound host, nor its logger.
func TestALaterWriteToTheMetadataDoesNotReachTheHub(t *testing.T) {
	t.Parallel()

	scene := newHarness(t)
	built := &spy{}
	meta := declared("dev.maroid.gwp")

	require.NoError(t, scene.loader.load(probePath, meta, built.constructor(), nil))

	meta.ID.Name = "telasi"
	meta.Name = "Telasi"

	assert.True(t, scene.plugins.Has("dev.maroid.gwp"))
	assert.False(t, scene.plugins.Has("dev.maroid.telasi"))
	assert.Equal(t, "Probe", scene.plugins.All()[0].Meta.Name)
	assert.Contains(t, pageOf(t, built.host), "/plugins/dev.maroid.gwp/")

	built.host.Logger().Info("collected")
	assert.Contains(t, scene.logs.String(), "plugin=dev.maroid.gwp ")
	assert.NotContains(t, scene.logs.String(), "telasi")
}
