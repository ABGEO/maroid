package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

// NewHTTPRouter creates a new HTTP router with middleware.
func NewHTTPRouter() *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(render.SetContentType(render.ContentTypeJSON))

	// @todo: setup CORS
	// @todo: setup logging

	return router
}

// NewHTTP creates a new HTTP server with the given configuration and router.
func NewHTTP(cfg *config.Config, router chi.Router) (*http.Server, error) {
	return &http.Server{
		Addr:              cfg.Server.Address(),
		Handler:           router,
		ReadTimeout:       cfg.Server.ReadTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}, nil
}
