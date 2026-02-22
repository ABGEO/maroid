package step

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/abgeo/maroid/libs/pluginapi"
	conversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
	"github.com/abgeo/maroid/plugins/parking/service"
)

var lotNumberRegex = regexp.MustCompile(`^[A-Ca-c]\d+$`)

// EnterLot is a conversation step that prompts the user to enter the parking lot number manually.
type EnterLot struct {
	telegramBot  pluginapi.TelegramBot
	apiClientSvc service.APIClientService
}

var _ conversationapi.Step = (*EnterLot)(nil)

// NewEnterLot creates a new EnterLot step.
func NewEnterLot(
	telegramBot pluginapi.TelegramBot,
	apiClientSvc service.APIClientService,
) *EnterLot {
	return &EnterLot{
		telegramBot:  telegramBot,
		apiClientSvc: apiClientSvc,
	}
}

// ID returns the unique identifier for the step.
func (s *EnterLot) ID() string { return stepEnterLot }

// OnEnter is called when the step is entered. It prompts the user to enter the parking lot number manually.
func (s *EnterLot) OnEnter(
	ctx *conversationapi.Context,
	_ telego.Update,
) error {
	return editCtrlMessageWithKeyboard(
		s.telegramBot, ctx,
		"Please enter the parking lot number manually."+
			"\n\nJust type the number and send it."+
			"\n\nExpected format: <b>[A-C][number]</b> (e.g. A124, B3, C42)",
		cancelKeyboard(),
	)
}

// OnMessage is called when a message is received while this step is active.
// It validates the entered lot number and checks if it exists.
func (s *EnterLot) OnMessage(
	ctx *conversationapi.Context,
	update telego.Update,
) (string, error) {
	if update.CallbackQuery != nil {
		if err := answerCallback(s.telegramBot, update); err != nil {
			return "", err
		}

		if update.CallbackQuery.Data == callbackCancel {
			err := editCtrlMessage(
				s.telegramBot, ctx, "Parking cancelled.",
			)

			return "", err
		}
	}

	if update.Message == nil || update.Message.Text == "" {
		return stepEnterLot, nil
	}

	return s.processLot(ctx, update)
}

func (s *EnterLot) processLot(ctx *conversationapi.Context, update telego.Update) (string, error) {
	lot := strings.TrimSpace(update.Message.Text)

	if !lotNumberRegex.MatchString(lot) {
		_ = editCtrlMessageWithKeyboard(
			s.telegramBot, ctx,
			"Invalid lot number format."+
				"\n\nExpected format: <b>[A-C][number]</b> "+
				"(e.g. A124, B3, C42)."+
				"\n\nPlease try again.",
			cancelKeyboard(),
		)

		return stepEnterLot, nil
	}

	lot = strings.ToUpper(lot)
	zone := string(lot[0])
	number := lot[1:]

	place, err := s.apiClientSvc.GetParkingPlace(
		context.Background(), zone, number,
	)
	if errors.Is(err, service.ErrParkingPlaceNotFound) {
		_ = editCtrlMessageWithKeyboard(
			s.telegramBot, ctx,
			"Parking lot <b>"+lot+"</b> was not found."+
				"\n\nPlease check the number and try again.",
			cancelKeyboard(),
		)

		return stepEnterLot, nil
	}

	if err != nil {
		return "", fmt.Errorf("fetching parking place: %w", err)
	}

	ctx.Data["lot_number"] = lot
	ctx.Data["free_parking"] = place.FreeParking

	return stepSelectType, nil
}
