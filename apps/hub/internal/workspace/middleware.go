package workspace

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// PathParam names the segment of the path that holds the workspace.
const PathParam = "workspaceId"

// Middleware resolves the workspace of the path, checks the membership of the
// acting user in it, and puts the acting workspace into the context. It runs
// behind auth.Middleware, which puts the acting user into the context.
//
// It reads the membership on every request and holds no cache, so a member that a
// removal takes out reaches nothing at the next request. A workspace of no
// membership answers exactly as one that does not exist, so the answer tells the
// caller nothing about a workspace of another person.
func Middleware(
	logger *slog.Logger,
	members repository.WorkspaceMemberRepository,
) func(http.Handler) http.Handler {
	logger = logger.With(
		slog.String("component", "middleware"),
		slog.String("middleware", "workspace"),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			workspaceID, err := uuid.Parse(chi.URLParam(r, PathParam))
			if err != nil {
				problem.Write(w, r, problem.NewNotFound())

				return
			}

			_, err = members.Get(ctx, workspaceID.String(), pluginapi.ActingUserFromContext(ctx))
			if err != nil {
				if !errors.Is(err, errs.ErrMemberNotFound) {
					logger.ErrorContext(
						ctx,
						"reading the membership failed",
						slog.Any("error", err),
					)
					problem.Write(w, r, problem.NewInternal())

					return
				}

				problem.Write(w, r, problem.NewNotFound())

				return
			}

			ctx = pluginapi.ContextWithActingWorkspace(ctx, workspaceID.String())

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
