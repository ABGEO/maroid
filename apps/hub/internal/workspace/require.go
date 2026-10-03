package workspace

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/problem"
)

type roleKey struct{}

// ContextWithRole returns a context that carries the role of the acting user in the
// acting workspace.
func ContextWithRole(ctx context.Context, role pluginapi.Role) context.Context {
	return context.WithValue(ctx, roleKey{}, role)
}

// RoleFromContext returns the role that the context carries, or the empty role,
// which holds no permission.
func RoleFromContext(ctx context.Context) pluginapi.Role {
	role, _ := ctx.Value(roleKey{}).(pluginapi.Role)

	return role
}

// Require answers permission-denied unless the role of the acting user reaches the
// permission. It runs behind Middleware, which puts the role into the context.
func Require(
	logger *slog.Logger,
	authorizer authz.Authorizer,
	permission string,
) func(http.Handler) http.Handler {
	return RequireOf(logger, authorizer, func(*http.Request) string { return permission })
}

// RequireOf checks the permission that pick names for each request, for a route whose
// permission depends on its target.
func RequireOf(
	logger *slog.Logger,
	authorizer authz.Authorizer,
	pick func(*http.Request) string,
) func(http.Handler) http.Handler {
	logger = logger.With(
		slog.String("component", "middleware"),
		slog.String("middleware", "require"),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			permission := pick(r)

			allowed, lowest, err := authorizer.Allowed(RoleFromContext(r.Context()), permission)
			if err != nil {
				logger.ErrorContext(r.Context(), "checking the permission failed",
					slog.String("permission", permission), slog.Any("error", err))
				problem.Write(w, r, problem.NewInternal())

				return
			}

			if !allowed {
				problem.Write(w, r, problem.NewPermissionDenied(permission, string(lowest)))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
