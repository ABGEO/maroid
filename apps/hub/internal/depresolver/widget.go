package depresolver

import (
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/widget"
)

// WidgetRegistry initializes and returns the registry of the widgets of the plugins.
func (c *Container) WidgetRegistry() *registry.WidgetRegistry {
	c.widgetRegistry.once.Do(func() {
		c.widgetRegistry.instance = registry.NewWidgetRegistry()
	})

	return c.widgetRegistry.instance
}

// buildWidgetHandler resolves every dependency of the handler of the widgets.
func (c *Container) buildWidgetHandler() (*handler.Widget, error) {
	verifier, err := c.TokenVerifier()
	if err != nil {
		return nil, err
	}

	identityResolver, err := c.IdentityResolver()
	if err != nil {
		return nil, err
	}

	access, err := c.workspaceAccess()
	if err != nil {
		return nil, err
	}

	enablements, err := c.EnablementService()
	if err != nil {
		return nil, err
	}

	return handler.NewWidget(
		c.Logger(),
		verifier,
		identityResolver,
		access,
		enablements,
		c.WidgetRegistry(),
		widget.NewReader(widget.Deadline),
	), nil
}
