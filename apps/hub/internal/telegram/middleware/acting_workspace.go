package middleware

import (
	"context"
	"log/slog"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	teleupdate "github.com/abgeo/maroid/apps/hub/internal/telegram/update"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// ChatWorkspaces answers the workspace that a chat of the acting user acts in.
type ChatWorkspaces interface {
	Acting(ctx context.Context, chatID int64) (string, error)
}

// ActingWorkspace returns a middleware that puts the workspace of the chat into the
// context of the update. An update with no workspace still reaches the commands of
// the hub, and the wrapper of a plugin command refuses it.
func ActingWorkspace(logger *slog.Logger, chats ChatWorkspaces) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		acting := resolveActingWorkspace(ctx, logger, chats, update)
		if acting == "" {
			return ctx.Next(update)
		}

		return ctx.WithContext(
			pluginapi.ContextWithActingWorkspace(ctx.Context(), acting),
		).Next(update)
	}
}

func resolveActingWorkspace(
	ctx context.Context,
	logger *slog.Logger,
	chats ChatWorkspaces,
	update telego.Update,
) string {
	chatID, ok := teleupdate.ChatOf(update)
	if !ok {
		return ""
	}

	acting, err := chats.Acting(ctx, chatID)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"resolving the workspace of the chat",
			slog.Int64("chat_id", chatID),
			slog.Any("error", err),
		)

		return ""
	}

	return acting
}
