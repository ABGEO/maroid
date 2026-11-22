package middleware

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	teleupdate "github.com/abgeo/maroid/apps/hub/internal/telegram/update"
)

// ActingUser returns a middleware that drops an update from a person that holds
// no active user record, and puts the acting user into the context of every
// other update.
func ActingUser(logger *slog.Logger, resolver auth.IdentityResolver) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		acting, ok := actingContext(ctx.Context(), logger, resolver, update)
		if !ok {
			return nil
		}

		return ctx.WithContext(acting).Next(update)
	}
}

// actingContext returns a context that carries the acting user of the update and the
// time zone of that record. It reports false when the update reaches nothing.
func actingContext(
	ctx context.Context,
	logger *slog.Logger,
	resolver auth.IdentityResolver,
	update telego.Update,
) (context.Context, bool) {
	user, ok := resolveActingUser(ctx, logger, resolver, update)
	if !ok {
		return ctx, false
	}

	return auth.WithActingUser(ctx, resolver, user), true
}

// resolveActingUser returns the record of the acting user of the update. It reports
// false when the update reaches nothing, because no active record owns it.
func resolveActingUser(
	ctx context.Context,
	logger *slog.Logger,
	resolver auth.IdentityResolver,
	update telego.Update,
) (*model.User, bool) {
	sender := teleupdate.SentFrom(update)
	if sender == nil {
		logger.WarnContext(
			ctx,
			"received update with no identifiable sender, dropping",
			slog.Any("update", update),
		)

		return nil, false
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

		return nil, false
	}

	return user, true
}
