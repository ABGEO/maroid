package step

import (
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
	conversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
	"github.com/abgeo/maroid/plugins/parking/model"
)

// SelectVehicle is a conversation step that asks the member to choose one vehicle of
// the parking account, when the account holds two or more.
type SelectVehicle struct {
	telegramBot pluginapi.TelegramBot
}

var _ conversationapi.Step = (*SelectVehicle)(nil)

// NewSelectVehicle creates a new SelectVehicle step.
func NewSelectVehicle(telegramBot pluginapi.TelegramBot) *SelectVehicle {
	return &SelectVehicle{
		telegramBot: telegramBot,
	}
}

// ID returns the unique identifier for the step.
func (s *SelectVehicle) ID() string { return stepSelectVehicle }

// OnEnter shows one button for each vehicle that the previous step stored.
func (s *SelectVehicle) OnEnter(
	ctx *conversationapi.Context,
	_ telego.Update,
) error {
	vehicles, _ := ctx.Data[dataVehicles].([]model.Vehicle)

	rows := make([][]telego.InlineKeyboardButton, 0, len(vehicles)+1)
	for _, vehicle := range vehicles {
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(vehicleLabel(vehicle)).
				WithCallbackData(callbackVehiclePrefix+strconv.Itoa(vehicle.ID)),
		))
	}

	rows = append(rows, tu.InlineKeyboardRow(
		tu.InlineKeyboardButton("Cancel").WithCallbackData(callbackCancel),
	))

	return editCtrlMessageWithKeyboard(
		s.telegramBot, ctx, "Which vehicle do you park?", tu.InlineKeyboard(rows...),
	)
}

// OnMessage accepts a vehicle of the stored list, or the cancel button.
func (s *SelectVehicle) OnMessage(
	ctx *conversationapi.Context,
	update telego.Update,
) (string, error) {
	if update.CallbackQuery == nil {
		return stepSelectVehicle, nil
	}

	if err := answerCallback(s.telegramBot, update); err != nil {
		return "", err
	}

	data := update.CallbackQuery.Data
	if data == callbackCancel {
		return "", editCtrlMessage(s.telegramBot, ctx, "Parking cancelled.")
	}

	vehicle, found := chosenVehicle(ctx, data)
	if !found {
		return stepSelectVehicle, nil
	}

	selectVehicle(ctx, vehicle)

	return stepConfirm, nil
}

// chosenVehicle finds the vehicle of the button in the stored list. A button that
// names no vehicle of the list chooses nothing.
func chosenVehicle(ctx *conversationapi.Context, data string) (model.Vehicle, bool) {
	id, err := strconv.Atoi(strings.TrimPrefix(data, callbackVehiclePrefix))
	if err != nil || !strings.HasPrefix(data, callbackVehiclePrefix) {
		return model.Vehicle{}, false
	}

	vehicles, _ := ctx.Data[dataVehicles].([]model.Vehicle)
	for _, vehicle := range vehicles {
		if vehicle.ID == id {
			return vehicle, true
		}
	}

	return model.Vehicle{}, false
}
