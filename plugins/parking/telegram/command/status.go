package command

import (
	"fmt"
	"math"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/service"
)

// Status is a Telegram command that allows users to check the status of their active parking session.
type Status struct {
	bot          pluginapi.TelegramBot
	apiClientSvc service.APIClientService
}

var _ pluginapi.TelegramCommand = (*Status)(nil)

// NewStatus creates a new Status.
func NewStatus(
	bot pluginapi.TelegramBot,
	apiClientSvc service.APIClientService,
) *Status {
	return &Status{
		bot:          bot,
		apiClientSvc: apiClientSvc,
	}
}

// Meta returns the metadata for the command.
func (c *Status) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "status",
		Description: "Check active parking session status",
	}
}

// Validate checks if the update is valid for this command.
func (c *Status) Validate(_ telego.Update) error {
	return nil
}

// Handle processes the command.
func (c *Status) Handle(ctx *th.Context, update telego.Update) error {
	session, err := c.apiClientSvc.GetActiveSession(ctx.Context())
	if err != nil {
		return fmt.Errorf("fetching active session: %w", err)
	}

	if session == nil {
		return sendMessage(c.bot, update, "No active parking session.")
	}

	elapsed := formatElapsed(session.StartDate)

	text := fmt.Sprintf(
		"Parking is <b>active</b>\n\n"+
			"Lot: <b>%s</b>\n"+
			"Address: <b>%s</b>\n"+
			"Started at: <b>%s</b>\n"+
			"Elapsed: <b>%s</b>",
		session.ParkingPlace.UniqueNumber,
		session.ParkingPlace.Address,
		session.StartDate,
		elapsed,
	)

	return sendMessage(c.bot, update, text)
}

func formatElapsed(startDate string) string {
	start, err := time.Parse("2006-01-02T15:04:05.999999", startDate)
	if err != nil {
		return "unknown"
	}

	d := time.Since(start)
	hours := int(math.Floor(d.Hours()))
	minutes := int(math.Floor(d.Minutes())) % 60 //nolint: mnd

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}

	return fmt.Sprintf("%dm", minutes)
}
