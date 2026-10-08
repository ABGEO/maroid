package workspace

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// ChatSelection keeps the workspace that each chat of the acting user acts in.
type ChatSelection struct {
	db *sqlx.DB
}

// NewChatSelection creates a new ChatSelection.
func NewChatSelection(db *sqlx.DB) *ChatSelection {
	return &ChatSelection{db: db}
}

// Acting returns the workspace that the chat acts in for the acting user and the role
// of the user in it, or the empty string when the person must pick one. A selection
// of a workspace that the person left is cleared, and the only membership of a person
// becomes the selection.
func (c *ChatSelection) Acting(ctx context.Context, chatID int64) (string, pluginapi.Role, error) {
	userID := pluginapi.ActingUserFromContext(ctx)

	var (
		acting string
		role   pluginapi.Role
	)

	err := database.WithScopeTx(ctx, c.db, func(tx *sqlx.Tx) error {
		chats := repository.NewTelegramChat(tx)

		chat, err := chats.Get(ctx, chatID)
		if err != nil {
			return fmt.Errorf("reading the chat: %w", err)
		}

		if chat != nil && chat.SelectedWorkspaceID != nil {
			selected := *chat.SelectedWorkspaceID

			member, memberErr := repository.NewWorkspaceMember(tx).Get(ctx, selected, userID)
			if memberErr == nil {
				acting, role = selected, member.Role

				return nil
			}

			if !errors.Is(memberErr, errs.ErrMemberNotFound) {
				return fmt.Errorf("checking the membership of the selection: %w", memberErr)
			}

			if err = chats.Select(ctx, chatID, nil); err != nil {
				return fmt.Errorf("clearing the selection: %w", err)
			}
		}

		acting, role, err = selectTheOnly(ctx, tx, chatID, userID)

		return err
	})
	if err != nil {
		return "", "", fmt.Errorf("resolving the workspace of the chat: %w", err)
	}

	return acting, role, nil
}

// Select stores the workspace as the selection of the chat, when the acting user is
// a member of it, and returns it.
func (c *ChatSelection) Select(
	ctx context.Context,
	chatID int64,
	workspaceID string,
) (*model.Workspace, error) {
	var selected *model.Workspace

	err := database.WithScopeTx(ctx, c.db, func(tx *sqlx.Tx) error {
		_, err := repository.NewWorkspaceMember(tx).Get(
			ctx, workspaceID, pluginapi.ActingUserFromContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("checking the membership of the pick: %w", err)
		}

		if selected, err = repository.NewWorkspace(tx).GetByID(ctx, workspaceID); err != nil {
			return fmt.Errorf("reading the picked workspace: %w", err)
		}

		if err = repository.NewTelegramChat(tx).Select(ctx, chatID, &workspaceID); err != nil {
			return fmt.Errorf("storing the pick: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("selecting the workspace of the chat: %w", err)
	}

	return selected, nil
}

// Choices lists the workspaces that the acting user can pick.
func (c *ChatSelection) Choices(ctx context.Context) ([]model.Workspace, error) {
	choices, err := database.FetchTx(ctx, c.db, func(tx *sqlx.Tx) ([]model.Workspace, error) {
		return repository.NewWorkspace(tx).ListOfUser(ctx, pluginapi.ActingUserFromContext(ctx))
	})
	if err != nil {
		return nil, fmt.Errorf("listing the workspaces of the person: %w", err)
	}

	return choices, nil
}

// selectTheOnly stores the only membership of the person as the selection of the
// chat, and returns it. A person of two or more workspaces picks one.
func selectTheOnly(
	ctx context.Context,
	tx *sqlx.Tx,
	chatID int64,
	userID string,
) (string, pluginapi.Role, error) {
	memberships, err := repository.NewWorkspace(tx).ListOfUser(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("listing the workspaces of the person: %w", err)
	}

	if len(memberships) != 1 {
		return "", "", nil
	}

	only := memberships[0].ID

	if err = repository.NewTelegramChat(tx).Select(ctx, chatID, &only); err != nil {
		return "", "", fmt.Errorf("selecting the only workspace: %w", err)
	}

	return only, memberships[0].Role, nil
}
