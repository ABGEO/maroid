package registrar

import (
	"fmt"
	"log/slog"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// HandlerRegistrar is responsible for registering plugin HTTP routes as handlers.
type HandlerRegistrar struct {
	logger       *slog.Logger
	verifier     auth.TokenVerifier
	resolver     auth.IdentityResolver
	registry     *handler.Registry
	capabilities *registry.CapabilityRegistry
}

var _ Registrar = (*HandlerRegistrar)(nil)

// NewHandlerRegistrar creates a new HandlerRegistrar.
func NewHandlerRegistrar(
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	reg *handler.Registry,
	capabilities *registry.CapabilityRegistry,
) *HandlerRegistrar {
	return &HandlerRegistrar{
		logger:       logger,
		verifier:     verifier,
		resolver:     resolver,
		registry:     reg,
		capabilities: capabilities,
	}
}

// Name returns the name of the registrar.
func (r *HandlerRegistrar) Name() string {
	return "handler"
}

// Supports indicates whether the registrar can handle the given plugin.
func (r *HandlerRegistrar) Supports(plugin pluginapi.Plugin) bool {
	_, ok := plugin.(pluginapi.RoutePlugin)

	return ok
}

// Register handles the registration of plugin HTTP routes.
func (r *HandlerRegistrar) Register(plugin pluginapi.Plugin) error {
	id := plugin.Meta().ID

	routePlugin, ok := plugin.(pluginapi.RoutePlugin)
	if !ok {
		return fmt.Errorf(
			"plugin %s does not support Route capability: %w",
			id,
			errs.ErrPluginCapabilityNotSupported,
		)
	}

	routes, err := routePlugin.Routes()
	if err != nil {
		return fmt.Errorf("retrieving routes for plugin %s: %w", id, err)
	}

	pluginHandler := handler.NewPluginWrapper(
		r.logger,
		r.verifier,
		r.resolver,
		id,
		routes,
	)

	err = r.registry.Register(id.String(), pluginHandler)
	if err != nil {
		return fmt.Errorf("registering handler for plugin %s: %w", id, err)
	}

	items := make([]registry.APIRoute, 0, len(routes))
	for _, route := range routes {
		items = append(items, registry.APIRoute{
			Method: route.Method,
			Path:   "/plugins/" + id.String() + "/api" + route.Pattern,
		})
	}

	r.capabilities.Record(id, registry.CapAPI, items)

	return nil
}
