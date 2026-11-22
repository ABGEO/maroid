// Package pluginmsg sends the plugin messages of a plugin to the acting user, and
// routes a press and a reply on them back to the plugin that sent them.
package pluginmsg

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
	messageapi "github.com/abgeo/maroid/libs/pluginapi/telegram/message"
)

// The descriptions that Telegram answers, which the service reads.
const (
	descriptionChatNotFound = "chat not found"
	descriptionNotModified  = "message is not modified"
)

// Service sends, edits, and releases the plugin messages of the acting user.
type Service struct {
	logger *slog.Logger
	db     *sqlx.DB
	bot    *telego.Bot
	now    func() time.Time
}

// NewService creates a Service.
func NewService(logger *slog.Logger, db *sqlx.DB, bot *telego.Bot) *Service {
	return &Service{logger: logger, db: db, bot: bot, now: time.Now}
}

// Send sends the message of the plugin to the private chat of the acting user, and
// records it.
func (s *Service) Send(
	ctx context.Context,
	pluginID string,
	msg messageapi.Message,
) (messageapi.ID, error) {
	if pluginapi.ActingUserFromContext(ctx) == "" {
		return "", messageapi.ErrNoActingUser
	}

	if err := Validate(msg, s.now()); err != nil {
		return "", err
	}

	chatID, err := s.chatOfActingUser(ctx)
	if err != nil {
		return "", err
	}

	params := &telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: chatID},
		Text:        msg.Text,
		ReplyMarkup: Markup(msg),
	}

	sent, err := s.bot.SendMessage(ctx, params)
	if err != nil {
		return "", fmt.Errorf("sending the plugin message: %w", unreachable(err))
	}

	id, err := s.record(ctx, pluginID, chatID, sent.MessageID, msg.ClaimReplyUntil)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"recording a sent plugin message",
			slog.String("plugin_id", pluginID),
			slog.Int64("chat_id", chatID),
			slog.Any("error", err),
		)

		return "", err
	}

	return messageapi.ID(id), nil
}

// Edit replaces the text and the inline buttons of a message that the plugin sent.
func (s *Service) Edit(
	ctx context.Context,
	pluginID string,
	id messageapi.ID,
	text string,
	buttons [][]messageapi.Button,
) error {
	if err := validateText(text); err != nil {
		return err
	}

	if err := validateButtons(buttons); err != nil {
		return err
	}

	message, err := s.byID(ctx, pluginID, id)
	if err != nil {
		return err
	}

	_, err = s.bot.EditMessageText(ctx, &telego.EditMessageTextParams{
		ChatID:      telego.ChatID{ID: message.ChatID},
		MessageID:   int(message.MessageID),
		Text:        text,
		ReplyMarkup: InlineMarkup(buttons),
	})
	if err != nil && !isTelegramError(err, http.StatusBadRequest, descriptionNotModified) {
		return fmt.Errorf("editing the plugin message: %w", unreachable(err))
	}

	return nil
}

// Release ends the reply claim on a message that the plugin sent.
func (s *Service) Release(
	ctx context.Context,
	pluginID string,
	id messageapi.ID,
) error {
	if uuid.Validate(string(id)) != nil {
		return messageapi.ErrUnknown
	}

	var found bool

	err := database.WithScopeTx(ctx, s.db, func(tx *sqlx.Tx) error {
		var err error

		found, err = repository.NewTelegramMessage(tx).Release(ctx, string(id), pluginID)

		return err //nolint:wrapcheck // the repository wraps it.
	})
	if err != nil {
		return fmt.Errorf("releasing the claim of a plugin message: %w", err)
	}

	if !found {
		return messageapi.ErrUnknown
	}

	return nil
}

func (s *Service) chatOfActingUser(ctx context.Context) (int64, error) {
	identities, err := database.FetchScopeTx(
		ctx,
		s.db,
		func(tx *sqlx.Tx) ([]model.Identity, error) {
			return repository.NewIdentity(tx).ListByUser(ctx, pluginapi.ActingUserFromContext(ctx))
		},
	)
	if err != nil {
		return 0, fmt.Errorf("reading the Telegram account of the acting user: %w", err)
	}

	for _, identity := range identities {
		if identity.Provider != auth.ProviderTelegram {
			continue
		}

		chatID, err := strconv.ParseInt(identity.ProviderUserID, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("reading the Telegram account of the acting user: %w", err)
		}

		return chatID, nil
	}

	return 0, messageapi.ErrNoAccount
}

func (s *Service) record(
	ctx context.Context,
	pluginID string,
	chatID int64,
	messageID int,
	claimUntil time.Time,
) (string, error) {
	row := &model.TelegramMessage{
		PluginID:  pluginID,
		ChatID:    chatID,
		MessageID: int64(messageID),
	}

	if !claimUntil.IsZero() {
		row.ClaimUntil = &claimUntil
	}

	var id string

	err := database.WithScopeTx(ctx, s.db, func(tx *sqlx.Tx) error {
		messages := repository.NewTelegramMessage(tx)
		if err := messages.Purge(ctx); err != nil {
			return err //nolint:wrapcheck // the repository wraps it.
		}

		var err error

		id, err = messages.Insert(ctx, row)

		return err //nolint:wrapcheck // the repository wraps it.
	})
	if err != nil {
		return "", fmt.Errorf("recording the plugin message: %w", err)
	}

	return id, nil
}

func (s *Service) byID(
	ctx context.Context,
	pluginID string,
	id messageapi.ID,
) (*model.TelegramMessage, error) {
	if uuid.Validate(string(id)) != nil {
		return nil, messageapi.ErrUnknown
	}

	message, err := database.FetchScopeTx(
		ctx,
		s.db,
		func(tx *sqlx.Tx) (*model.TelegramMessage, error) {
			return repository.NewTelegramMessage(tx).ByID(ctx, string(id), pluginID)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("reading the plugin message: %w", err)
	}

	if message == nil {
		return nil, messageapi.ErrUnknown
	}

	return message, nil
}

// unreachable maps the refusal of Telegram to deliver to the person, a blocked bot or
// a chat that never opened, to messageapi.ErrUnreachable.
func unreachable(err error) error {
	if isTelegramError(err, http.StatusForbidden, "") ||
		isTelegramError(err, http.StatusBadRequest, descriptionChatNotFound) {
		return fmt.Errorf("%w: %w", messageapi.ErrUnreachable, err)
	}

	return err
}

func isTelegramError(err error, code int, description string) bool {
	var apiErr *ta.Error
	if !errors.As(err, &apiErr) {
		return false
	}

	return apiErr.ErrorCode == code && strings.Contains(apiErr.Description, description)
}
