package workspace

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// FilePurger queues the purge of every file of the acting workspace through a
// transaction, and answers the count of the purges.
type FilePurger interface {
	PurgeWorkspace(ctx context.Context, tx *sqlx.Tx) (int, error)
}

// Remover removes the acting workspace with every record it holds.
type Remover struct {
	logger *slog.Logger
	db     *sqlx.DB
	files  FilePurger
}

// NewRemover creates a Remover over the database and the purge of the files.
func NewRemover(logger *slog.Logger, db *sqlx.DB, files FilePurger) *Remover {
	return &Remover{
		logger: logger.With(slog.String("component", "workspace_remover")),
		db:     db,
		files:  files,
	}
}

// Remove deletes the acting workspace in one transaction. It answers
// errs.MembersRemainError when an acting user who is no administrator shares the
// workspace, and errs.ErrWorkspaceNotFound when the workspace is gone. The files and
// the key of the workspace go after the commit, through the outbox, so no call to an
// external store holds the transaction open.
func (r *Remover) Remove(ctx context.Context) error {
	workspaceID := pluginapi.ActingWorkspaceFromContext(ctx)
	actor := pluginapi.ActingUserFromContext(ctx)
	administrator := auth.IsAdministratorFromContext(ctx)

	var purges int

	err := database.WithScopeTx(ctx, r.db, func(tx *sqlx.Tx) error {
		workspaces := repository.NewWorkspace(tx)

		// The lock comes before the count, and an insert of a membership waits for it,
		// so the count and the delete see the same members.
		if err := workspaces.Lock(ctx, workspaceID); err != nil {
			return fmt.Errorf("locking the workspace: %w", err)
		}

		if !administrator {
			others, err := repository.NewWorkspaceMember(tx).CountOthers(ctx, workspaceID, actor)
			if err != nil {
				return fmt.Errorf("counting the other members: %w", err)
			}

			if others > 0 {
				return &errs.MembersRemainError{Others: others}
			}
		}

		var err error

		if purges, err = r.files.PurgeWorkspace(ctx, tx); err != nil {
			return fmt.Errorf("queuing the purge of the files: %w", err)
		}

		if err = repository.NewOutboxTask(tx).
			Enqueue(ctx, secret.DestroyTask(secret.WorkspaceKey(workspaceID))); err != nil {
			return fmt.Errorf("queuing the destruction of the key: %w", err)
		}

		if err = workspaces.Delete(ctx, workspaceID); err != nil {
			return fmt.Errorf("deleting the workspace: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("removing the acting workspace: %w", err)
	}

	r.logger.InfoContext(
		ctx,
		"removed a workspace",
		slog.String("workspace_id", workspaceID),
		slog.String("user_id", actor),
		slog.Bool("administrator", administrator),
		slog.Int("purges", purges),
	)

	return nil
}
