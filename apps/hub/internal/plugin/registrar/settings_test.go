package registrar_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/plugin/registrar"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// configurable is a plugin that declares the settings model it receives.
type configurable struct {
	id    string
	model any
}

func (p *configurable) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginapi.ParsePluginID(p.id),
		Version:    "0.1.0",
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *configurable) SettingsModel() (any, error) {
	return p.model, nil
}

const probeID = "dev.maroid.probe"

type probeModel struct {
	Email string `json:"email" jsonschema:"title=Email,required"`
}

// PSET-SC-001: A plugin whose model is not a struct fails the load, and the registry
// holds no entry for it.
func TestRegisterRejectsAModelThatIsNotAStruct(t *testing.T) {
	t.Parallel()

	reg := registry.NewSettingsRegistry()
	settingsRegistrar := registrar.NewSettingsRegistrar(reg)

	err := settingsRegistrar.Register(&configurable{id: probeID, model: "not a struct"})

	require.ErrorContains(t, err, "settings model")

	_, found := reg.Get(probeID)
	require.False(t, found)
}

// PSET-FR-001: A plugin that declares a struct reaches the registry.
func TestRegisterHoldsTheSchemaOfThePlugin(t *testing.T) {
	t.Parallel()

	reg := registry.NewSettingsRegistry()
	settingsRegistrar := registrar.NewSettingsRegistrar(reg)
	plugin := &configurable{id: probeID, model: &probeModel{}}

	require.True(t, settingsRegistrar.Supports(plugin))
	require.NoError(t, settingsRegistrar.Register(plugin))

	schema, found := reg.Get(probeID)
	require.True(t, found)
	require.Contains(t, string(schema.Document), `"email"`)
}

// PLG-011: A second registration under the same identifier returns a sentinel error.
func TestRegisterRejectsASecondSchemaForOnePlugin(t *testing.T) {
	t.Parallel()

	reg := registry.NewSettingsRegistry()
	settingsRegistrar := registrar.NewSettingsRegistrar(reg)
	plugin := &configurable{id: probeID, model: &probeModel{}}

	require.NoError(t, settingsRegistrar.Register(plugin))
	require.ErrorContains(t, settingsRegistrar.Register(plugin), "already registered")
}
