// Package command provides an interface and related functionality for defining and handling
// Telegram bot commands using the telego library.
package command

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

// Command defines the interface for a Telegram bot command.
type Command interface {
	// Command returns the bot command definition.
	Command() telego.BotCommand
	// Scope returns the scope in which the command is applicable.
	Scope() telego.BotCommandScope
	// Validate checks if the update is valid for this command.
	Validate(update telego.Update) error
	// Handle processes the help command.
	Handle(ctx *th.Context, update telego.Update) error
}
