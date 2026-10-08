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
	"github.com/abgeo/maroid/apps/hub/internal/server"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/libs/rest/idempotency"
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
		router, routerErr := server.NewHTTPRouter(c.Config(), c.Logger())
		if routerErr != nil {
			err = routerErr

			return
		}

		c.httpRouter.instance = router

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

	workspaceHandler, err := c.buildWorkspaceHandler()
	if err != nil {
		return err
	}

	if err = reg.Register("workspace", workspaceHandler); err != nil {
		return fmt.Errorf("register workspace handler: %w", err)
	}

	userHandler, err := c.buildUserHandler()
	if err != nil {
		return err
	}

	if err = reg.Register("user", userHandler); err != nil {
		return fmt.Errorf("register user handler: %w", err)
	}

	providerHandler, err := c.buildProviderHandler()
	if err != nil {
		return err
	}

	if err = reg.Register("provider", providerHandler); err != nil {
		return fmt.Errorf("register provider handler: %w", err)
	}

	return nil
}

// buildProviderHandler resolves every dependency of the handler of /providers.
func (c *Container) buildProviderHandler() (*handler.Provider, error) {
	verifier, err := c.TokenVerifier()
	if err != nil {
		return nil, err
	}

	identityResolver, err := c.IdentityResolver()
	if err != nil {
		return nil, err
	}

	idempotencyStore, err := c.IdempotencyStore()
	if err != nil {
		return nil, err
	}

	providers, err := c.ProviderService()
	if err != nil {
		return nil, err
	}

	return handler.NewProvider(
		c.Logger(), verifier, identityResolver, idempotencyStore, providers,
	), nil
}

// buildUserHandler resolves every dependency of the handler of /users.
func (c *Container) buildUserHandler() (*handler.User, error) {
	verifier, err := c.TokenVerifier()
	if err != nil {
		return nil, err
	}

	identityResolver, err := c.IdentityResolver()
	if err != nil {
		return nil, err
	}

	idempotencyStore, err := c.IdempotencyStore()
	if err != nil {
		return nil, err
	}

	service, err := c.UserService()
	if err != nil {
		return nil, err
	}

	dbInstance, err := c.Database()
	if err != nil {
		return nil, err
	}

	accounts, err := c.LocalAccounts()
	if err != nil {
		return nil, err
	}

	return handler.NewUser(
		c.Logger(), verifier, identityResolver, idempotencyStore, service, c.Config().Auth.DeckURL,
		dbInstance, accounts,
	), nil
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

	authHandler, err := c.buildAuthHandler(cfg, logger, verifier)
	if err != nil {
		return nil, err
	}

	mcpHandler, err := c.buildMCPHandler(cfg, logger, identityResolver)
	if err != nil {
		return nil, err
	}

	idempotencyStore, err := c.IdempotencyStore()
	if err != nil {
		return nil, err
	}

	healthService, err := c.HealthService()
	if err != nil {
		return nil, err
	}

	pluginHandler, err := c.buildPluginHandler(
		logger, verifier, identityResolver, settingsSvc, idempotencyStore,
	)
	if err != nil {
		return nil, err
	}

	return map[string]handler.Handler{
		"health": handler.NewHealth(logger, healthService),
		"auth":   authHandler,
		"plugin": pluginHandler,
		"mcp":    mcpHandler,
	}, nil
}

// buildPluginHandler resolves the member repository of the plugin handler.
func (c *Container) buildPluginHandler(
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	identityResolver auth.IdentityResolver,
	settingsSvc settings.Service,
	idempotencyStore idempotency.Store,
) (*handler.Plugin, error) {
	access, err := c.workspaceAccess()
	if err != nil {
		return nil, err
	}

	dbInstance, err := c.Database()
	if err != nil {
		return nil, err
	}

	return handler.NewPlugin(
		logger,
		verifier,
		identityResolver,
		c.PluginCatalog(),
		c.UIRegistry(),
		settingsSvc,
		idempotencyStore,
		access,
		dbInstance,
	), nil
}

// buildWorkspaceHandler resolves every dependency of the handler of the workspaces.
func (c *Container) buildWorkspaceHandler() (*handler.Workspace, error) {
	verifier, err := c.TokenVerifier()
	if err != nil {
		return nil, err
	}

	identityResolver, err := c.IdentityResolver()
	if err != nil {
		return nil, err
	}

	idempotencyStore, err := c.IdempotencyStore()
	if err != nil {
		return nil, err
	}

	service, err := c.WorkspaceService()
	if err != nil {
		return nil, err
	}

	authorizer, err := c.Authorizer()
	if err != nil {
		return nil, err
	}

	enablements, err := c.EnablementService()
	if err != nil {
		return nil, err
	}

	dbInstance, err := c.Database()
	if err != nil {
		return nil, err
	}

	return handler.NewWorkspace(
		c.Logger(),
		verifier,
		identityResolver,
		idempotencyStore,
		dbInstance,
		service,
		authorizer,
		service,
		handler.WorkspacePlugins{Enablements: enablements, Catalog: c.PluginCatalog()},
	), nil
}

// buildAuthHandler resolves every dependency of the auth handler.
func (c *Container) buildAuthHandler(
	cfg *config.Config,
	logger *slog.Logger,
	verifier auth.TokenVerifier,
) (*handler.Auth, error) {
	oidcFlow, err := c.OIDCFlow()
	if err != nil {
		return nil, err
	}

	dbInstance, err := c.Database()
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

	providers, err := c.ProviderService()
	if err != nil {
		return nil, err
	}

	accounts, err := c.LocalAccounts()
	if err != nil {
		return nil, err
	}

	return handler.NewAuth(
		cfg,
		logger,
		verifier,
		oidcFlow,
		dbInstance,
		identityResolver,
		authSvc,
		providers,
		accounts,
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

	access, err := c.workspaceAccess()
	if err != nil {
		return nil, err
	}

	return handler.NewMCP(
		cfg, logger, oidcSvc, identityResolver, toolRegistry,
		access.DB, access.Enablements, access.Authorizer,
	), nil
}
