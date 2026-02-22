package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/service"
)

// Balance is a Telegram command that allows users to check their parking balance and daily free parking left.
type Balance struct {
	bot          pluginapi.TelegramBot
	apiClientSvc service.APIClientService
}

var _ pluginapi.TelegramCommand = (*Balance)(nil)

// NewBalance creates a new Balance.
func NewBalance(
	bot pluginapi.TelegramBot,
	apiClientSvc service.APIClientService,
) *Balance {
	return &Balance{
		bot:          bot,
		apiClientSvc: apiClientSvc,
	}
}

// Meta returns the metadata for the command.
func (c *Balance) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "balance",
		Description: "Check parking balance",
	}
}

// Validate checks if the update is valid for this command.
func (c *Balance) Validate(_ telego.Update) error {
	return nil
}

// Handle processes the command.
func (c *Balance) Handle(ctx *th.Context, update telego.Update) error {
	person, err := c.apiClientSvc.GetPerson(ctx.Context())
	if err != nil {
		return fmt.Errorf("fetching person: %w", err)
	}

	text := fmt.Sprintf(
		"<b>%s %s</b>\n\n"+
			"Balance: <b>%.2f GEL</b>\n"+
			"Daily free parking left: <b>%d</b>",
		person.FirstName,
		person.LastName,
		person.BalanceAmount,
		person.DailyFreeParkingLeft,
	)

	return sendMessage(c.bot, update, text)
}
