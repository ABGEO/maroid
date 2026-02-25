package conversation

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/mymmrac/telego"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	telegramupdate "github.com/abgeo/maroid/apps/hub/internal/telegram/update"
	telegramconversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
)

// Engine is responsible for managing Telegram conversations, including handling incoming messages
// and starting new conversations.
type Engine struct {
	registry *registry.TelegramConversationRegistry
	store    telegramconversationapi.Store
	logger   *slog.Logger
}

var _ telegramconversationapi.Engine = (*Engine)(nil)

// NewEngine creates a new Engine with the given conversation registry and state store.
func NewEngine(
	registry *registry.TelegramConversationRegistry,
	store telegramconversationapi.Store,
	logger *slog.Logger,
) *Engine {
	return &Engine{
		registry: registry,
		store:    store,
		logger:   logger,
	}
}

// HandleMessage processes an incoming Telegram update, updates the conversation state accordingly,
// and triggers the appropriate step handlers.
// If there is no active conversation for the user, it simply returns without doing anything.
//
//nolint:funlen // sequential conversation state machine; splitting would obscure the flow
func (e *Engine) HandleMessage(update telego.Update) error {
	userID := strconv.FormatInt(telegramupdate.SentFrom(update).ID, 10)

	state, _ := e.store.Get(userID)
	if state == nil {
		return nil // no active conversation
	}

	convo := e.registry.Get(state.ConversationID)
	if convo == nil {
		return fmt.Errorf(
			"getting conversation %s: %w",
			state.ConversationID,
			errs.ErrTelegramConversationNotFound,
		)
	}

	step := e.registry.Step(state.ConversationID, state.StepID)
	if step == nil {
		return fmt.Errorf(
			"getting step %s for conversation %s: %w",
			state.StepID,
			state.ConversationID,
			errs.ErrTelegramConversationStepNotFound,
		)
	}

	ctx := &telegramconversationapi.Context{
		UserID:         userID,
		ConversationID: state.ConversationID,
		Data:           state.Data,
	}

	next, err := step.OnMessage(ctx, update)
	if err != nil {
		e.logger.Error(
			"processing conversation step failed",
			slog.String("conversation_id", state.ConversationID),
			slog.String("step_id", state.StepID),
			slog.String("user_id", userID),
			slog.Any("error", err),
		)

		return nil
	}

	if next == "" {
		if err = e.store.Clear(userID); err != nil {
			return fmt.Errorf("clearing conversation state: %w", err)
		}

		return nil
	}

	state.StepID = next
	state.Data = ctx.Data

	err = e.store.Save(state)
	if err != nil {
		return fmt.Errorf("storing conversation state: %w", err)
	}

	nextStep := e.registry.Step(state.ConversationID, next)
	if nextStep == nil {
		return fmt.Errorf(
			"getting step %s for conversation %s: %w",
			next,
			state.ConversationID,
			errs.ErrTelegramConversationStepNotFound,
		)
	}

	err = nextStep.OnEnter(ctx, update)
	if err != nil {
		return fmt.Errorf("entering step %s: %w", next, err)
	}

	return nil
}

// Start initiates a new conversation for the user based on the provided conversation ID
// and the incoming Telegram update.
func (e *Engine) Start(update telego.Update, conversationID string) error {
	userID := strconv.FormatInt(telegramupdate.SentFrom(update).ID, 10)

	convo := e.registry.Get(conversationID)
	if convo == nil {
		return fmt.Errorf(
			"getting conversation %s: %w",
			conversationID,
			errs.ErrTelegramConversationNotFound,
		)
	}

	entry := convo.Entry()

	state := &telegramconversationapi.State{
		UserID:         userID,
		ConversationID: conversationID,
		StepID:         entry,
		Data:           map[string]any{},
	}

	err := e.store.Save(state)
	if err != nil {
		return fmt.Errorf("storing conversation state: %w", err)
	}

	ctx := &telegramconversationapi.Context{
		UserID:         userID,
		ConversationID: conversationID,
		Data:           state.Data,
	}

	entryStep := e.registry.Step(conversationID, entry)
	if entryStep == nil {
		return fmt.Errorf(
			"getting step %s for conversation %s: %w",
			entry,
			conversationID,
			errs.ErrTelegramConversationStepNotFound,
		)
	}

	err = entryStep.OnEnter(ctx, update)
	if err != nil {
		return fmt.Errorf("entering step %s: %w", entry, err)
	}

	return nil
}
