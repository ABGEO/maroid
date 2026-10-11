package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/notification"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	kindID   = "telegram"
	kindName = "Telegram"

	fieldBotToken = "bot_token"
	fieldChatID   = "chat_id"
	fieldThreadID = "thread_id"

	// callTimeout bounds one call to the Bot API, with its retries.
	callTimeout = 10 * time.Second

	retryMaxAttempts  = 4
	retryExponentBase = 2
	retryStartDelay   = time.Millisecond * 10
	retryMaxDelay     = time.Second
)

// ProviderFields declares the fields of a Telegram notification provider.
type ProviderFields struct {
	BotToken string `json:"bot_token" jsonschema:"title=Bot token,format=password,writeOnly=true,required"`
}

// DestinationFields declares the chat and the optional thread of a forum that receive
// a notification.
type DestinationFields struct {
	ChatID   string `json:"chat_id"   jsonschema:"title=Chat ID,pattern=^-?[0-9]+$,required"`
	ThreadID string `json:"thread_id" jsonschema:"title=Thread ID,pattern=^[0-9]*$"`
}

// Kind delivers a notification through a Telegram bot.
type Kind struct {
	options []telego.BotOption
}

var _ notification.Kind = (*Kind)(nil)

// New creates a Kind. The options reach every bot that the kind builds, so a test
// points them at a stub of the Bot API.
func New(options ...telego.BotOption) *Kind {
	return &Kind{options: options}
}

// ID returns the identifier of the kind.
func (k *Kind) ID() string {
	return kindID
}

// Name returns the name that a person reads for the kind.
func (k *Kind) Name() string {
	return kindName
}

// ProviderModel returns the fields of a notification provider.
func (k *Kind) ProviderModel() any {
	return ProviderFields{}
}

// DestinationModel returns the fields of a destination.
func (k *Kind) DestinationModel() any {
	return DestinationFields{}
}

// Verify calls getMe with the token. It changes nothing at Telegram.
func (k *Kind) Verify(ctx context.Context, provider map[string]any) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	bot, err := k.bot(provider)
	if err != nil {
		return err
	}

	if _, err = bot.GetMe(ctx); err != nil {
		return classify(err)
	}

	return nil
}

// Send delivers the message to the chat and the thread of the destination.
func (k *Kind) Send(
	ctx context.Context,
	provider map[string]any,
	destination map[string]any,
	msg pluginapi.Message,
) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	bot, err := k.bot(provider)
	if err != nil {
		return err
	}

	target, err := targetOf(destination)
	if err != nil {
		return err
	}

	return classify((&delivery{bot: bot, target: target}).send(ctx, msg))
}

func (k *Kind) bot(provider map[string]any) (*telego.Bot, error) {
	token, _ := provider[fieldBotToken].(string)

	options := append([]telego.BotOption{
		telego.WithDiscardLogger(),
		telego.WithAPICaller(&ta.RetryCaller{
			Caller:       ta.DefaultFastHTTPCaller,
			MaxAttempts:  retryMaxAttempts,
			ExponentBase: retryExponentBase,
			StartDelay:   retryStartDelay,
			MaxDelay:     retryMaxDelay,
		}),
	}, k.options...)

	bot, err := telego.NewBot(token, options...)
	if err != nil {
		return nil, &notification.RefusedError{Reason: "the bot token has no valid form"}
	}

	return bot, nil
}

// target is the chat, and the thread of a forum when it is not 0.
type target struct {
	chat   telego.ChatID
	thread int
}

func targetOf(destination map[string]any) (target, error) {
	chatText, _ := destination[fieldChatID].(string)

	chatID, err := strconv.ParseInt(chatText, 10, 64)
	if err != nil {
		return target{}, fmt.Errorf("%w: the chat ID %q", errs.ErrInvalidNotificationKind, chatText)
	}

	threadText, _ := destination[fieldThreadID].(string)
	if threadText == "" {
		return target{chat: tu.ID(chatID)}, nil
	}

	thread, err := strconv.Atoi(threadText)
	if err != nil {
		return target{}, fmt.Errorf(
			"%w: the thread ID %q",
			errs.ErrInvalidNotificationKind,
			threadText,
		)
	}

	return target{chat: tu.ID(chatID), thread: thread}, nil
}

// classify tells a refusal of Telegram from a call that got no answer. EXT-005
// makes ok: false an error, never an empty result.
func classify(err error) error {
	if err == nil {
		return nil
	}

	if apiErr, ok := errors.AsType[*ta.Error](err); ok {
		return &notification.RefusedError{Reason: apiErr.Description}
	}

	return fmt.Errorf("%w: %w", errs.ErrMessagingUnreachable, err)
}
