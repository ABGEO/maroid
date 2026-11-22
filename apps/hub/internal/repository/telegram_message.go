package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/model"
)

// A row older than the lifetime is gone for every read, whether or not a purge
// deleted it yet.
const (
	telegramMessageColumns = `id, plugin_id, chat_id, message_id, claim_until,
		claim_until IS NOT NULL AND claim_until > now() AS claim_open,
		created_at, updated_at`
	telegramMessageAlive = `created_at > now() - INTERVAL '30 days'`
)

// TelegramMessageRepository defines the data access contract for the plugin messages
// of the acting user.
type TelegramMessageRepository interface {
	Insert(ctx context.Context, message *model.TelegramMessage) (string, error)
	ByTelegram(ctx context.Context, chatID, messageID int64) (*model.TelegramMessage, error)
	ByID(ctx context.Context, id, pluginID string) (*model.TelegramMessage, error)
	OpenClaims(ctx context.Context) ([]model.TelegramMessage, error)
	Release(ctx context.Context, id, pluginID string) (bool, error)
	Purge(ctx context.Context) error
}

// TelegramMessage is a SQL based implementation of TelegramMessageRepository.
type TelegramMessage struct {
	tx *sqlx.Tx
}

var _ TelegramMessageRepository = (*TelegramMessage)(nil)

// NewTelegramMessage creates a new TelegramMessage repository instance.
func NewTelegramMessage(tx *sqlx.Tx) *TelegramMessage {
	return &TelegramMessage{tx: tx}
}

// Insert records a plugin message of the acting user, and returns its identifier.
func (r *TelegramMessage) Insert(
	ctx context.Context,
	message *model.TelegramMessage,
) (string, error) {
	query := `
		INSERT INTO public.telegram_messages (plugin_id, chat_id, message_id, claim_until)
		VALUES ($1, $2, $3, $4)
		RETURNING id;`

	var id string

	err := r.tx.GetContext(
		ctx,
		&id,
		query,
		message.PluginID,
		message.ChatID,
		message.MessageID,
		message.ClaimUntil,
	)
	if err != nil {
		return "", fmt.Errorf("inserting TelegramMessage: %w", err)
	}

	return id, nil
}

// ByTelegram returns the plugin message that Telegram knows by the chat and the
// message identifier. It returns a nil entity when no row exists.
func (r *TelegramMessage) ByTelegram(
	ctx context.Context,
	chatID, messageID int64,
) (*model.TelegramMessage, error) {
	query := `SELECT ` + telegramMessageColumns + ` FROM public.telegram_messages
		WHERE chat_id = $1 AND message_id = $2 AND ` + telegramMessageAlive + `;`

	return r.getOne(ctx, "by Telegram", query, chatID, messageID)
}

// ByID returns the plugin message that the plugin sent. It returns a nil entity
// when no row exists, or when another plugin sent it.
func (r *TelegramMessage) ByID(
	ctx context.Context,
	id, pluginID string,
) (*model.TelegramMessage, error) {
	query := `SELECT ` + telegramMessageColumns + ` FROM public.telegram_messages
		WHERE id = $1 AND plugin_id = $2 AND ` + telegramMessageAlive + `;`

	return r.getOne(ctx, "by ID", query, id, pluginID)
}

// OpenClaims returns the plugin messages of the acting user whose reply claim holds,
// the oldest first.
func (r *TelegramMessage) OpenClaims(ctx context.Context) ([]model.TelegramMessage, error) {
	query := `SELECT ` + telegramMessageColumns + ` FROM public.telegram_messages
		WHERE claim_until > now() AND ` + telegramMessageAlive + `
		ORDER BY created_at, id;`

	var entities []model.TelegramMessage

	if err := r.tx.SelectContext(ctx, &entities, query); err != nil {
		return nil, fmt.Errorf("listing the open claims of TelegramMessage: %w", err)
	}

	return entities, nil
}

// Release ends the reply claim on the plugin message of the plugin. It reports
// whether the row exists, so a second release of the same claim still reports true.
func (r *TelegramMessage) Release(ctx context.Context, id, pluginID string) (bool, error) {
	query := `UPDATE public.telegram_messages SET claim_until = NULL
		WHERE id = $1 AND plugin_id = $2 AND ` + telegramMessageAlive + `;`

	result, err := r.tx.ExecContext(ctx, query, id, pluginID)
	if err != nil {
		return false, fmt.Errorf("releasing the claim of TelegramMessage: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("counting the released TelegramMessage: %w", err)
	}

	return affected > 0, nil
}

// Purge deletes the plugin messages of the acting user that passed their lifetime.
func (r *TelegramMessage) Purge(ctx context.Context) error {
	query := `DELETE FROM public.telegram_messages WHERE NOT (` + telegramMessageAlive + `);`

	if _, err := r.tx.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("purging TelegramMessage: %w", err)
	}

	return nil
}

func (r *TelegramMessage) getOne(
	ctx context.Context,
	by string,
	query string,
	args ...any,
) (*model.TelegramMessage, error) {
	var entity model.TelegramMessage

	if err := r.tx.GetContext(ctx, &entity, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // no row is no plugin message, not a failure.
		}

		return nil, fmt.Errorf("getting TelegramMessage %s: %w", by, err)
	}

	return &entity, nil
}
