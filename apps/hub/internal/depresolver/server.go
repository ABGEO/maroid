package depresolver

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/server"
)

// HandlerRegistry initializes and returns the handler registry.
func (c *Container) HandlerRegistry() (*handler.Registry, error) {
	c.handlerRegistry.mu.Lock()
	defer c.handlerRegistry.mu.Unlock()

	var err error

	c.handlerRegistry.once.Do(func() {
		c.handlerRegistry.instance = handler.NewRegistry()

		regErr := c.registerHandlers(c.handlerRegistry.instance)
		if regErr != nil {
			err = regErr

			return
		}
	})

	if err != nil {
		c.handlerRegistry.once = sync.Once{}

		return nil, fmt.Errorf("initializing handler registry: %w", err)
	}

	return c.handlerRegistry.instance, nil
}

// HTTPRouter initializes and returns the HTTP router.
func (c *Container) HTTPRouter() (*chi.Mux, error) {
	c.httpRouter.mu.Lock()
	defer c.httpRouter.mu.Unlock()

	var err error

	c.httpRouter.once.Do(func() {
		c.httpRouter.instance = server.NewHTTPRouter(c.Config())

		handlerRegistry, handlerRegistryErr := c.HandlerRegistry()
		if handlerRegistryErr != nil {
			err = handlerRegistryErr

			return
		}

		handler.RegisterHandlers(
			c.httpRouter.instance,
			handlerRegistry.All()...,
		)
	})

	if err != nil {
		c.httpRouter.once = sync.Once{}

		return nil, fmt.Errorf("initializing HTTP router: %w", err)
	}

	return c.httpRouter.instance, nil
}

// HTTPServer initializes and returns the HTTP server.
func (c *Container) HTTPServer() (*http.Server, error) {
	c.httpServer.mu.Lock()
	defer c.httpServer.mu.Unlock()

	var err error

	c.httpServer.once.Do(func() {
		router, routerErr := c.HTTPRouter()
		if routerErr != nil {
			err = routerErr

			return
		}

		c.httpServer.instance, err = server.NewHTTP(c.Config(), router)
	})

	if err != nil {
		c.httpServer.once = sync.Once{}

		return nil, fmt.Errorf("initializing HTTP server: %w", err)
	}

	return c.httpServer.instance, nil
}

// CloseHTTPServer immediately closes the HTTP server.
func (c *Container) CloseHTTPServer() error {
	if c.httpServer.instance == nil {
		return nil
	}

	err := c.httpServer.instance.Close()
	if err != nil {
		return fmt.Errorf("closing HTTP Server: %w", err)
	}

	return nil
}

func (c *Container) registerHandlers(reg *handler.Registry) error {
	handlers, err := c.buildHandlers()
	if err != nil {
		return err
	}

	for id, one := range handlers {
		if err = reg.Register(id, one); err != nil {
			return fmt.Errorf("register %s handler: %w", id, err)
		}
	}

	return nil
}

// buildHandlers resolves every handler of the HTTP API, keyed by its identifier.
func (c *Container) buildHandlers() (map[string]handler.Handler, error) {
	cfg := c.Config()
	logger := c.Logger()

	verifier, err := c.TokenVerifier()
	if err != nil {
		return nil, err
	}

	settingsSvc, err := c.SettingsService()
	if err != nil {
		return nil, err
	}

	identityResolver, err := c.IdentityResolver()
	if err != nil {
		return nil, err
	}

	userRepo, err := c.UserRepository()
	if err != nil {
		return nil, err
	}

	authHandler, err := c.buildAuthHandler(cfg, logger, verifier, userRepo)
	if err != nil {
		return nil, err
	}

	mcpHandler, err := c.buildMCPHandler(cfg, logger, identityResolver)
	if err != nil {
		return nil, err
	}

	return map[string]handler.Handler{
		"auth": authHandler,
		"ping": handler.NewPing(logger),
		"plugin": handler.NewPlugin(
			logger,
			verifier,
			identityResolver,
			c.PluginRegistry(),
			c.UIRegistry(),
			c.CapabilityRegistry(),
			settingsSvc,
		),
		"mcp": mcpHandler,
	}, nil
}

// buildAuthHandler resolves every dependency of the auth handler.
func (c *Container) buildAuthHandler(
	cfg *config.Config,
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	userRepo repository.UserRepository,
) (*handler.Auth, error) {
	oidcFlow, err := c.OIDCFlow()
	if err != nil {
		return nil, err
	}

	identityRepo, err := c.IdentityRepository()
	if err != nil {
		return nil, err
	}

	invitationRepo, err := c.InvitationRepository()
	if err != nil {
		return nil, err
	}

	authSvc, err := c.AuthService()
	if err != nil {
		return nil, err
	}

	identityResolver, err := c.IdentityResolver()
	if err != nil {
		return nil, err
	}

	return handler.NewAuth(
		cfg,
		logger,
		verifier,
		oidcFlow,
		userRepo,
		identityRepo,
		identityResolver,
		invitationRepo,
		authSvc,
	), nil
}

// buildMCPHandler resolves every dependency of the Model Context Protocol handler.
func (c *Container) buildMCPHandler(
	cfg *config.Config,
	logger *slog.Logger,
	identityResolver auth.IdentityResolver,
) (*handler.MCP, error) {
	oidcSvc, err := c.OIDCService()
	if err != nil {
		return nil, err
	}

	toolRegistry, err := c.MCPToolRegistry()
	if err != nil {
		return nil, err
	}

	return handler.NewMCP(cfg, logger, oidcSvc, identityResolver, toolRegistry), nil
}
