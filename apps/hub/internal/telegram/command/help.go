package command

import (
	"fmt"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// Help represents the help command.
type Help struct {
	bot             *telego.Bot
	commandRegistry *registry.TelegramCommandRegistry
}

var _ pluginapi.TelegramCommand = (*Help)(nil)

// NewHelp creates a new Help command.
func NewHelp(bot *telego.Bot, commandRegistry *registry.TelegramCommandRegistry) *Help {
	return &Help{
		bot:             bot,
		commandRegistry: commandRegistry,
	}
}

// Meta returns the metadata for the command.
func (c *Help) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "help",
		Description: "Show help information",
	}
}

// Validate checks if the update is valid for this command.
func (c *Help) Validate(_ telego.Update) error {
	return nil
}

// Handle processes the help command.
func (c *Help) Handle(ctx *th.Context, update telego.Update) error {
	return sendMessage(c.bot, ctx, update, c.buildHelpMessage())
}

func (c *Help) buildHelpMessage() string {
	var textBuilder strings.Builder

	textBuilder.WriteString("Here’s what I can do for you 👇\n\n")

	for _, cmd := range c.commandRegistry.All() {
		meta := cmd.Meta()
		textBuilder.WriteString(fmt.Sprintf("/%s - %s\n", meta.Command, meta.Description))
	}

	return textBuilder.String()
}
