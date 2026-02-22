package step

import (
	"context"
	"errors"
	"fmt"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
	conversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
)

const (
	callbackCancel = "cancel"

	stepAskLocation = "ask_location"
	stepSelectLot   = "select_lot"
	stepEnterLot    = "enter_lot"
	stepSelectType  = "select_type"
	stepConfirm     = "confirm"
)

const (
	parkingTypeOnlyFree   = "ONLY_FREE_PARKING"
	parkingTypeBoth       = "BOTH"
	parkingTypeOnlyPriced = "ONLY_PRICED_PARKING"
)

var (
	errCtrlMsgIDMissing  = errors.New("ctrl_msg_id not found in context data")
	errCtrlChatIDMissing = errors.New("ctrl_chat_id not found in context data")
)

func editCtrlMessage(bot pluginapi.TelegramBot, ctx *conversationapi.Context, text string) error {
	msgID, chatID, err := ctrlMessageIDs(ctx)
	if err != nil {
		return err
	}

	params := tu.EditMessageText(tu.ID(chatID), msgID, text).
		WithParseMode(telego.ModeHTML)

	if _, err = bot.EditMessageText(context.Background(), params); err != nil {
		return fmt.Errorf("editing control message: %w", err)
	}

	return nil
}

func editCtrlMessageWithKeyboard(
	bot pluginapi.TelegramBot,
	ctx *conversationapi.Context,
	text string,
	keyboard *telego.InlineKeyboardMarkup,
) error {
	msgID, chatID, err := ctrlMessageIDs(ctx)
	if err != nil {
		return err
	}

	params := tu.EditMessageText(tu.ID(chatID), msgID, text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(keyboard)

	if _, err = bot.EditMessageText(context.Background(), params); err != nil {
		return fmt.Errorf("editing control message: %w", err)
	}

	return nil
}

func ctrlMessageIDs(ctx *conversationapi.Context) (int, int64, error) {
	msgID, ok := ctx.Data["ctrl_msg_id"].(int)
	if !ok {
		return 0, 0, errCtrlMsgIDMissing
	}

	chatID, ok := ctx.Data["ctrl_chat_id"].(int64)
	if !ok {
		return 0, 0, errCtrlChatIDMissing
	}

	return msgID, chatID, nil
}

func cancelKeyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Cancel").
				WithCallbackData(callbackCancel),
		),
	)
}

func answerCallback(bot pluginapi.TelegramBot, update telego.Update) error {
	if update.CallbackQuery == nil {
		return nil
	}

	if err := bot.AnswerCallbackQuery(
		context.Background(),
		tu.CallbackQuery(update.CallbackQuery.ID),
	); err != nil {
		return fmt.Errorf("answering callback query: %w", err)
	}

	return nil
}
