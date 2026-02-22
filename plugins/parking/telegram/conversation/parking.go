package conversation

import (
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
	"github.com/abgeo/maroid/plugins/parking/service"
	"github.com/abgeo/maroid/plugins/parking/telegram/conversation/step"
)

// ParkingConversation implements the conversation flow for starting a parking session with the Telegram bot.
type ParkingConversation struct {
	steps map[string]conversation.Step
}

var _ conversation.Conversation = (*ParkingConversation)(nil)

// NewParkingConversation creates a new ParkingConversation.
func NewParkingConversation(
	telegramBot pluginapi.TelegramBot,
	apiClientSvc service.APIClientService,
) *ParkingConversation {
	steps := []conversation.Step{
		step.NewSelectLot(telegramBot, apiClientSvc),
		step.NewAskLocation(telegramBot),
		step.NewConfirm(telegramBot, apiClientSvc),
		step.NewEnterLot(telegramBot, apiClientSvc),
		step.NewSelectType(telegramBot),
	}

	stepMap := make(map[string]conversation.Step, len(steps))
	for _, s := range steps {
		stepMap[s.ID()] = s
	}

	return &ParkingConversation{
		steps: stepMap,
	}
}

// ID returns the unique identifier for the conversation.
func (c *ParkingConversation) ID() string {
	return "start_parking"
}

// Entry returns the ID of the first step in the conversation.
func (c *ParkingConversation) Entry() string {
	return "ask_location"
}

// Steps returns a map of step IDs to their corresponding Step implementations.
func (c *ParkingConversation) Steps() map[string]conversation.Step {
	return c.steps
}
