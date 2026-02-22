package middleware

import (
	"log/slog"
	"slices"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	teleupdate "github.com/abgeo/maroid/apps/hub/internal/telegram/update"
)

// AllowedUsers returns a middleware that silently drops updates
// from users whose Telegram ID is not in the allowed list.
func AllowedUsers(logger *slog.Logger, allowedUsers []int64) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		user := teleupdate.SentFrom(update)
		if user == nil {
			logger.Warn(
				"received update with no identifiable sender, dropping",
				slog.Any("update", update),
			)

			return nil
		}

		if !slices.Contains(allowedUsers, user.ID) {
			if user != nil {
				logger.Warn(
					"unauthorized access attempt",
					slog.Int64("user_id", user.ID),
					slog.String("username", user.Username),
				)
			}

			return nil
		}

		return ctx.Next(update)
	}
}
