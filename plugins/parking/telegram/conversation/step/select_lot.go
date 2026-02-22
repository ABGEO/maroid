package step

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
	conversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
	"github.com/abgeo/maroid/plugins/parking/dto"
	"github.com/abgeo/maroid/plugins/parking/service"
)

// bboxOffset defines the approximate bounding box offset in degrees
// around the user's location (~10m).
const bboxOffset = 0.0001

// SelectLot is a conversation step that allows the user to select a parking lot from a list of nearby lots.
type SelectLot struct {
	telegramBot  pluginapi.TelegramBot
	apiClientSvc service.APIClientService
}

var _ conversationapi.Step = (*SelectLot)(nil)

// NewSelectLot creates a new SelectLot step.
func NewSelectLot(
	telegramBot pluginapi.TelegramBot,
	apiClientSvc service.APIClientService,
) *SelectLot {
	return &SelectLot{
		telegramBot:  telegramBot,
		apiClientSvc: apiClientSvc,
	}
}

// ID returns the unique identifier for the step.
func (s *SelectLot) ID() string { return stepSelectLot }

// OnEnter is called when the step is entered. It fetches nearby parking lots based on the user's location
// and prompts the user to select one from the list.
func (s *SelectLot) OnEnter(
	ctx *conversationapi.Context,
	update telego.Update,
) error {
	lat, _ := ctx.Data["latitude"].(float64)
	lng, _ := ctx.Data["longitude"].(float64)

	if err := s.dismissReplyKeyboard(update); err != nil {
		return err
	}

	lots, err := s.apiClientSvc.GetParkingLots(
		context.Background(),
		lng-bboxOffset, lng+bboxOffset,
		lat+bboxOffset, lat-bboxOffset,
	)
	if err != nil {
		return fmt.Errorf("fetching parking lots: %w", err)
	}

	keyboard := buildLotKeyboard(lots)
	text := buildLotMessage(lots)

	sent, err := s.sendLotSelection(update, text, keyboard)
	if err != nil {
		return err
	}

	ctx.Data["ctrl_msg_id"] = sent.MessageID
	ctx.Data["ctrl_chat_id"] = sent.Chat.ID

	return nil
}

// OnMessage is called when a message or callback query is received while this step is active.
func (s *SelectLot) OnMessage(
	ctx *conversationapi.Context,
	update telego.Update,
) (string, error) {
	if update.CallbackQuery == nil {
		return stepSelectLot, nil
	}

	if err := answerCallback(s.telegramBot, update); err != nil {
		return "", err
	}

	data := update.CallbackQuery.Data
	if data == "enter_manually" {
		return stepEnterLot, nil
	}

	if data == callbackCancel {
		err := editCtrlMessage(
			s.telegramBot, ctx, "Parking cancelled.",
		)

		return "", err
	}

	place, err := s.apiClientSvc.GetParkingPlace(
		context.Background(), string(data[0]), data[1:],
	)
	if err != nil {
		return "", fmt.Errorf("fetching parking place: %w", err)
	}

	ctx.Data["lot_number"] = data
	ctx.Data["free_parking"] = place.FreeParking

	return stepSelectType, nil
}

func (s *SelectLot) dismissReplyKeyboard(
	update telego.Update,
) error {
	msg := tu.Message(
		tu.ID(update.Message.Chat.ID),
		"Location received. Looking up nearby parking lots...",
	).
		WithMessageThreadID(update.Message.MessageThreadID).
		WithReplyMarkup(tu.ReplyKeyboardRemove())

	if update.Message.DirectMessagesTopic != nil {
		msg.WithDirectMessagesTopicID(
			int(update.Message.DirectMessagesTopic.TopicID),
		)
	}

	if _, err := s.telegramBot.SendMessage(
		context.Background(), msg,
	); err != nil {
		return fmt.Errorf("removing reply keyboard: %w", err)
	}

	return nil
}

func buildLotKeyboard(
	lots []dto.ParkingLot,
) *telego.InlineKeyboardMarkup {
	extraRows := 2
	rows := make(
		[][]telego.InlineKeyboardButton,
		0, len(lots)+extraRows,
	)

	for _, lot := range lots {
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				lot.UniqueNumber,
			).WithCallbackData(lot.UniqueNumber),
		))
	}

	rows = append(rows,
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Enter manually").
				WithCallbackData("enter_manually"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Cancel").
				WithCallbackData(callbackCancel),
		),
	)

	return tu.InlineKeyboard(rows...)
}

func buildLotMessage(lots []dto.ParkingLot) string {
	if len(lots) == 0 {
		return "No parking lots found near your location." +
			"\n\nYou can <b>Enter manually</b> " +
			"if you know the lot number, or cancel."
	}

	return "I found a few parking lots near you." +
		"\n\nPlease select the correct parking lot " +
		"from the list below, or choose " +
		"<b>Enter manually</b> if it's not listed."
}

func (s *SelectLot) sendLotSelection(
	update telego.Update,
	text string,
	keyboard *telego.InlineKeyboardMarkup,
) (*telego.Message, error) {
	msg := tu.Message(
		tu.ID(update.Message.Chat.ID),
		text,
	).
		WithMessageThreadID(update.Message.MessageThreadID).
		WithReplyMarkup(keyboard).
		WithParseMode(telego.ModeHTML)

	if update.Message.DirectMessagesTopic != nil {
		msg.WithDirectMessagesTopicID(
			int(update.Message.DirectMessagesTopic.TopicID),
		)
	}

	sent, err := s.telegramBot.SendMessage(
		context.Background(), msg,
	)
	if err != nil {
		return nil, fmt.Errorf("sending control message: %w", err)
	}

	return sent, nil
}
