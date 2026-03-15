package command

import (
	"strings"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// Start represents the start command.
type Start struct {
	bot             *telego.Bot
	commandRegistry *registry.TelegramCommandRegistry
}

var _ pluginapi.TelegramCommand = (*Start)(nil)

// NewStart creates a new Start command.
func NewStart(bot *telego.Bot, commandRegistry *registry.TelegramCommandRegistry) *Start {
	return &Start{
		bot:             bot,
		commandRegistry: commandRegistry,
	}
}

// Meta returns the metadata for the command.
func (c *Start) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "start",
		Description: "Start interacting with the bot",
	}
}

// Validate checks if the update is valid for this command.
func (c *Start) Validate(_ telego.Update) error {
	return nil
}

// Handle processes the start command.
func (c *Start) Handle(ctx *th.Context, update telego.Update) error {
	_, _, args := tu.ParseCommand(update.Message.Text)
	if len(args) > 0 {
		return c.processArguments(ctx, update, args)
	}

	return sendMessage(c.bot, ctx, update, `Hello there 👋! I’m Maroid, your assistant for automating tasks.

Type /help to see what I can do and get started 🚀`)
}

func (c *Start) processArguments(ctx *th.Context, update telego.Update, args []string) error {
	cmd, ok := c.commandRegistry.Get(args[0])
	if !ok {
		return sendMessage(
			ctx.Bot(),
			ctx,
			update,
			"Sorry, I couldn't recognize the command you provided.\nPlease type /help to see the list of available commands.",
		)
	}

	// Rewrite message text so the routed command sees its own name.
	update.Message.Text = "/" + strings.Join(args, " ")

	if err := cmd.Validate(update); err != nil {
		return sendMessage(ctx.Bot(), ctx, update, err.Error())
	}

	return cmd.Handle(ctx, update)
}
