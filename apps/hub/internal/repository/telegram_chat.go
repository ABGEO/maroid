package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/model"
)

const telegramChatColumns = `id, chat_id, selected_workspace_id, created_at, updated_at`

// TelegramChatRepository defines the data access contract for the chats of the
// acting user.
type TelegramChatRepository interface {
	Get(ctx context.Context, chatID int64) (*model.TelegramChat, error)
	Select(ctx context.Context, chatID int64, workspaceID *string) error
}

// TelegramChat is a SQL based implementation of TelegramChatRepository.
type TelegramChat struct {
	tx *sqlx.Tx
}

var _ TelegramChatRepository = (*TelegramChat)(nil)

// NewTelegramChat creates a new TelegramChat repository instance.
func NewTelegramChat(tx *sqlx.Tx) *TelegramChat {
	return &TelegramChat{tx: tx}
}

// Get retrieves the row of the chat for the acting user. It returns a nil entity
// when no row exists.
func (r *TelegramChat) Get(ctx context.Context, chatID int64) (*model.TelegramChat, error) {
	var entity model.TelegramChat

	query := `SELECT ` + telegramChatColumns + ` FROM public.telegram_chats WHERE chat_id = $1;`

	if err := r.tx.GetContext(ctx, &entity, query, chatID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // no row is a chat with no state, not a failure.
		}

		return nil, fmt.Errorf("getting TelegramChat by chat ID: %w", err)
	}

	return &entity, nil
}

// Select stores the selected workspace of the chat for the acting user. A nil
// workspace clears the selection and keeps the row.
func (r *TelegramChat) Select(ctx context.Context, chatID int64, workspaceID *string) error {
	query := `
		INSERT INTO public.telegram_chats (chat_id, selected_workspace_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, chat_id) DO UPDATE
		SET selected_workspace_id = EXCLUDED.selected_workspace_id;`

	if _, err := r.tx.ExecContext(ctx, query, chatID, workspaceID); err != nil {
		return fmt.Errorf("selecting the workspace of TelegramChat: %w", err)
	}

	return nil
}
