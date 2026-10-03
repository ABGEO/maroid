package command

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// WorkspaceCallbackPrefix starts the data of a button that picks a workspace.
const WorkspaceCallbackPrefix = "workspace:"

// ErrNoMessage reports an update that carries no message to answer.
var ErrNoMessage = errors.New("the update carries no message")

// WorkspaceChats keeps the workspace that each chat of the acting user acts in.
// workspace.ChatSelection satisfies it.
type WorkspaceChats interface {
	Acting(ctx context.Context, chatID int64) (string, error)
	Choices(ctx context.Context) ([]model.Workspace, error)
	Select(ctx context.Context, chatID int64, workspaceID string) (*model.Workspace, error)
}

// Prompter asks the person to pick the workspace of the chat.
type Prompter interface {
	Prompt(ctx *th.Context, update telego.Update, text string) error
}

// Workspace is the command that answers one button for each workspace of the person.
type Workspace struct {
	chats WorkspaceChats
}

var (
	_ pluginapi.TelegramCommand = (*Workspace)(nil)
	_ Prompter                  = (*Workspace)(nil)
)

// NewWorkspace creates a new Workspace command.
func NewWorkspace(chats WorkspaceChats) *Workspace {
	return &Workspace{chats: chats}
}

// Meta returns the metadata for the command.
func (c *Workspace) Meta() pluginapi.TelegramCommandMeta {
	return pluginapi.TelegramCommandMeta{
		Command:     "workspace",
		Description: "Pick the workspace of this chat",
	}
}

// Validate checks that the update carries a message to answer.
func (c *Workspace) Validate(update telego.Update) error {
	if update.Message == nil {
		return ErrNoMessage
	}

	return nil
}

// Handle answers the buttons of the workspaces of the person.
func (c *Workspace) Handle(ctx *th.Context, update telego.Update) error {
	return c.Prompt(ctx, update, "Pick the workspace of this chat")
}

// Prompt answers the text with one button for each workspace of the person.
func (c *Workspace) Prompt(ctx *th.Context, update telego.Update, text string) error {
	if update.Message == nil {
		return ErrNoMessage
	}

	choices, err := c.chats.Choices(ctx)
	if err != nil {
		return fmt.Errorf("listing the workspaces to pick: %w", err)
	}

	if len(choices) == 0 {
		return sendMessage(ctx, update, "You are a member of no workspace yet.")
	}

	rows := make([][]telego.InlineKeyboardButton, 0, len(choices))
	for _, choice := range choices {
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(choice.Name).
				WithCallbackData(WorkspaceCallbackPrefix+choice.ID),
		))
	}

	message := tu.Message(tu.ID(update.Message.Chat.ID), text).
		WithMessageThreadID(update.Message.MessageThreadID).
		WithReplyMarkup(tu.InlineKeyboard(rows...))

	if _, err = ctx.Bot().SendMessage(ctx, message); err != nil {
		return fmt.Errorf("sending the workspace buttons: %w", err)
	}

	return nil
}

// WorkspaceSelect handles a tap on a button of Workspace.
type WorkspaceSelect struct {
	chats WorkspaceChats
}

// NewWorkspaceSelect creates a new WorkspaceSelect handler.
func NewWorkspaceSelect(chats WorkspaceChats) *WorkspaceSelect {
	return &WorkspaceSelect{chats: chats}
}

// Handle stores the picked workspace for the chat, and names it in the message.
func (h *WorkspaceSelect) Handle(ctx *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	if query == nil || query.Message == nil {
		return nil
	}

	chatID := query.Message.GetChat().ID
	workspaceID := strings.TrimPrefix(query.Data, WorkspaceCallbackPrefix)

	picked, err := h.pick(ctx, chatID, workspaceID)
	if errors.Is(err, errs.ErrMemberNotFound) {
		return answer(ctx, query.ID, "That workspace is no longer yours")
	}

	if err != nil {
		return err
	}

	if err = answer(ctx, query.ID, ""); err != nil {
		return err
	}

	_, err = ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
		ChatID:    tu.ID(chatID),
		MessageID: query.Message.GetMessageID(),
		Text:      "This chat now works in " + picked.Name,
	})
	if err != nil {
		return fmt.Errorf("naming the picked workspace: %w", err)
	}

	return nil
}

// pick reads data that a client can forge, so an identifier that is not a UUID
// answers as a workspace of no membership.
func (h *WorkspaceSelect) pick(
	ctx context.Context,
	chatID int64,
	workspaceID string,
) (*model.Workspace, error) {
	if uuid.Validate(workspaceID) != nil {
		return nil, errs.ErrMemberNotFound
	}

	picked, err := h.chats.Select(ctx, chatID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("picking the workspace: %w", err)
	}

	return picked, nil
}

func answer(ctx *th.Context, queryID string, text string) error {
	params := tu.CallbackQuery(queryID)
	if text != "" {
		params = params.WithText(text)
	}

	if err := ctx.Bot().AnswerCallbackQuery(ctx, params); err != nil {
		return fmt.Errorf("answering the tap: %w", err)
	}

	return nil
}
