package registrar

import (
	"fmt"
	"log/slog"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// HandlerRegistrar is responsible for registering plugin HTTP routes as handlers.
type HandlerRegistrar struct {
	logger   *slog.Logger
	cfg      *config.Config
	jwtSvc   *auth.JWTService
	registry *handler.Registry
}

var _ Registrar = (*HandlerRegistrar)(nil)

// NewHandlerRegistrar creates a new HandlerRegistrar.
func NewHandlerRegistrar(
	logger *slog.Logger,
	cfg *config.Config,
	jwtSvc *auth.JWTService,
	reg *handler.Registry,
) *HandlerRegistrar {
	return &HandlerRegistrar{
		logger:   logger,
		cfg:      cfg,
		jwtSvc:   jwtSvc,
		registry: reg,
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
		r.cfg,
		r.jwtSvc,
		id,
		routes,
	)

	err = r.registry.Register(id.String(), pluginHandler)
	if err != nil {
		return fmt.Errorf("registering handler for plugin %s: %w", id, err)
	}

	return nil
}
