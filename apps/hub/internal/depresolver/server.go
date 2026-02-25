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
func (c *Container) HTTPRouter() *chi.Mux {
	c.httpRouter.once.Do(func() {
		c.httpRouter.instance = server.NewHTTPRouter()

		handler.RegisterHandlers(c.httpRouter.instance, c.getHTTPHandlers()...)
	})

	return c.httpRouter.instance
}

// HTTPServer initializes and returns the HTTP server.
func (c *Container) HTTPServer() (*http.Server, error) {
	c.httpServer.mu.Lock()
	defer c.httpServer.mu.Unlock()

	var err error

	c.httpServer.once.Do(func() {
		c.httpServer.instance, err = server.NewHTTP(c.Config(), c.HTTPRouter())
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

func (c *Container) getHTTPHandlers() []handler.Handler {
	return []handler.Handler{
		handler.NewPing(c.Logger()),
	}
}
