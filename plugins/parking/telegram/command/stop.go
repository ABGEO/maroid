package command

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/permission"
	"github.com/abgeo/maroid/plugins/parking/service"
	"github.com/abgeo/maroid/plugins/parking/telegram/notice"
)

// Stop is a Telegram command that allows users to stop their active parking session.
type Stop struct {
	apiClientSvc service.APIClientService
	answers      *notice.Notice
}

var _ pluginapi.TelegramCommand = (*Stop)(nil)

// NewStop creates a new Stop.
func NewStop(
	apiClientSvc service.APIClientService,
	answers *notice.Notice,
) *Stop {
	return &Stop{
		apiClientSvc: apiClientSvc,
		answers:      answers,
	}
}

// Meta returns the metadata for the command.
func (c *Stop) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "stop",
		Description: "Stop the active parking session",
		Permission:  permission.SessionsWrite,
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
		return replyFailure(ctx, update, c.answers, fmt.Errorf("fetching active session: %w", err))
	}

	if session == nil {
		return sendMessage(ctx, update, "No active parking session to stop.")
	}

	stopped, err := c.apiClientSvc.StopParking(ctx.Context(), session.ID)
	if err != nil {
		return replyFailure(ctx, update, c.answers, fmt.Errorf("stopping parking session: %w", err))
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

	return sendMessage(ctx, update, text)
}
