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
	db         *sqlx.DB
	workspaces repository.WorkspaceRepository
	members    repository.WorkspaceMemberRepository
}

// NewChatSelection creates a new ChatSelection.
func NewChatSelection(
	db *sqlx.DB,
	workspaces repository.WorkspaceRepository,
	members repository.WorkspaceMemberRepository,
) *ChatSelection {
	return &ChatSelection{db: db, workspaces: workspaces, members: members}
}

// Acting returns the workspace that the chat acts in for the acting user, or the
// empty string when the person must pick one. A selection of a workspace that the
// person left is cleared, and the only membership of a person becomes the selection.
func (c *ChatSelection) Acting(ctx context.Context, chatID int64) (string, error) {
	userID := pluginapi.ActingUserFromContext(ctx)

	var acting string

	err := database.WithScopeTx(ctx, c.db, func(tx *sqlx.Tx) error {
		chats := repository.NewTelegramChat(tx)

		chat, err := chats.Get(ctx, chatID)
		if err != nil {
			return fmt.Errorf("reading the chat: %w", err)
		}

		if chat != nil && chat.SelectedWorkspaceID != nil {
			selected := *chat.SelectedWorkspaceID

			_, err = c.members.Get(ctx, selected, userID)
			if err == nil {
				acting = selected

				return nil
			}

			if !errors.Is(err, errs.ErrMemberNotFound) {
				return fmt.Errorf("checking the membership of the selection: %w", err)
			}

			if err = chats.Select(ctx, chatID, nil); err != nil {
				return fmt.Errorf("clearing the selection: %w", err)
			}
		}

		acting, err = c.selectTheOnly(ctx, chats, chatID, userID)

		return err
	})
	if err != nil {
		return "", fmt.Errorf("resolving the workspace of the chat: %w", err)
	}

	return acting, nil
}

// Select stores the workspace as the selection of the chat, when the acting user is
// a member of it, and returns it.
func (c *ChatSelection) Select(
	ctx context.Context,
	chatID int64,
	workspaceID string,
) (*model.Workspace, error) {
	if _, err := c.members.Get(ctx, workspaceID, pluginapi.ActingUserFromContext(ctx)); err != nil {
		return nil, fmt.Errorf("checking the membership of the pick: %w", err)
	}

	selected, err := c.workspaces.GetByID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("reading the picked workspace: %w", err)
	}

	err = database.WithScopeTx(ctx, c.db, func(tx *sqlx.Tx) error {
		return repository.NewTelegramChat(tx).Select(ctx, chatID, &workspaceID)
	})
	if err != nil {
		return nil, fmt.Errorf("storing the pick: %w", err)
	}

	return selected, nil
}

// Choices lists the workspaces that the acting user can pick.
func (c *ChatSelection) Choices(ctx context.Context) ([]model.Workspace, error) {
	choices, err := c.workspaces.ListOfUser(ctx, pluginapi.ActingUserFromContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("listing the workspaces of the person: %w", err)
	}

	return choices, nil
}

// selectTheOnly stores the only membership of the person as the selection of the
// chat, and returns it. A person of two or more workspaces picks one.
func (c *ChatSelection) selectTheOnly(
	ctx context.Context,
	chats repository.TelegramChatRepository,
	chatID int64,
	userID string,
) (string, error) {
	memberships, err := c.workspaces.ListOfUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("listing the workspaces of the person: %w", err)
	}

	if len(memberships) != 1 {
		return "", nil
	}

	only := memberships[0].ID

	if err = chats.Select(ctx, chatID, &only); err != nil {
		return "", fmt.Errorf("selecting the only workspace: %w", err)
	}

	return only, nil
}
