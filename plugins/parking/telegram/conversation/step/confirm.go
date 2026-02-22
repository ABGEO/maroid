package step

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
	conversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
	"github.com/abgeo/maroid/plugins/parking/service"
)

// Confirm is a conversation step that confirms the user's parking session details before starting it.
type Confirm struct {
	telegramBot  pluginapi.TelegramBot
	apiClientSvc service.APIClientService
}

var _ conversationapi.Step = (*Confirm)(nil)

// NewConfirm creates a new Confirm step.
func NewConfirm(
	telegramBot pluginapi.TelegramBot,
	apiClientSvc service.APIClientService,
) *Confirm {
	return &Confirm{
		telegramBot:  telegramBot,
		apiClientSvc: apiClientSvc,
	}
}

// ID returns the unique identifier for the step.
func (s *Confirm) ID() string { return stepConfirm }

// OnEnter is called when the step is entered. It shows a confirmation message with the selected parking lot and type,
// and asks the user to confirm starting the parking session.
func (s *Confirm) OnEnter(
	ctx *conversationapi.Context,
	_ telego.Update,
) error {
	lot, _ := ctx.Data["lot_number"].(string)
	parkingType, _ := ctx.Data["parking_type"].(string)

	typeLabel := parkingTypeLabel(parkingType)

	keyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Yes, start parking").
				WithCallbackData(stepConfirm),
			tu.InlineKeyboardButton("Cancel").
				WithCallbackData(callbackCancel),
		),
	)

	text := fmt.Sprintf(
		"You're about to start <b>%s</b> parking "+
			"at Lot <b>%s</b>.\n\nWould you like to continue?",
		typeLabel,
		lot,
	)

	return editCtrlMessageWithKeyboard(
		s.telegramBot, ctx, text, keyboard,
	)
}

// OnMessage is called when a message or callback query is received while this step is active.
// It handles the user's response to the confirmation prompt.
func (s *Confirm) OnMessage(
	ctx *conversationapi.Context,
	update telego.Update,
) (string, error) {
	if update.CallbackQuery == nil {
		return stepConfirm, nil
	}

	if err := answerCallback(s.telegramBot, update); err != nil {
		return "", err
	}

	if update.CallbackQuery.Data != stepConfirm {
		_ = editCtrlMessage(
			s.telegramBot, ctx, "Parking cancelled.",
		)

		return "", nil
	}

	lot, _ := ctx.Data["lot_number"].(string)
	parkingType, _ := ctx.Data["parking_type"].(string)

	session, err := s.apiClientSvc.StartParking(
		context.Background(), lot, parkingType,
	)
	if err != nil {
		_ = editCtrlMessage(
			s.telegramBot, ctx,
			"Failed to start parking. Please try again.",
		)

		return "", fmt.Errorf("starting parking session: %w", err)
	}

	_ = editCtrlMessage(
		s.telegramBot, ctx,
		fmt.Sprintf(
			"Parking started!\n\n"+
				"Lot: <b>%s</b>\n"+
				"Address: <b>%s</b>\n"+
				"Started at: <b>%s</b>",
			session.ParkingPlace.UniqueNumber,
			session.ParkingPlace.Address,
			session.StartDate,
		),
	)

	return "", nil
}

func parkingTypeLabel(parkingType string) string {
	labels := map[string]string{
		parkingTypeOnlyFree:   "Free 15 minutes",
		parkingTypeBoth:       "Free 15 minutes + Paid time",
		parkingTypeOnlyPriced: "Paid time only",
	}

	if label, ok := labels[parkingType]; ok {
		return label
	}

	return parkingType
}
