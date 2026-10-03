package administration

import (
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// Permission names what a route of the administration needs, in the answer that
// refuses a person who is no administrator.
const Permission = "administration"

// Require answers permission-denied, with the permission administration, to a person
// who is no administrator. It runs behind auth.Middleware, which puts the mark into
// the context.
func Require(logger *slog.Logger) func(http.Handler) http.Handler {
	logger = logger.With(
		slog.String("component", "middleware"),
		slog.String("middleware", "administration"),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !auth.IsAdministratorFromContext(r.Context()) {
				logger.InfoContext(r.Context(), "a route of the administration refused a person")
				problem.Write(w, r, problem.NewPermissionDenied(Permission, ""))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
