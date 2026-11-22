package pluginmsg

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
	messageapi "github.com/abgeo/maroid/libs/pluginapi/telegram/message"
)

// The answers that the bot gives in the name of the hub.
const (
	AnswerFailed         = "Something went wrong. Try again."
	AnswerButtonGone     = "This button no longer works."
	AnswerNoClaim        = "This message takes no answer now."
	botCommandEntityType = "bot_command"
)

// ConversationHolder reports whether a person holds an open conversation.
type ConversationHolder interface {
	Holds(telegramUserID string) bool
}

// Router routes a press and a reply on a plugin message to the plugin that sent it.
type Router struct {
	logger        *slog.Logger
	db            *sqlx.DB
	bot           *telego.Bot
	receivers     *registry.TelegramMessageRegistry
	conversations ConversationHolder
}

// NewRouter creates a Router.
func NewRouter(
	logger *slog.Logger,
	db *sqlx.DB,
	bot *telego.Bot,
	receivers *registry.TelegramMessageRegistry,
	conversations ConversationHolder,
) *Router {
	return &Router{
		logger:        logger,
		db:            db,
		bot:           bot,
		receivers:     receivers,
		conversations: conversations,
	}
}

// Middleware answers the middleware that takes the update that a plugin owns, and
// passes every other update on. It runs after the middlewares that set the acting
// user and the acting workspace.
func (r *Router) Middleware() th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		taken, err := r.take(ctx, update)
		if err != nil {
			return err
		}

		if taken {
			return nil
		}

		return ctx.Next(update)
	}
}

// Fallback answers a press that no handler took and that no open conversation waits
// for. It passes every other update to next.
func (r *Router) Fallback(next th.Handler) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		query := update.CallbackQuery
		if query == nil || r.conversations.Holds(strconv.FormatInt(query.From.ID, 10)) {
			return next(ctx, update)
		}

		return r.answerPress(ctx, query.ID, AnswerButtonGone)
	}
}

func (r *Router) take(ctx context.Context, update telego.Update) (bool, error) {
	if pluginapi.ActingUserFromContext(ctx) == "" {
		return false, nil
	}

	switch {
	case update.CallbackQuery != nil:
		return r.press(ctx, update.CallbackQuery)
	case update.Message != nil && !isCommand(update.Message) && carriesAnswer(update.Message):
		return r.message(ctx, update.Message)
	default:
		return false, nil
	}
}

func (r *Router) press(ctx context.Context, query *telego.CallbackQuery) (bool, error) {
	if query.Message == nil {
		return false, nil
	}

	row, err := r.byTelegram(ctx, query.Message.GetChat().ID, int64(query.Message.GetMessageID()))
	if err != nil || row == nil {
		return false, err
	}

	receiver, loaded := r.receivers.Get(row.PluginID)
	if !loaded {
		return true, r.answerPress(ctx, query.ID, AnswerButtonGone)
	}

	notice, err := receiver.OnTelegramPress(withoutWorkspace(ctx), messageapi.Press{
		Message: messageapi.ID(row.ID),
		Value:   query.Data,
	})
	if err != nil {
		r.logFailure(ctx, "a plugin failed on a press", row, query.From.ID, err)

		notice = AnswerFailed
	}

	return true, r.answerPress(ctx, query.ID, notice)
}

func (r *Router) message(ctx context.Context, message *telego.Message) (bool, error) {
	if message.ReplyToMessage != nil {
		return r.reply(ctx, message)
	}

	if message.From == nil || r.conversations.Holds(strconv.FormatInt(message.From.ID, 10)) {
		return false, nil
	}

	claims, err := database.FetchScopeTx(
		ctx,
		r.db,
		func(tx *sqlx.Tx) ([]model.TelegramMessage, error) {
			return repository.NewTelegramMessage(tx).OpenClaims(ctx)
		},
	)
	if err != nil {
		return false, fmt.Errorf("reading the open claims of the person: %w", err)
	}

	for index := range claims {
		if receiver, loaded := r.receivers.Get(claims[index].PluginID); loaded {
			return true, r.deliver(ctx, receiver, &claims[index], message)
		}
	}

	return false, nil
}

func (r *Router) reply(ctx context.Context, message *telego.Message) (bool, error) {
	row, err := r.byTelegram(ctx, message.Chat.ID, int64(message.ReplyToMessage.MessageID))
	if err != nil || row == nil {
		return false, err
	}

	receiver, loaded := r.receivers.Get(row.PluginID)
	if !row.ClaimOpen || !loaded {
		return true, r.say(ctx, message.Chat.ID, AnswerNoClaim)
	}

	return true, r.deliver(ctx, receiver, row, message)
}

func (r *Router) deliver(
	ctx context.Context,
	receiver pluginapi.TelegramMessagePlugin,
	row *model.TelegramMessage,
	message *telego.Message,
) error {
	reply := messageapi.Reply{
		Message: messageapi.ID(row.ID),
		Text:    message.Text,
	}

	if message.Location != nil {
		reply.Location = &messageapi.Location{
			Latitude:  message.Location.Latitude,
			Longitude: message.Location.Longitude,
		}
	}

	if err := receiver.OnTelegramReply(withoutWorkspace(ctx), reply); err != nil {
		r.logFailure(ctx, "a plugin failed on a reply", row, senderID(message), err)

		return r.say(ctx, message.Chat.ID, AnswerFailed)
	}

	return nil
}

func (r *Router) byTelegram(
	ctx context.Context,
	chatID, messageID int64,
) (*model.TelegramMessage, error) {
	row, err := database.FetchScopeTx(ctx, r.db, func(tx *sqlx.Tx) (*model.TelegramMessage, error) {
		return repository.NewTelegramMessage(tx).ByTelegram(ctx, chatID, messageID)
	})
	if err != nil {
		return nil, fmt.Errorf("reading the plugin message of an update: %w", err)
	}

	return row, nil
}

func (r *Router) answerPress(ctx context.Context, queryID, notice string) error {
	err := r.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: queryID,
		Text:            notice,
	})
	if err != nil {
		return fmt.Errorf("answering a press: %w", err)
	}

	return nil
}

func (r *Router) say(ctx context.Context, chatID int64, text string) error {
	_, err := r.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
	})
	if err != nil {
		return fmt.Errorf("answering a reply: %w", err)
	}

	return nil
}

func (r *Router) logFailure(
	ctx context.Context,
	what string,
	row *model.TelegramMessage,
	telegramUserID int64,
	err error,
) {
	r.logger.ErrorContext(
		ctx,
		what,
		slog.String("plugin_id", row.PluginID),
		slog.String("telegram_message_id", row.ID),
		slog.Int64("telegram_user_id", telegramUserID),
		slog.Any("error", err),
	)
}

// withoutWorkspace clears the workspace that the chat selected, because a plugin
// message belongs to no workspace. The acting user and the time zone stay.
func withoutWorkspace(ctx context.Context) context.Context {
	return workspace.ContextWithRole(pluginapi.ContextWithActingWorkspace(ctx, ""), "")
}

func isCommand(message *telego.Message) bool {
	return len(message.Entities) > 0 &&
		message.Entities[0].Type == botCommandEntityType &&
		message.Entities[0].Offset == 0
}

func carriesAnswer(message *telego.Message) bool {
	return message.Text != "" || message.Location != nil
}

func senderID(message *telego.Message) int64 {
	if message.From == nil {
		return 0
	}

	return message.From.ID
}
