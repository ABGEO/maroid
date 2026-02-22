package host

import (
	"context"

	"github.com/mymmrac/telego"
)

type telegramBotWrapper struct {
	bot *telego.Bot
}

func (t *telegramBotWrapper) SendMessage(
	ctx context.Context,
	params *telego.SendMessageParams,
) (*telego.Message, error) {
	return t.bot.SendMessage(ctx, params) //nolint:wrapcheck
}

func (t *telegramBotWrapper) AnswerCallbackQuery(
	ctx context.Context,
	params *telego.AnswerCallbackQueryParams,
) error {
	return t.bot.AnswerCallbackQuery(ctx, params) //nolint:wrapcheck
}

func (t *telegramBotWrapper) EditMessageText(
	ctx context.Context,
	params *telego.EditMessageTextParams,
) (*telego.Message, error) {
	return t.bot.EditMessageText(ctx, params) //nolint:wrapcheck
}
