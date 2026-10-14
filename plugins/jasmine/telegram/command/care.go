package command

import (
	"context"
	"fmt"
	"html"
	"unicode/utf16"

	"github.com/jmoiron/sqlx"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/jasmine/digest"
	"github.com/abgeo/maroid/plugins/jasmine/model"
	"github.com/abgeo/maroid/plugins/jasmine/permission"
	"github.com/abgeo/maroid/plugins/jasmine/repository"
)

const (
	// messageLimit is the length that Telegram accepts for the text of one message.
	messageLimit = 4096
	// messageCap keeps one answer from flooding the chat, since Telegram accepts
	// about one message each second in one chat.
	messageCap = 3

	attentionPath  = "/attention"
	nothingDueText = "No plant needs care."
)

// Care is a Telegram command that answers the attention list of the acting workspace.
type Care struct {
	db       *pluginapi.PluginDB
	bot      pluginapi.TelegramBot
	pages    pluginapi.PageLinker
	pluginID *pluginapi.PluginID
}

var _ pluginapi.TelegramCommand = (*Care)(nil)

// NewCare creates a new Care.
func NewCare(
	db *pluginapi.PluginDB,
	bot pluginapi.TelegramBot,
	pages pluginapi.PageLinker,
	pluginID *pluginapi.PluginID,
) *Care {
	return &Care{db: db, bot: bot, pages: pages, pluginID: pluginID}
}

// Meta returns the metadata for the command.
func (c *Care) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "care",
		Description: "Plants that need care",
		Permission:  permission.PlantsRead,
	}
}

// Validate checks if the update is valid for this command.
func (c *Care) Validate(_ telego.Update) error {
	return nil
}

// Handle processes the command.
func (c *Care) Handle(ctx *th.Context, update telego.Update) error {
	return c.Reply(ctx, update.Message)
}

// Reply sends the attention list of the acting workspace to the chat of the message.
// A failed send stops the reply, so no later part arrives without the part before it.
func (c *Care) Reply(ctx context.Context, message *telego.Message) error {
	var intervals []model.CareInterval

	err := c.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		var listErr error

		intervals, listErr = repository.NewCareInterval(tx).ListAttention(ctx)
		if listErr != nil {
			return fmt.Errorf("listing the care that is due: %w", listErr)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("reading the care list: %w", err)
	}

	texts := []string{nothingDueText}
	if len(intervals) > 0 {
		texts = pack(digest.Blocks(intervals), messageLimit, messageCap, c.more(ctx))
	}

	for _, text := range texts {
		if err = c.send(ctx, message, text); err != nil {
			return fmt.Errorf("sending the care list: %w", err)
		}
	}

	return nil
}

// more counts the lines that the answer leaves out, and links to the attention list,
// which shows every line. A host that cannot build the address gets the name of the
// page, so the answer still tells the person where to go.
func (c *Care) more(ctx context.Context) func(left int) string {
	address, err := c.pages.PageURL(ctx, c.pluginID, attentionPath)

	return func(left int) string {
		count := fmt.Sprintf("And %d more lines.", left)
		if err != nil {
			return count + " Open Needs attention in Maroid."
		}

		return count + ` <a href="` + html.EscapeString(address) + `">Open Needs attention</a>.`
	}
}

func (c *Care) send(ctx context.Context, message *telego.Message, text string) error {
	params := tu.Message(tu.ID(message.Chat.ID), text).
		WithMessageThreadID(message.MessageThreadID).
		WithParseMode(telego.ModeHTML)

	if message.DirectMessagesTopic != nil {
		params.WithDirectMessagesTopicID(message.DirectMessagesTopic.TopicID)
	}

	if _, err := c.bot.SendMessage(ctx, params); err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	return nil
}

// pack lays the blocks out as digest.Body does, under the title, and divides the text
// into messages of at most limit UTF-16 code units. Telegram counts the text after it
// parses the HTML, which is never longer, so the bound holds. A message divides only
// between two lines, and a message that continues a block starts with its heading.
// The last of maxMessages messages keeps room for the text of more, and ends with it
// when a line is left out.
func pack(blocks []digest.Block, limit, maxMessages int, more func(left int) string) []string {
	total := 0
	for _, block := range blocks {
		total += len(block.Lines)
	}

	reserve := unitsOf("\n\n" + more(total))

	var messages []string

	current := "<b>" + digest.Title + "</b>"
	units := unitsOf(current)
	placed := 0

	for _, block := range blocks {
		separator := "\n\n" + block.Heading + "\n"

		for _, line := range block.Lines {
			isLast := len(messages) == maxMessages-1

			room := limit
			if isLast && placed < total-1 {
				room -= reserve
			}

			if units+unitsOf(separator+line) > room {
				if isLast {
					return append(messages, current+"\n\n"+more(total-placed))
				}

				messages = append(messages, current)
				current, units = "", 0
				separator = block.Heading + "\n"
			}

			current += separator + line
			units += unitsOf(separator + line)
			separator = "\n"
			placed++
		}
	}

	return append(messages, current)
}

func unitsOf(text string) int {
	units := 0
	for _, r := range text {
		units += utf16.RuneLen(r)
	}

	return units
}
