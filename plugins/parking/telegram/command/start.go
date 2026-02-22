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
	"github.com/abgeo/maroid/plugins/parking/service"
)

// Parking is a Telegram command that allows users to start a parking session.
type Parking struct {
	bot                        pluginapi.TelegramBot
	telegramConversationEngine conversation.Engine
	apiClientSvc               service.APIClientService
}

var _ pluginapi.TelegramCommand = (*Parking)(nil)

// NewParking creates a new Parking.
func NewParking(
	bot pluginapi.TelegramBot,
	telegramConversationEngine conversation.Engine,
	apiClientSvc service.APIClientService,
) *Parking {
	return &Parking{
		bot:                        bot,
		telegramConversationEngine: telegramConversationEngine,
		apiClientSvc:               apiClientSvc,
	}
}

// Meta returns the metadata for the command.
func (c *Parking) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "start",
		Description: "Start a parking session",
	}
}

// Validate checks if the update is valid for this command.
func (c *Parking) Validate(_ telego.Update) error {
	return nil
}

// Handle processes the command.
func (c *Parking) Handle(ctx *th.Context, update telego.Update) error {
	person, session, err := c.preflight(ctx.Context())
	if err != nil {
		return err
	}

	if msg := validateStart(person, session); msg != "" {
		return sendMessage(c.bot, update, msg)
	}

	if err = c.telegramConversationEngine.Start(update, "start_parking"); err != nil {
		return fmt.Errorf("starting parking conversation: %w", err)
	}

	return nil
}

func (c *Parking) preflight(ctx context.Context) (*dto.Person, *dto.ActiveSession, error) {
	const actionsCount = 2

	var (
		person    *dto.Person
		session   *dto.ActiveSession
		personErr error
		sessErr   error
		wg        sync.WaitGroup
	)

	wg.Add(actionsCount)

	go func() {
		defer wg.Done()

		person, personErr = c.apiClientSvc.GetPerson(ctx)
	}()

	go func() {
		defer wg.Done()

		session, sessErr = c.apiClientSvc.GetActiveSession(ctx)
	}()

	wg.Wait()

	if personErr != nil {
		return nil, nil, fmt.Errorf("fetching person: %w", personErr)
	}

	if sessErr != nil {
		return nil, nil, fmt.Errorf("fetching active session: %w", sessErr)
	}

	return person, session, nil
}

func validateStart(person *dto.Person, session *dto.ActiveSession) string {
	if session != nil {
		return fmt.Sprintf(
			"A parking session is already active at lot <b>%s</b>."+
				"\n\nPlease stop it before starting a new one.",
			session.ParkingPlace.UniqueNumber,
		)
	}

	if person.BalanceAmount <= 0 {
		return "Your balance is <b>0 GEL</b>." +
			"\n\nPlease top up before starting a parking session."
	}

	return ""
}
