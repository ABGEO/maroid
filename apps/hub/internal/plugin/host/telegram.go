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
