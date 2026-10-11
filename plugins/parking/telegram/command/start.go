package command

import (
	"context"
	"fmt"
	"sync"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
	"github.com/abgeo/maroid/plugins/parking/dto"
	"github.com/abgeo/maroid/plugins/parking/model"
	"github.com/abgeo/maroid/plugins/parking/permission"
	"github.com/abgeo/maroid/plugins/parking/service"
)

// Parking is a Telegram command that allows users to start a parking session.
type Parking struct {
	telegramConversationEngine conversation.Engine
	apiClientSvc               service.APIClientService
	answers                    *Notice
}

var _ pluginapi.TelegramCommand = (*Parking)(nil)

// NewParking creates a new Parking.
func NewParking(
	telegramConversationEngine conversation.Engine,
	apiClientSvc service.APIClientService,
	answers *Notice,
) *Parking {
	return &Parking{
		telegramConversationEngine: telegramConversationEngine,
		apiClientSvc:               apiClientSvc,
		answers:                    answers,
	}
}

// Meta returns the metadata for the command.
func (c *Parking) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "start",
		Description: "Start a parking session",
		Permission:  permission.SessionsWrite,
	}
}

// Validate checks if the update is valid for this command.
func (c *Parking) Validate(_ telego.Update) error {
	return nil
}

// Handle processes the command.
func (c *Parking) Handle(ctx *th.Context, update telego.Update) error {
	state, err := c.preflight(ctx.Context())
	if err != nil {
		return replyFailure(ctx, update, c.answers, err)
	}

	if msg := validateStart(state); msg != "" {
		return sendMessage(ctx, update, msg)
	}

	if err = c.telegramConversationEngine.Start(ctx, update, "start_parking"); err != nil {
		return fmt.Errorf("starting parking conversation: %w", err)
	}

	return nil
}

// startState is what the preflight of a start reads from the parking service.
type startState struct {
	person   *dto.Person
	session  *dto.ActiveSession
	vehicles []model.Vehicle
}

// preflight reads the person, the active session, and the vehicles in parallel, so a
// start that cannot succeed ends before the first question.
func (c *Parking) preflight(ctx context.Context) (*startState, error) {
	var (
		state                          startState
		personErr, sessErr, vehicleErr error
		wg                             sync.WaitGroup
	)

	wg.Go(func() { state.person, personErr = c.apiClientSvc.GetPerson(ctx) })
	wg.Go(func() { state.session, sessErr = c.apiClientSvc.GetActiveSession(ctx) })
	wg.Go(func() { state.vehicles, vehicleErr = c.apiClientSvc.ListVehicles(ctx) })
	wg.Wait()

	if personErr != nil {
		return nil, fmt.Errorf("fetching person: %w", personErr)
	}

	if sessErr != nil {
		return nil, fmt.Errorf("fetching active session: %w", sessErr)
	}

	if vehicleErr != nil {
		return nil, fmt.Errorf("fetching vehicles: %w", vehicleErr)
	}

	return &state, nil
}

func validateStart(state *startState) string {
	if state.session != nil {
		return fmt.Sprintf(
			"A parking session is already active at lot <b>%s</b>."+
				"\n\nPlease stop it before starting a new one.",
			state.session.ParkingPlace.UniqueNumber,
		)
	}

	if state.person.BalanceAmount < 1 {
		return fmt.Sprintf(
			"Your balance is <b>%f GEL</b>."+
				"\n\nPlease top up before starting a parking session.",
			state.person.BalanceAmount,
		)
	}

	if len(state.vehicles) == 0 {
		return "Your parking account holds no vehicle." +
			"\n\nAdd one in the mobile application of the parking service, then start again."
	}

	return ""
}
