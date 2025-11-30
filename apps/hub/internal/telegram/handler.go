package telegram

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

// UpdatesHandler represents a handler for Telegram updates.
type UpdatesHandler interface {
	BotHandler() *th.BotHandler
	Handle() error
	Stop(ctx context.Context) error
}

// ChannelHandler is an implementation of UpdatesHandler that handles Telegram updates via webhooks.
type ChannelHandler struct {
	bot    *telego.Bot
	logger *slog.Logger

	updates    <-chan telego.Update
	botHandler *th.BotHandler
}

var _ UpdatesHandler = (*ChannelHandler)(nil)

// NewUpdatesHandler creates a new ChannelHandler for handling Telegram updates.
func NewUpdatesHandler(
	cfg *config.Config,
	logger *slog.Logger,
	bot *telego.Bot,
	router chi.Router,
) (*ChannelHandler, error) {
	var webhookOptions []telego.WebhookOption

	ctx := context.Background()

	if cfg.Telegram.Setup {
		webhookOptions = append(
			webhookOptions,
			telego.WithWebhookSet(ctx, &telego.SetWebhookParams{
				URL:         cfg.Server.Hostname + cfg.Telegram.Webhook.Path,
				SecretToken: bot.SecretToken(),
			}),
		)
	}

	webhookHandler := getWebhookHandler(router, cfg.Telegram.Webhook.Path, bot.SecretToken())

	updates, err := bot.UpdatesViaWebhook(ctx, webhookHandler, webhookOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram updates via webhook: %w", err)
	}

	botHandler, err := th.NewBotHandler(bot, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot handler: %w", err)
	}

	return &ChannelHandler{
		bot: bot,
		logger: logger.With(
			slog.String("component", "telegram-updates-handler"),
		),
		updates:    updates,
		botHandler: botHandler,
	}, nil
}

// BotHandler returns the underlying BotHandler.
func (h *ChannelHandler) BotHandler() *th.BotHandler {
	return h.botHandler
}

// Handle starts handling Telegram updates.
func (h *ChannelHandler) Handle() error {
	err := h.botHandler.Start()
	if err != nil {
		return fmt.Errorf("failed to start telegram bot handler: %w", err)
	}

	return nil
}

// Stop stops handling Telegram updates.
func (h *ChannelHandler) Stop(ctx context.Context) error {
	err := h.botHandler.StopWithContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to stop telegram bot handler: %w", err)
	}

	return nil
}

func getWebhookHandler(
	router chi.Router,
	pattern string,
	token string,
) func(handler telego.WebhookHandler) error {
	return func(handler telego.WebhookHandler) error {
		router.Post(pattern, func(w http.ResponseWriter, r *http.Request) {
			defer func() { _ = r.Body.Close() }()

			if r.Header.Get(telego.WebhookSecretTokenHeader) != token {
				w.WriteHeader(http.StatusUnauthorized)

				return
			}

			data, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)

				return
			}

			if err = handler(r.Context(), data); err != nil {
				w.WriteHeader(http.StatusInternalServerError)

				return
			}

			w.WriteHeader(http.StatusOK)
		})

		return nil
	}
}
