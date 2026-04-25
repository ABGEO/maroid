package registrar

import (
	"fmt"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	pluginui "github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// UIRegistrar is responsible for registering plugin UI manifests and assets.
type UIRegistrar struct {
	registry *pluginui.UIRegistry
}

var _ Registrar = (*UIRegistrar)(nil)

// NewUIRegistrar creates a new UIRegistrar.
func NewUIRegistrar(reg *pluginui.UIRegistry) *UIRegistrar {
	return &UIRegistrar{
		registry: reg,
	}
}

// Name returns the name of the registrar.
func (r *UIRegistrar) Name() string {
	return "ui"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *UIRegistrar) Supports(plugin pluginapi.Plugin) bool {
	_, ok := plugin.(pluginapi.UIPlugin)

	return ok
}

// Register handles the registration of a plugin UI manifest and assets.
func (r *UIRegistrar) Register(plugin pluginapi.Plugin) error {
	id := plugin.Meta().ID

	uiPlugin, ok := plugin.(pluginapi.UIPlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support UI capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	manifest, err := uiPlugin.UIManifest()
	if err != nil {
		return fmt.Errorf("retrieving UI manifest for plugin %s: %w", id, err)
	}

	r.registry.Register(id, manifest)

	return nil
}
