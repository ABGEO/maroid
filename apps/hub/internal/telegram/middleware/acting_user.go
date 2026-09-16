package middleware

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	teleupdate "github.com/abgeo/maroid/apps/hub/internal/telegram/update"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// ActingUser returns a middleware that drops an update from a person that holds
// no active user record, and puts the acting user into the context of every
// other update.
func ActingUser(logger *slog.Logger, resolver auth.IdentityResolver) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		actingUser, ok := resolveActingUser(ctx, logger, resolver, update)
		if !ok {
			return nil
		}

		return ctx.WithContext(
			pluginapi.ContextWithActingUser(ctx.Context(), actingUser),
		).Next(update)
	}
}

// resolveActingUser returns the acting user of the update. It reports false when
// the update reaches nothing, because no active record owns it.
func resolveActingUser(
	ctx context.Context,
	logger *slog.Logger,
	resolver auth.IdentityResolver,
	update telego.Update,
) (string, bool) {
	sender := teleupdate.SentFrom(update)
	if sender == nil {
		logger.WarnContext(
			ctx,
			"received update with no identifiable sender, dropping",
			slog.Any("update", update),
		)

		return "", false
	}

	user, err := resolver.ResolveByProvider(
		ctx,
		auth.ProviderTelegram,
		strconv.FormatInt(sender.ID, 10),
	)
	if err != nil {
		logger.WarnContext(
			ctx,
			"sender holds no active user record",
			slog.Int64("telegram_id", sender.ID),
			slog.String("username", sender.Username),
			slog.Any("error", err),
		)

		return "", false
	}

	return user.ID, true
}
