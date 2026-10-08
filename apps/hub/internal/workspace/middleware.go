package workspace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// PathParam names the segment of the path that holds the workspace.
const PathParam = "workspaceId"

// Middleware resolves the workspace of the path, checks the membership of the
// acting user in it, and puts the acting workspace and the role into the context. It runs
// behind auth.Middleware, which puts the acting user into the context.
//
// It reads the membership on every request and holds no cache, so a member that a
// removal takes out reaches nothing at the next request. A workspace of no
// membership answers exactly as one that does not exist, so the answer tells the
// caller nothing about a workspace of another person.
func Middleware(
	logger *slog.Logger,
	db *sqlx.DB,
	options ...MiddlewareOption,
) func(http.Handler) http.Handler {
	logger = logger.With(
		slog.String("component", "middleware"),
		slog.String("middleware", "workspace"),
	)

	var settings middlewareSettings
	for _, option := range options {
		option(&settings)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			workspaceID, err := uuid.Parse(chi.URLParam(r, PathParam))
			if err != nil {
				problem.Write(w, r, problem.NewNotFound())

				return
			}

			role, found, err := settings.roleIn(ctx, db, workspaceID.String())
			if err != nil {
				logger.ErrorContext(ctx, "reading the membership failed", slog.Any("error", err))
				problem.Write(w, r, problem.NewInternal())

				return
			}

			if !found {
				problem.Write(w, r, problem.NewNotFound())

				return
			}

			ctx = pluginapi.ContextWithActingWorkspace(ctx, workspaceID.String())
			ctx = ContextWithRole(ctx, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// MiddlewareOption changes what Middleware admits.
type MiddlewareOption func(*middlewareSettings)

type middlewareSettings struct {
	admitAdministrator bool
}

// AdmitAdministrator lets an administrator who is no member of the workspace pass as a
// manager. A route of the members, of the enablements, and the read of the workspace
// take it. A route of a plugin and of its settings never do, so an administrator reads
// no record of a workspace that they are no member of.
func AdmitAdministrator() MiddlewareOption {
	return func(settings *middlewareSettings) {
		settings.admitAdministrator = true
	}
}

// roleIn answers the role of the acting user in the workspace, and whether the request
// reaches the workspace at all.
func (settings middlewareSettings) roleIn(
	ctx context.Context,
	db *sqlx.DB,
	workspaceID string,
) (pluginapi.Role, bool, error) {
	var (
		role  pluginapi.Role
		found bool
	)

	err := database.WithTx(ctx, db, func(tx *sqlx.Tx) error {
		var err error

		role, found, err = settings.roleInTx(ctx, tx, workspaceID)

		return err
	})
	if err != nil {
		return "", false, fmt.Errorf("reading the role in the workspace: %w", err)
	}

	return role, found, nil
}

func (settings middlewareSettings) roleInTx(
	ctx context.Context,
	tx *sqlx.Tx,
	workspaceID string,
) (pluginapi.Role, bool, error) {
	member, err := repository.NewWorkspaceMember(tx).Get(
		ctx, workspaceID, pluginapi.ActingUserFromContext(ctx),
	)
	if err == nil {
		return member.Role, true, nil
	}

	if !errors.Is(err, errs.ErrMemberNotFound) {
		return "", false, fmt.Errorf("reading the membership: %w", err)
	}

	if !settings.admitAdministrator || !auth.IsAdministratorFromContext(ctx) {
		return "", false, nil
	}

	_, err = repository.NewWorkspace(tx).GetByID(ctx, workspaceID)
	if errors.Is(err, errs.ErrWorkspaceNotFound) {
		return "", false, nil
	}

	if err != nil {
		return "", false, fmt.Errorf("reading the workspace: %w", err)
	}

	return pluginapi.RoleManager, true, nil
}
