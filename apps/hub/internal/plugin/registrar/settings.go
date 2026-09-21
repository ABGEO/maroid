package registrar

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// SettingsRegistrar is responsible for registering the settings schema of a plugin.
type SettingsRegistrar struct {
	registry     *registry.SettingsRegistry
	capabilities *registry.CapabilityRegistry
}

var _ Registrar = (*SettingsRegistrar)(nil)

// NewSettingsRegistrar creates a new SettingsRegistrar.
func NewSettingsRegistrar(
	reg *registry.SettingsRegistry,
	capabilities *registry.CapabilityRegistry,
) *SettingsRegistrar {
	return &SettingsRegistrar{
		registry:     reg,
		capabilities: capabilities,
	}
}

// Name returns the name of the registrar.
func (r *SettingsRegistrar) Name() string {
	return "settings"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *SettingsRegistrar) Supports(plugin pluginapi.Plugin) bool {
	_, ok := plugin.(pluginapi.ConfigurablePlugin)

	return ok
}

// Register infers the settings schema of a plugin and registers it.
func (r *SettingsRegistrar) Register(plugin pluginapi.Plugin) error {
	id := plugin.Meta().ID

	configurable, ok := plugin.(pluginapi.ConfigurablePlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support Settings capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	settingsModel, err := configurable.SettingsModel()
	if err != nil {
		return fmt.Errorf("retrieving the settings model for plugin %s: %w", id, err)
	}

	schema, err := settings.Infer(settingsModel)
	if err != nil {
		return fmt.Errorf("inferring the settings schema for plugin %s: %w", id, err)
	}

	if err = r.registry.Register(id, schema); err != nil {
		return fmt.Errorf("registering the settings schema for plugin %s: %w", id, err)
	}

	r.capabilities.Record(id, registry.CapSettings, registry.Present)

	return nil
}
