package command

import (
	"context"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/pulse/permission"
)

const resumedText = "Check-ins resumed."

// Resume is the command /resume, which ends the pause.
type Resume struct {
	pauser Pauser
	bot    pluginapi.TelegramBot
}

var _ pluginapi.TelegramCommand = (*Resume)(nil)

// NewResume creates a Resume.
func NewResume(pauser Pauser, bot pluginapi.TelegramBot) *Resume {
	return &Resume{pauser: pauser, bot: bot}
}

// Meta names the command.
func (r *Resume) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "resume",
		Description: "End the pause of the check-ins",
		Scope:       tu.ScopeAllPrivateChats(),
		Permission:  permission.CheckinsUse,
	}
}

// Validate accepts every update.
func (r *Resume) Validate(_ telego.Update) error {
	return nil
}

// Handle answers the command in its chat.
func (r *Resume) Handle(ctx *th.Context, update telego.Update) error {
	text, err := r.Reply(ctx)
	if err != nil {
		return err
	}

	return answer(ctx, r.bot, update.Message, text)
}

// Reply ends the pause and answers the text for the person.
func (r *Resume) Reply(ctx context.Context) (string, error) {
	if err := r.pauser.Resume(ctx); err != nil {
		return offOr(err)
	}

	return resumedText, nil
}
