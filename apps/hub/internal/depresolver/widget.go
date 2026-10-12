package depresolver

import "github.com/abgeo/maroid/apps/hub/internal/registry"

// WidgetRegistry initializes and returns the registry of the widgets of the plugins.
func (c *Container) WidgetRegistry() *registry.WidgetRegistry {
	c.widgetRegistry.once.Do(func() {
		c.widgetRegistry.instance = registry.NewWidgetRegistry()
	})

	return c.widgetRegistry.instance
}
