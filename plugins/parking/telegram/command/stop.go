package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/service"
)

// Stop is a Telegram command that allows users to stop their active parking session.
type Stop struct {
	bot          pluginapi.TelegramBot
	apiClientSvc service.APIClientService
}

var _ pluginapi.TelegramCommand = (*Stop)(nil)

// NewStop creates a new Stop.
func NewStop(
	bot pluginapi.TelegramBot,
	apiClientSvc service.APIClientService,
) *Stop {
	return &Stop{
		bot:          bot,
		apiClientSvc: apiClientSvc,
	}
}

// Meta returns the metadata for the command.
func (c *Stop) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "stop",
		Description: "Stop the active parking session",
	}
}

// Validate checks if the update is valid for this command.
func (c *Stop) Validate(_ telego.Update) error {
	return nil
}

// Handle processes the command.
func (c *Stop) Handle(ctx *th.Context, update telego.Update) error {
	session, err := c.apiClientSvc.GetActiveSession(ctx.Context())
	if err != nil {
		return fmt.Errorf("fetching active session: %w", err)
	}

	if session == nil {
		return sendMessage(c.bot, update, "No active parking session to stop.")
	}

	stopped, err := c.apiClientSvc.StopParking(ctx.Context(), session.ID)
	if err != nil {
		return fmt.Errorf("stopping parking session: %w", err)
	}

	text := fmt.Sprintf(
		"Parking stopped!\n\n"+
			"Lot: <b>%s</b>\n"+
			"Address: <b>%s</b>\n"+
			"Started at: <b>%s</b>\n"+
			"Ended at: <b>%s</b>",
		stopped.ParkingPlace.UniqueNumber,
		stopped.ParkingPlace.Address,
		stopped.StartDate,
		ptrOr(stopped.EndDate, "unknown"),
	)

	return sendMessage(c.bot, update, text)
}
