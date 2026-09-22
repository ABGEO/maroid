package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	hubmiddleware "github.com/abgeo/maroid/apps/hub/internal/middleware"
	"github.com/abgeo/maroid/libs/problem"
)

// NewHTTPRouter creates a new HTTP router with middleware.
func NewHTTPRouter(cfg *config.Config, logger *slog.Logger) (*chi.Mux, error) {
	resolveClientIP, err := clientIP(cfg.Server.TrustedProxies)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	router.Use(problem.RequestID)
	router.Use(resolveClientIP)
	router.Use(accessLog(logger))
	router.Use(recoverer(logger))
	router.Use(middleware.StripSlashes)
	router.Use(render.SetContentType(render.ContentTypeJSON))

	if cfg.CORS.Enabled {
		router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.CORS.AllowOrigins,
			AllowedMethods:   cfg.CORS.AllowMethods,
			AllowedHeaders:   cfg.CORS.AllowHeaders,
			ExposedHeaders:   cfg.CORS.ExposeHeaders,
			AllowCredentials: cfg.CORS.AllowCredentials,
			MaxAge:           int(cfg.CORS.MaxAge.Seconds()),
		}))
	}

	router.NotFound(notFound())
	router.MethodNotAllowed(methodNotAllowed(router))

	return router, nil
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

// clientIP resolves the address of the caller into the context, and never from
// a header that the caller chose.
func clientIP(trustedProxies []string) (func(http.Handler) http.Handler, error) {
	if len(trustedProxies) == 0 {
		return middleware.ClientIPFromRemoteAddr, nil
	}

	// ClientIPFromXFF panics on a prefix it cannot read, so this reads them first.
	if _, err := hubmiddleware.ParsePrefixes(trustedProxies); err != nil {
		return nil, fmt.Errorf("reading the trusted proxies: %w", err)
	}

	return middleware.ClientIPFromXFF(trustedProxies...), nil
}
