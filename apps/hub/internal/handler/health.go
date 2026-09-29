package handler

import (
	"log/slog"
	"maps"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/hellofresh/health-go/v5"

	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/healthcheck"
	"github.com/abgeo/maroid/libs/rest/problem"
)

const drainDetail = "The hub is shutting down."

// Health answers whether the hub runs and whether it can serve a request.
type Health struct {
	logger  *slog.Logger
	checker healthcheck.Checker
}

// NewHealth creates a new Health handler.
func NewHealth(logger *slog.Logger, checker healthcheck.Checker) *Health {
	return &Health{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "health"),
		),
		checker: checker,
	}
}

// Register registers the health routes.
func (h *Health) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	router.Get(healthcheck.LivenessPath, h.Liveness)
	router.Get(healthcheck.ReadinessPath, h.Readiness)
}

// Liveness answers whether the process runs.
func (h *Health) Liveness(w http.ResponseWriter, r *http.Request) {
	writeCheck(w, r, h.checker.Liveness(r.Context()))
}

// Readiness answers whether the hub can serve a request.
func (h *Health) Readiness(w http.ResponseWriter, r *http.Request) {
	if h.checker.Draining() {
		draining := problems.NewNotReady()
		draining.Detail = drainDetail

		problem.Write(w, r, draining)

		return
	}

	check := h.checker.Readiness(r.Context())
	if check.Status != health.StatusUnavailable {
		writeCheck(w, r, check)

		return
	}

	for dependency, failure := range check.Failures {
		h.logger.WarnContext(
			r.Context(),
			"dependency check failed",
			slog.String("dependency", dependency),
			slog.String("error", failure),
		)
	}

	failed := slices.Sorted(maps.Keys(check.Failures))
	problem.Write(w, r, problems.NewNotReady(failed...))
}

// writeCheck answers the measurement of the library, with the moment in UTC.
func writeCheck(w http.ResponseWriter, r *http.Request, check health.Check) {
	check.Timestamp = check.Timestamp.UTC()

	render.Status(r, http.StatusOK)
	render.JSON(w, r, check)
}
