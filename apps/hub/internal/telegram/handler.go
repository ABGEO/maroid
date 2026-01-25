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
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/telegram/command"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// UpdatesHandler represents a handler for Telegram updates.
type UpdatesHandler interface {
	Handle(ctx context.Context) error
	Stop(ctx context.Context) error
}

// ChannelHandler is an implementation of UpdatesHandler that handles Telegram updates via webhooks.
type ChannelHandler struct {
	cfg              *config.Config
	bot              *telego.Bot
	logger           *slog.Logger
	router           chi.Router
	commandsRegistry *registry.TelegramCommandRegistry

	updates    <-chan telego.Update
	botHandler *th.BotHandler
}

type commandScope struct {
	scope    telego.BotCommandScope
	commands []telego.BotCommand
}

var _ UpdatesHandler = (*ChannelHandler)(nil)

// NewUpdatesHandler creates a new ChannelHandler for handling Telegram updates.
func NewUpdatesHandler(
	cfg *config.Config,
	logger *slog.Logger,
	bot *telego.Bot,
	router chi.Router,
	commandsRegistry *registry.TelegramCommandRegistry,
) (*ChannelHandler, error) {
	var (
		err            error
		webhookOptions []telego.WebhookOption
	)

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

	handlerInstance := &ChannelHandler{
		cfg: cfg,
		bot: bot,
		logger: logger.With(
			slog.String("component", "telegram-updates-handler"),
		),
		router:           router,
		commandsRegistry: commandsRegistry,
	}

	handlerInstance.updates, err = bot.UpdatesViaWebhook(
		ctx,
		handlerInstance.getWebhookHandler(),
		webhookOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram updates via webhook: %w", err)
	}

	handlerInstance.botHandler, err = th.NewBotHandler(bot, handlerInstance.updates)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot handler: %w", err)
	}

	return handlerInstance, nil
}

// Handle starts handling Telegram updates.
func (h *ChannelHandler) Handle(ctx context.Context) error {
	h.registerHandlers()

	err := h.setCommands(ctx)
	if err != nil {
		return err
	}

	err = h.botHandler.Start()
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

func (h *ChannelHandler) getWebhookHandler() func(handler telego.WebhookHandler) error {
	return func(handler telego.WebhookHandler) error {
		h.router.Post(h.cfg.Telegram.Webhook.Path, func(w http.ResponseWriter, r *http.Request) {
			defer func() { _ = r.Body.Close() }()

			if r.Header.Get(telego.WebhookSecretTokenHeader) != h.bot.SecretToken() {
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

func (h *ChannelHandler) registerHandlers() {
	unknownCommand := command.NewUnknown(h.bot)

	for _, cmd := range h.commandsRegistry.All() {
		cmdName := cmd.Meta().Command

		h.botHandler.Handle(wrapCommandHandler(cmd), th.CommandEqual(cmdName))
		h.logger.Info("command handler has been registered", slog.String("command", cmdName))
	}

	h.botHandler.Handle(unknownCommand.Handle, th.AnyCommand())
}

func wrapCommandHandler(
	cmd pluginapi.TelegramCommand,
) func(ctx *th.Context, update telego.Update) error {
	return func(ctx *th.Context, update telego.Update) error {
		// @todo: log command execution attempt.
		err := cmd.Validate(update)
		if err != nil {
			// @todo: send message
			return fmt.Errorf("command validation failed: %w", err)
		}

		err = cmd.Handle(ctx, update)
		if err != nil {
			// @todo: send message
			return fmt.Errorf("command handling failed: %w", err)
		}

		return nil
	}
}

func (h *ChannelHandler) setCommands(ctx context.Context) error {
	if !h.cfg.Telegram.Setup {
		return nil
	}

	return h.applyCommandsToScopes(ctx, h.groupCommandsByScope())
}

func (h *ChannelHandler) groupCommandsByScope() map[string]*commandScope {
	commandsByScope := make(map[string]*commandScope)

	for _, cmd := range h.commandsRegistry.All() {
		meta := cmd.Meta()
		scope := meta.Scope

		if scope == nil {
			scope = tu.ScopeDefault()
		}

		scopeType := scope.ScopeType()

		cs, exists := commandsByScope[scopeType]
		if !exists {
			cs = &commandScope{
				scope:    scope,
				commands: make([]telego.BotCommand, 0),
			}
			commandsByScope[scopeType] = cs
		}

		cs.commands = append(cs.commands, telego.BotCommand{
			Command:     meta.Command,
			Description: meta.Description,
		})
	}

	return commandsByScope
}

func (h *ChannelHandler) applyCommandsToScopes(
	ctx context.Context,
	commandsByScope map[string]*commandScope,
) error {
	for _, cs := range commandsByScope {
		err := h.bot.SetMyCommands(ctx, &telego.SetMyCommandsParams{
			Commands: cs.commands,
			Scope:    cs.scope,
		})
		if err != nil {
			return fmt.Errorf("failed to set bot commands for scope %T: %w", cs.scope, err)
		}
	}

	return nil
}
