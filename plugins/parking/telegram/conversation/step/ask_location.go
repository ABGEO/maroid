package step

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
	conversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
)

// AskLocation is a conversation step that prompts the user to share their location.
type AskLocation struct {
	telegramBot pluginapi.TelegramBot
}

var _ conversationapi.Step = (*AskLocation)(nil)

// NewAskLocation creates a new AskLocation step.
func NewAskLocation(telegramBot pluginapi.TelegramBot) *AskLocation {
	return &AskLocation{
		telegramBot: telegramBot,
	}
}

// ID returns the unique identifier for the step.
func (s *AskLocation) ID() string { return stepAskLocation }

// OnEnter is called when the step is entered. It prompts the user to share their location.
func (s *AskLocation) OnEnter(
	_ *conversationapi.Context,
	update telego.Update,
) error {
	keyboard := tu.Keyboard(
		tu.KeyboardRow(
			tu.KeyboardButton("Share my location").WithRequestLocation(),
		),
		tu.KeyboardRow(
			tu.KeyboardButton("Enter manually"),
		),
		tu.KeyboardRow(
			tu.KeyboardButton("Cancel"),
		),
	).WithResizeKeyboard().WithOneTimeKeyboard()

	message := tu.Message(
		tu.ID(update.Message.Chat.ID),
		"To start parking, I need to know where you are."+
			"\n\nPlease share your current location "+
			"so I can find the nearest parking lot.",
	).
		WithMessageThreadID(update.Message.MessageThreadID).
		WithReplyMarkup(keyboard)

	if update.Message.DirectMessagesTopic != nil {
		message.WithDirectMessagesTopicID(
			int(update.Message.DirectMessagesTopic.TopicID),
		)
	}

	if _, err := s.telegramBot.SendMessage(
		context.Background(), message,
	); err != nil {
		return fmt.Errorf("sending location prompt: %w", err)
	}

	return nil
}

// OnMessage is called when a message is received while this step is active.
// It checks if the user shared their location and saves it in the context data.
func (s *AskLocation) OnMessage(
	ctx *conversationapi.Context,
	update telego.Update,
) (string, error) {
	if update.Message == nil {
		return stepAskLocation, nil
	}

	if update.Message.Text == "Cancel" {
		msg := tu.Message(
			tu.ID(update.Message.Chat.ID),
			"Parking cancelled.",
		).
			WithMessageThreadID(update.Message.MessageThreadID).
			WithReplyMarkup(tu.ReplyKeyboardRemove())

		if _, err := s.telegramBot.SendMessage(
			context.Background(), msg,
		); err != nil {
			return "", fmt.Errorf("sending cancel message: %w", err)
		}

		return "", nil
	}

	if update.Message.Text == "Enter manually" {
		sent, err := s.sendEnterManuallyPrompt(update)
		if err != nil {
			return "", err
		}

		ctx.Data["ctrl_msg_id"] = sent.MessageID
		ctx.Data["ctrl_chat_id"] = sent.Chat.ID

		return stepEnterLot, nil
	}

	if update.Message.Location == nil {
		return stepAskLocation, nil
	}

	ctx.Data["latitude"] = update.Message.Location.Latitude
	ctx.Data["longitude"] = update.Message.Location.Longitude

	return stepSelectLot, nil
}

func (s *AskLocation) sendEnterManuallyPrompt(update telego.Update) (*telego.Message, error) {
	dismiss := tu.Message(
		tu.ID(update.Message.Chat.ID),
		"Entering parking lot number manually.",
	).
		WithMessageThreadID(update.Message.MessageThreadID).
		WithReplyMarkup(tu.ReplyKeyboardRemove())

	if update.Message.DirectMessagesTopic != nil {
		dismiss.WithDirectMessagesTopicID(
			int(update.Message.DirectMessagesTopic.TopicID),
		)
	}

	if _, err := s.telegramBot.SendMessage(context.Background(), dismiss); err != nil {
		return nil, fmt.Errorf("dismissing reply keyboard: %w", err)
	}

	ctrl := tu.Message(
		tu.ID(update.Message.Chat.ID),
		"Loading...",
	).
		WithMessageThreadID(update.Message.MessageThreadID).
		WithReplyMarkup(cancelKeyboard())

	if update.Message.DirectMessagesTopic != nil {
		ctrl.WithDirectMessagesTopicID(
			int(update.Message.DirectMessagesTopic.TopicID),
		)
	}

	sent, err := s.telegramBot.SendMessage(context.Background(), ctrl)
	if err != nil {
		return nil, fmt.Errorf("sending enter manually prompt: %w", err)
	}

	return sent, nil
}
