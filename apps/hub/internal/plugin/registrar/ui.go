package registrar

import (
	"fmt"
	"strings"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	pluginui "github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// UIRegistrar is responsible for registering plugin UI manifests and assets.
type UIRegistrar struct {
	registry     *pluginui.UIRegistry
	capabilities *pluginui.CapabilityRegistry
}

var _ Registrar = (*UIRegistrar)(nil)

// NewUIRegistrar creates a new UIRegistrar.
func NewUIRegistrar(
	reg *pluginui.UIRegistry,
	capabilities *pluginui.CapabilityRegistry,
) *UIRegistrar {
	return &UIRegistrar{
		registry:     reg,
		capabilities: capabilities,
	}
}

// Name returns the name of the registrar.
func (r *UIRegistrar) Name() string {
	return "ui"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *UIRegistrar) Supports(plugin pluginui.LoadedPlugin) bool {
	_, ok := plugin.Instance.(pluginapi.UIPlugin)

	return ok
}

// Register handles the registration of a plugin UI manifest and assets.
func (r *UIRegistrar) Register(plugin pluginui.LoadedPlugin) error {
	id := plugin.Meta.ID

	uiPlugin, ok := plugin.Instance.(pluginapi.UIPlugin)
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

	if manifest != nil {
		for _, route := range manifest.Routes {
			if hasParameterSegment(route.Path) {
				return fmt.Errorf(
					"checking the route %s of plugin %s: %w",
					route.Path,
					id,
					errs.ErrInvalidUIRoute,
				)
			}
		}
	}

	r.registry.Register(id, manifest)
	r.capabilities.Record(id, pluginui.CapUI, manifest)

	return nil
}

// hasParameterSegment reports whether a segment of the path takes a value, which
// the navigation cannot give.
func hasParameterSegment(path string) bool {
	for segment := range strings.SplitSeq(path, "/") {
		if strings.HasPrefix(segment, ":") {
			return true
		}
	}

	return false
}
