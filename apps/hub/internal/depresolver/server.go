package depresolver

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/server"
)

// HTTPRouter initializes and returns the HTTP router.
func (c *Container) HTTPRouter() (*chi.Mux, error) {
	c.httpRouter.mu.Lock()
	defer c.httpRouter.mu.Unlock()

	var err error

	c.httpRouter.once.Do(func() {
		c.httpRouter.instance = server.NewHTTPRouter()

		handlers, handlerErr := c.getHTTPHandlers()
		if handlerErr != nil {
			err = handlerErr

			return
		}

		handler.RegisterHandlers(c.httpRouter.instance, handlers...)
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

func (c *Container) getHTTPHandlers() ([]handler.Handler, error) {
	jwtSvc, err := c.JWTService()
	if err != nil {
		return nil, err
	}

	oidcFlow, err := c.OIDCFlow()
	if err != nil {
		return nil, err
	}

	authHandler := handler.NewAuth(
		c.Config(),
		c.Logger(),
		jwtSvc,
		oidcFlow,
	)

	return []handler.Handler{
		authHandler,
		handler.NewPing(c.Logger()),
	}, nil
}
