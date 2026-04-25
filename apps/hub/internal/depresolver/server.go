package depresolver

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/abgeo/maroid/apps/hub/internal/handler"
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
		c.httpServer.once = sync.Once{}

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
	cfg := c.Config()
	logger := c.Logger()
	pluginRegistry := c.PluginRegistry()
	uiRegistry := c.UIRegistry()

	jwtSvc, err := c.JWTService()
	if err != nil {
		return err
	}

	oidcFlow, err := c.OIDCFlow()
	if err != nil {
		return err
	}

	authHandler := handler.NewAuth(cfg, logger, jwtSvc, oidcFlow)
	pluginHandler := handler.NewPlugin(cfg, logger, jwtSvc, pluginRegistry, uiRegistry)

	err = reg.Register("auth", authHandler)
	if err != nil {
		return fmt.Errorf("register auth handler: %w", err)
	}

	err = reg.Register("ping", handler.NewPing(logger))
	if err != nil {
		return fmt.Errorf("register ping handler: %w", err)
	}

	err = reg.Register("plugin", pluginHandler)
	if err != nil {
		return fmt.Errorf("register plugin handler: %w", err)
	}

	return nil
}
