package pluginmsg_test

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/telegram/pluginmsg"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
	messageapi "github.com/abgeo/maroid/libs/pluginapi/telegram/message"
)

const (
	answerCallbackQuery = "answerCallbackQuery"
	selectedWorkspace   = "01998aa0-2222-7000-8000-0000000000aa"
	routedWithin        = 5 * time.Second
)

var errOfPlugin = errors.New("the plugin failed")

// received is one call that the plugin P received, with the scope it ran in.
type received struct {
	press     *messageapi.Press
	reply     *messageapi.Reply
	user      string
	workspace string
	role      pluginapi.Role
}

// pluginOfMessages is the plugin P. It records each call, and answers the notice and
// the error that the test sets.
type pluginOfMessages struct {
	mu     sync.Mutex
	calls  []received
	notice string
	err    error
}

func (p *pluginOfMessages) OnTelegramPress(
	ctx context.Context,
	press messageapi.Press,
) (string, error) {
	p.record(ctx, received{press: &press})

	p.mu.Lock()
	defer p.mu.Unlock()

	return p.notice, p.err
}

func (p *pluginOfMessages) OnTelegramReply(
	ctx context.Context,
	reply messageapi.Reply,
) error {
	p.record(ctx, received{reply: &reply})

	p.mu.Lock()
	defer p.mu.Unlock()

	return p.err
}

func (p *pluginOfMessages) answer(notice string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.notice, p.err = notice, err
}

func (p *pluginOfMessages) record(ctx context.Context, call received) {
	call.user = pluginapi.ActingUserFromContext(ctx)
	call.workspace = pluginapi.ActingWorkspaceFromContext(ctx)
	call.role = workspace.RoleFromContext(ctx)

	p.mu.Lock()
	defer p.mu.Unlock()

	p.calls = append(p.calls, call)
}

func (p *pluginOfMessages) received() []received {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]received(nil), p.calls...)
}

// conversations holds the Telegram users with an open conversation.
type conversations struct {
	mu   sync.Mutex
	open map[string]bool
}

func (c *conversations) Holds(telegramUserID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.open[telegramUserID]
}

func (c *conversations) begin(telegramUserID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.open[strconv.FormatInt(telegramUserID, 10)] = true
}

// routing runs the router in a bot handler. A first middleware sets the acting user
// of the sender and the workspace that the chat selected, as the middlewares of the
// hub do. The last handler records each update that reaches it.
type routing struct {
	*scene

	plugin        *pluginOfMessages
	conversations *conversations
	updates       chan telego.Update
	done          chan struct{}
	nextID        atomic.Int64

	mu      sync.Mutex
	reached []telego.Update
}

func newRouting(t *testing.T, scn *scene) *routing {
	t.Helper()

	receivers := registry.NewTelegramMessageRegistry()
	plugin := &pluginOfMessages{}
	receivers.Register(pluginP, plugin)

	rtg := &routing{
		scene:         scn,
		plugin:        plugin,
		conversations: &conversations{open: map[string]bool{}},
		updates:       make(chan telego.Update),
		done:          make(chan struct{}),
	}

	router := pluginmsg.NewRouter(
		slog.New(slog.DiscardHandler), scn.instance.DB, scn.bot, receivers, rtg.conversations,
	)

	handler, err := th.NewBotHandler(scn.bot, rtg.updates)
	require.NoError(t, err)

	handler.Use(func(ctx *th.Context, update telego.Update) error {
		defer func() { rtg.done <- struct{}{} }()

		return ctx.Next(update)
	})
	handler.Use(rtg.acting())
	handler.Use(router.Middleware())
	handler.Handle(router.Fallback(rtg.last))

	go func() { _ = handler.Start() }()

	t.Cleanup(func() { close(rtg.updates) })

	return rtg
}

// acting answers the middleware that stands for the middlewares of the hub.
func (r *routing) acting() th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		return ctx.WithContext(r.scopeOf(ctx, update)).Next(update)
	}
}

func (r *routing) scopeOf(ctx context.Context, update telego.Update) context.Context {
	users := map[int64]string{telegramOfNina: r.nina, telegramOfBeka: r.beka}

	var sender int64

	switch {
	case update.CallbackQuery != nil:
		sender = update.CallbackQuery.From.ID
	case update.Message != nil:
		sender = update.Message.From.ID
	}

	scoped := pluginapi.ContextWithActingWorkspace(
		pluginapi.ContextWithActingUser(ctx, users[sender]),
		selectedWorkspace,
	)

	return workspace.ContextWithRole(scoped, pluginapi.RoleManager)
}

func (r *routing) last(_ *th.Context, update telego.Update) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reached = append(r.reached, update)

	return nil
}

func (r *routing) reachedNext() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.reached)
}

func (r *routing) route(t *testing.T, update telego.Update) {
	t.Helper()

	update.UpdateID = int(r.nextID.Add(1))

	select {
	case r.updates <- update:
	case <-time.After(routedWithin):
		t.Fatal("the bot handler takes no update")
	}

	select {
	case <-r.done:
	case <-time.After(routedWithin):
		t.Fatal("the update was not routed")
	}
}

// press is a press of the sender on a button of the message in the chat of Nina.
func (r *routing) press(t *testing.T, sender, messageID int64, value string) {
	t.Helper()

	r.route(t, telego.Update{CallbackQuery: &telego.CallbackQuery{
		ID:   "press-" + strconv.FormatInt(r.nextID.Load()+1, 10),
		From: telego.User{ID: sender},
		Message: &telego.Message{
			MessageID: int(messageID),
			Chat:      telego.Chat{ID: telegramOfNina, Type: telego.ChatTypePrivate},
		},
		Data: value,
	}})
}

// say is a message of Nina in the private chat. A reply names the message that it
// answers.
func (r *routing) say(t *testing.T, message telego.Message) {
	t.Helper()

	message.MessageID = int(9000 + r.nextID.Load())
	message.From = &telego.User{ID: telegramOfNina}
	message.Chat = telego.Chat{ID: telegramOfNina, Type: telego.ChatTypePrivate}

	r.route(t, telego.Update{Message: &message})
}

func replyTo(messageID int64) *telego.Message {
	return &telego.Message{
		MessageID: int(messageID),
		Chat:      telego.Chat{ID: telegramOfNina, Type: telego.ChatTypePrivate},
	}
}

// telegramIDOf answers the identifier that Telegram gave the plugin message.
func (s *scene) telegramIDOf(t *testing.T, user string, id messageapi.ID) int64 {
	t.Helper()

	for _, row := range s.rows(t, user) {
		if row.ID == string(id) {
			return row.MessageID
		}
	}

	t.Fatalf("no row %s", id)

	return 0
}

func (r *routing) answers(method string) []string {
	calls := r.api.CallsOf(method)
	texts := make([]string, 0, len(calls))

	for _, call := range calls {
		texts = append(texts, call.Text())
	}

	return texts
}

func (r *routing) claim(t *testing.T, text string) messageapi.ID {
	t.Helper()

	return r.send(t, r.nina, pluginP, messageapi.Message{
		Text:            text,
		ClaimReplyUntil: time.Now().Add(time.Hour),
	})
}

// TGMSG-SC-009: A press reaches P with Nina as the acting user and no workspace,
// although the chat selected one.
func TestAPressReachesThePluginThatSentTheMessage(t *testing.T) {
	t.Parallel()

	rtg := newRouting(t, newScene(t))
	mood := rtg.send(t, rtg.nina, pluginP, messageapi.Message{
		Text:    textMood,
		Buttons: [][]messageapi.Button{{{Label: "4", Value: "4"}}},
	})

	rtg.press(t, telegramOfNina, rtg.telegramIDOf(t, rtg.nina, mood), "4")

	calls := rtg.plugin.received()
	require.Len(t, calls, 1)
	require.NotNil(t, calls[0].press)
	assert.Equal(t, messageapi.Press{Message: mood, Value: "4"}, *calls[0].press)
	assert.Equal(t, rtg.nina, calls[0].user)
	assert.Empty(t, calls[0].workspace)
	assert.Empty(t, calls[0].role)
	assert.Zero(t, rtg.reachedNext())
}

// TGMSG-SC-010: The hub answers each press with the notice of P, or with the failure.
func TestTheHubAnswersEachPress(t *testing.T) {
	t.Parallel()

	rtg := newRouting(t, newScene(t))
	mood := rtg.send(t, rtg.nina, pluginP, messageapi.Message{
		Text:    textMood,
		Buttons: [][]messageapi.Button{{{Label: "4", Value: "4"}}},
	})
	messageID := rtg.telegramIDOf(t, rtg.nina, mood)

	rtg.plugin.answer("Saved", nil)
	rtg.press(t, telegramOfNina, messageID, "4")

	rtg.plugin.answer("", errOfPlugin)
	rtg.press(t, telegramOfNina, messageID, "4")

	assert.Equal(t, []string{"Saved", pluginmsg.AnswerFailed}, rtg.answers(answerCallbackQuery))
}

// TGMSG-SC-011: A press on the message of a plugin that the hub did not load, and a
// press on a message with no row, each read that the button no longer works.
func TestAPressThatReachesNoPluginIsAnswered(t *testing.T) {
	t.Parallel()

	rtg := newRouting(t, newScene(t))
	ofQ := rtg.send(t, rtg.nina, pluginQ, messageapi.Message{
		Text:    "Q asks",
		Buttons: [][]messageapi.Button{{{Label: "Yes", Value: "yes"}}},
	})

	rtg.press(t, telegramOfNina, rtg.telegramIDOf(t, rtg.nina, ofQ), "yes")
	rtg.press(t, telegramOfNina, 4242, "yes")

	assert.Equal(
		t,
		[]string{pluginmsg.AnswerButtonGone, pluginmsg.AnswerButtonGone},
		rtg.answers(answerCallbackQuery),
	)
	assert.Empty(t, rtg.plugin.received())
}

// TGMSG-FR-012: An open conversation of the person takes a press that no plugin
// message owns, so the hub does not answer it.
func TestAnOpenConversationTakesAnOrphanPress(t *testing.T) {
	t.Parallel()

	rtg := newRouting(t, newScene(t))
	rtg.conversations.begin(telegramOfNina)

	rtg.press(t, telegramOfNina, 4242, "zone-3")

	assert.Empty(t, rtg.answers(answerCallbackQuery))
	assert.Equal(t, 1, rtg.reachedNext())
}

// TGMSG-SC-012: Beka presses a button with the chat and the message of a message of
// Nina. The policy hides the row, so P receives nothing.
func TestAPressOfAnotherPersonReachesNoPlugin(t *testing.T) {
	t.Parallel()

	rtg := newRouting(t, newScene(t))
	mood := rtg.send(t, rtg.nina, pluginP, messageapi.Message{
		Text:    textMood,
		Buttons: [][]messageapi.Button{{{Label: "4", Value: "4"}}},
	})

	rtg.press(t, telegramOfBeka, rtg.telegramIDOf(t, rtg.nina, mood), "4")

	assert.Empty(t, rtg.plugin.received())
	assert.Equal(t, []string{pluginmsg.AnswerButtonGone}, rtg.answers(answerCallbackQuery))
}

// TGMSG-SC-013: A reply to a claimed message reaches P as a text, or as a location.
func TestAReplyReachesTheClaim(t *testing.T) {
	t.Parallel()

	rtg := newRouting(t, newScene(t))
	weight := rtg.claim(t, textWeight)
	place := rtg.send(t, rtg.nina, pluginP, messageapi.Message{
		Text: "Where are you?",
		Keyboard: [][]messageapi.Key{
			{{Label: "Send location", RequestLocation: true}},
		},
		ClaimReplyUntil: time.Now().Add(time.Hour),
	})

	rtg.say(
		t,
		telego.Message{
			Text:           answerWeight,
			ReplyToMessage: replyTo(rtg.telegramIDOf(t, rtg.nina, weight)),
		},
	)
	rtg.say(t, telego.Message{
		Location:       &telego.Location{Latitude: 41.7151, Longitude: 44.8271},
		ReplyToMessage: replyTo(rtg.telegramIDOf(t, rtg.nina, place)),
	})

	calls := rtg.plugin.received()
	require.Len(t, calls, 2)
	assert.Equal(t, messageapi.Reply{Message: weight, Text: answerWeight}, *calls[0].reply)
	assert.Equal(t, messageapi.Reply{
		Message:  place,
		Location: &messageapi.Location{Latitude: 41.7151, Longitude: 44.8271},
	}, *calls[1].reply)
	assert.Empty(t, calls[0].workspace)
}

// TGMSG-SC-014: A reply to a released message reaches no plugin, and the bot says
// that the message takes no answer.
func TestAReplyToAReleasedMessageIsRefused(t *testing.T) {
	t.Parallel()

	rtg := newRouting(t, newScene(t))
	weight := rtg.claim(t, textWeight)
	require.NoError(t, rtg.service.Release(rtg.as(t, rtg.nina), pluginP, weight))

	rtg.say(
		t,
		telego.Message{
			Text:           answerWeight,
			ReplyToMessage: replyTo(rtg.telegramIDOf(t, rtg.nina, weight)),
		},
	)

	assert.Empty(t, rtg.plugin.received())
	assert.Contains(t, rtg.answers(sendMessage), pluginmsg.AnswerNoClaim)
}

// TGMSG-SC-015: An open conversation takes a plain message before any claim.
func TestAnOpenConversationComesBeforeAClaim(t *testing.T) {
	t.Parallel()

	rtg := newRouting(t, newScene(t))
	rtg.claim(t, textWeight)
	rtg.conversations.begin(telegramOfNina)

	rtg.say(t, telego.Message{Text: "AB-123-CD"})

	assert.Equal(t, 1, rtg.reachedNext())
	assert.Empty(t, rtg.plugin.received())
}

// TGMSG-SC-016: A plain message reaches the oldest open claim. With no claim open, it
// passes on.
func TestAPlainMessageReachesTheOldestClaim(t *testing.T) {
	t.Parallel()

	rtg := newRouting(t, newScene(t))
	ctx := rtg.as(t, rtg.nina)
	first := rtg.claim(t, textWeight)
	second := rtg.claim(t, "Hours of sleep?")

	rtg.say(t, telego.Message{Text: answerWeight})
	require.NoError(t, rtg.service.Release(ctx, pluginP, first))
	rtg.say(t, telego.Message{Text: "73"})
	require.NoError(t, rtg.service.Release(ctx, pluginP, second))
	rtg.say(t, telego.Message{Text: "hello"})

	calls := rtg.plugin.received()
	require.Len(t, calls, 2)
	assert.Equal(t, messageapi.Reply{Message: first, Text: answerWeight}, *calls[0].reply)
	assert.Equal(t, messageapi.Reply{Message: second, Text: "73"}, *calls[1].reply)
	assert.Equal(t, 1, rtg.reachedNext())
}

// TGMSG-SC-017: A command passes on, past an open claim.
func TestACommandPassesAnOpenClaim(t *testing.T) {
	t.Parallel()

	rtg := newRouting(t, newScene(t))
	rtg.claim(t, textWeight)

	rtg.say(t, telego.Message{
		Text:     "/help",
		Entities: []telego.MessageEntity{{Type: "bot_command", Offset: 0, Length: 5}},
	})

	assert.Equal(t, 1, rtg.reachedNext())
	assert.Empty(t, rtg.plugin.received())
}

// TGMSG-SC-018: A second router over the same database reads the claim that the
// first service wrote.
func TestAClaimOutlivesTheRouter(t *testing.T) {
	t.Parallel()

	scn := newScene(t)
	weight := scn.send(t, scn.nina, pluginP, messageapi.Message{
		Text:            textWeight,
		ClaimReplyUntil: time.Now().Add(time.Hour),
	})

	rtg := newRouting(t, scn)
	rtg.say(t, telego.Message{Text: answerWeight})

	calls := rtg.plugin.received()
	require.Len(t, calls, 1)
	assert.Equal(t, messageapi.Reply{Message: weight, Text: answerWeight}, *calls[0].reply)
}

// TGMSG-FR-015: A plugin that fails on a reply leaves the person a message that asks
// to try again.
func TestAFailedReplyAsksToTryAgain(t *testing.T) {
	t.Parallel()

	rtg := newRouting(t, newScene(t))
	rtg.claim(t, textWeight)
	rtg.plugin.answer("", errOfPlugin)

	rtg.say(t, telego.Message{Text: answerWeight})

	assert.Contains(t, rtg.answers(sendMessage), pluginmsg.AnswerFailed)
}
