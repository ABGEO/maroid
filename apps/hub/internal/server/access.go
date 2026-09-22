package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/httplog/v3"
)

// accessLog writes one record for each request that the hub answers.
func accessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	logger = logger.With(
		slog.String("component", "middleware"),
		slog.String("middleware", "access"),
	)

	return httplog.RequestLogger(logger, &httplog.Options{
		Schema: httplog.SchemaOTEL,

		// The recoverer of the hub answers a panic, because this one writes a
		// bare 500 and every failure of the hub carries a problem.
		RecoverPanics: false,
	})
}
