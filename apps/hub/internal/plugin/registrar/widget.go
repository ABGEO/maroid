package registrar

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const maxWidgetTitleLength = 64

// widgetKeyPattern keeps a widget key safe in an address.
var widgetKeyPattern = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

// WidgetRegistrar registers the widgets that a plugin shows on the home page.
type WidgetRegistrar struct {
	registry     *registry.WidgetRegistry
	capabilities *registry.CapabilityRegistry
	permissions  *registry.PermissionRegistry
}

var _ Registrar = (*WidgetRegistrar)(nil)

// NewWidgetRegistrar creates a new WidgetRegistrar.
func NewWidgetRegistrar(
	reg *registry.WidgetRegistry,
	capabilities *registry.CapabilityRegistry,
	permissions *registry.PermissionRegistry,
) *WidgetRegistrar {
	return &WidgetRegistrar{
		registry:     reg,
		capabilities: capabilities,
		permissions:  permissions,
	}
}

// Name returns the name of the registrar.
func (r *WidgetRegistrar) Name() string {
	return "widget"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *WidgetRegistrar) Supports(plugin registry.LoadedPlugin) bool {
	_, ok := plugin.Instance.(pluginapi.WidgetPlugin)

	return ok
}

// Register checks every widget of the plugin before it stores any, so a plugin
// with one broken widget loads none.
func (r *WidgetRegistrar) Register(plugin registry.LoadedPlugin) error {
	id := plugin.Meta.ID

	widgetPlugin, ok := plugin.Instance.(pluginapi.WidgetPlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support Widget capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	declared, err := widgetPlugin.Widgets()
	if err != nil {
		return fmt.Errorf("retrieving the widgets for plugin %s: %w", id, err)
	}

	widgets := make([]registry.Widget, 0, len(declared))
	items := make([]registry.WidgetItem, 0, len(declared))

	for _, declaredWidget := range declared {
		meta := declaredWidget.Meta()
		meta.Title = strings.TrimSpace(meta.Title)

		if err = validateWidget(id, meta); err != nil {
			return err
		}

		permission, permissionErr := permissionOf(r.permissions, id, meta.Key, meta.Permission)
		if permissionErr != nil {
			return permissionErr
		}

		widgets = append(widgets, registry.Widget{
			PluginID:   id.String(),
			Meta:       meta,
			Permission: permission,
			Load:       declaredWidget.Load,
		})
		items = append(items, registry.WidgetItem{
			PluginID:   id.String(),
			Key:        meta.Key,
			Title:      meta.Title,
			Kind:       string(meta.Kind),
			Size:       string(meta.Size),
			Permission: permission,
		})
	}

	if err = r.registry.Register(widgets...); err != nil {
		return fmt.Errorf("registering the widgets for plugin %s: %w", id, err)
	}

	r.capabilities.Record(id, registry.CapWidgets, items)

	return nil
}

func validateWidget(id *pluginapi.PluginID, meta pluginapi.WidgetMeta) error {
	var rule string

	switch titleLength := utf8.RuneCountInString(meta.Title); {
	case !widgetKeyPattern.MatchString(meta.Key):
		rule = "key"
	case titleLength == 0 || titleLength > maxWidgetTitleLength:
		rule = "title"
	case !knownKind(meta.Kind):
		rule = "kind"
	case !knownSize(meta.Size):
		rule = "size"
	default:
		return nil
	}

	return fmt.Errorf("%w: plugin %s, widget %s: %s", errs.ErrInvalidWidget, id, meta.Key, rule)
}

func knownKind(kind pluginapi.WidgetKind) bool {
	switch kind {
	case pluginapi.WidgetKindStat, pluginapi.WidgetKindList:
		return true
	default:
		return false
	}
}

func knownSize(size pluginapi.WidgetSize) bool {
	switch size {
	case pluginapi.WidgetSizeXS,
		pluginapi.WidgetSizeS,
		pluginapi.WidgetSizeM,
		pluginapi.WidgetSizeL:
		return true
	default:
		return false
	}
}
