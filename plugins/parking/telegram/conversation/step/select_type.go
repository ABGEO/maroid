package step

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
	conversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
	"github.com/abgeo/maroid/plugins/parking/service"
	"github.com/abgeo/maroid/plugins/parking/telegram/notice"
)

// SelectType is a conversation step that prompts the user to select the parking type (free, paid, or both)
// after they have selected a parking lot.
type SelectType struct {
	telegramBot  pluginapi.TelegramBot
	apiClientSvc service.APIClientService
	answers      *notice.Notice
}

var _ conversationapi.Step = (*SelectType)(nil)

// NewSelectType creates a new SelectType step.
func NewSelectType(
	telegramBot pluginapi.TelegramBot,
	apiClientSvc service.APIClientService,
	answers *notice.Notice,
) *SelectType {
	return &SelectType{
		telegramBot:  telegramBot,
		apiClientSvc: apiClientSvc,
		answers:      answers,
	}
}

// ID returns the unique identifier for the step.
func (s *SelectType) ID() string { return stepSelectType }

// OnEnter is called when the step is entered.
// It prompts the user to select the parking type (free, paid, or both) using an inline keyboard.
func (s *SelectType) OnEnter(
	ctx *conversationapi.Context,
	update telego.Update,
) error {
	keyboard := buildTypeKeyboard(ctx)

	text := "Please choose how you want to start parking:"

	// The update comes from a text message (a manual lot entry).
	// Send a new message, so the old control message stays in place.
	if update.Message != nil {
		return s.sendNewCtrlMessage(ctx, update, text, keyboard)
	}

	return editCtrlMessageWithKeyboard(
		s.telegramBot, ctx, text, keyboard,
	)
}

// OnMessage is called when a message or callback query is received while this step is active.
func (s *SelectType) OnMessage(
	ctx *conversationapi.Context,
	update telego.Update,
) (string, error) {
	if update.CallbackQuery == nil {
		return stepSelectType, nil
	}

	if err := answerCallback(s.telegramBot, update); err != nil {
		return "", err
	}

	switch data := update.CallbackQuery.Data; data {
	case callbackCancel:
		err := editCtrlMessage(
			s.telegramBot, ctx, "Parking cancelled.",
		)

		return "", err
	default:
		ctx.Data["parking_type"] = data

		return s.routeToVehicle(ctx)
	}
}

// routeToVehicle reads the vehicles again, because the member can change them in the
// mobile application of the parking service after the start of the conversation.
func (s *SelectType) routeToVehicle(ctx *conversationapi.Context) (string, error) {
	vehicles, err := s.apiClientSvc.ListVehicles(ctx)
	if err != nil {
		if endWithNotice(s.telegramBot, ctx, s.answers, err) {
			return "", nil
		}

		_ = editCtrlMessage(s.telegramBot, ctx, "Failed to read the vehicles. Please try again.")

		return "", fmt.Errorf("reading the vehicles: %w", err)
	}

	switch len(vehicles) {
	case 0:
		return "", editCtrlMessage(s.telegramBot, ctx, noVehicleText)
	case 1:
		selectVehicle(ctx, vehicles[0])

		return stepConfirm, nil
	default:
		ctx.Data[dataVehicles] = vehicles

		return stepSelectVehicle, nil
	}
}

func buildTypeKeyboard(
	ctx *conversationapi.Context,
) *telego.InlineKeyboardMarkup {
	freeParking, _ := ctx.Data["free_parking"].(bool)

	var rows [][]telego.InlineKeyboardButton

	if freeParking {
		rows = append(rows,
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("Free 15 minutes").
					WithCallbackData(parkingTypeOnlyFree),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("Free 15 min + Paid time").
					WithCallbackData(parkingTypeBoth),
			),
		)
	}

	rows = append(rows,
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Only paid time").
				WithCallbackData(parkingTypeOnlyPriced),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Cancel").
				WithCallbackData(callbackCancel),
		),
	)

	return tu.InlineKeyboard(rows...)
}

func (s *SelectType) sendNewCtrlMessage(
	ctx *conversationapi.Context,
	update telego.Update,
	text string,
	keyboard *telego.InlineKeyboardMarkup,
) error {
	msg := tu.Message(
		tu.ID(update.Message.Chat.ID),
		text,
	).
		WithMessageThreadID(update.Message.MessageThreadID).
		WithReplyMarkup(keyboard).
		WithParseMode(telego.ModeHTML)

	if update.Message.DirectMessagesTopic != nil {
		msg.WithDirectMessagesTopicID(
			update.Message.DirectMessagesTopic.TopicID,
		)
	}

	sent, err := s.telegramBot.SendMessage(
		context.Background(), msg,
	)
	if err != nil {
		return fmt.Errorf("sending type selection message: %w", err)
	}

	ctx.Data["ctrl_msg_id"] = sent.MessageID
	ctx.Data["ctrl_chat_id"] = sent.Chat.ID

	return nil
}
