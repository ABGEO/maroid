package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/httplog/v3"

	"github.com/abgeo/maroid/apps/hub/internal/healthcheck"
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

		Skip: isSuccessfulHealthProbe,
	})
}

// isSuccessfulHealthProbe reports a health probe that answered 200. An
// orchestrator probes every few seconds, and the record of a success says nothing.
func isSuccessfulHealthProbe(r *http.Request, status int) bool {
	if status != http.StatusOK {
		return false
	}

	return r.URL.Path == healthcheck.LivenessPath || r.URL.Path == healthcheck.ReadinessPath
}
