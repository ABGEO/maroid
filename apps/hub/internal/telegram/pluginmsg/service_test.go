package pluginmsg_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	messageapi "github.com/abgeo/maroid/libs/pluginapi/telegram/message"
)

const (
	sendMessage = "sendMessage"

	textMood     = "Mood?"
	textWeight   = "Weight in kg?"
	answerWeight = "72.5"
)

// TGMSG-SC-001, TGMSG-INV-001: A send for Nina reaches the chat of Nina, and the row names P.
func TestASendReachesTheChatOfTheActingUser(t *testing.T) {
	t.Parallel()

	sc := newScene(t)

	id := sc.send(t, sc.nina, pluginP, messageapi.Message{Text: "Good morning"})

	calls := sc.api.CallsOf(sendMessage)
	require.Len(t, calls, 1)
	assert.Equal(t, telegramOfNina, calls[0].ChatID())
	assert.Equal(t, "Good morning", calls[0].Text())

	rows := sc.rows(t, sc.nina)
	require.Len(t, rows, 1)
	assert.Equal(t, string(id), rows[0].ID)
	assert.Equal(t, pluginP, rows[0].PluginID)
	assert.Equal(t, telegramOfNina, rows[0].ChatID)
}

// TGMSG-SC-002: A send for Beka reaches Beka, and Nina reads no row of it.
func TestASendForAnotherPersonReachesThatPerson(t *testing.T) {
	t.Parallel()

	sc := newScene(t)

	sc.send(t, sc.beka, pluginP, messageapi.Message{Text: "Hi"})

	calls := sc.api.CallsOf(sendMessage)
	require.Len(t, calls, 1)
	assert.Equal(t, telegramOfBeka, calls[0].ChatID())
	assert.Empty(t, sc.rows(t, sc.nina))
	assert.Len(t, sc.rows(t, sc.beka), 1)
}

// TGMSG-SC-003: A unit of work with a workspace and no acting user sends nothing.
func TestASendWithNoActingUserIsRefused(t *testing.T) {
	t.Parallel()

	sc := newScene(t)
	ctx := pluginapi.ContextWithActingWorkspace(t.Context(), "01998aa0-2222-7000-8000-0000000000aa")

	_, err := sc.service.Send(ctx, pluginP, messageapi.Message{Text: "Hi"})

	require.ErrorIs(t, err, messageapi.ErrNoActingUser)
	assert.Empty(t, sc.api.Calls())
}

// TGMSG-SC-004: A person with a local account alone holds no Telegram account.
func TestASendToAPersonWithNoTelegramAccountIsRefused(t *testing.T) {
	t.Parallel()

	sc := newScene(t)

	_, err := sc.service.Send(sc.as(t, sc.zura), pluginP, messageapi.Message{Text: "Hi"})

	require.ErrorIs(t, err, messageapi.ErrNoAccount)
	assert.Empty(t, sc.api.Calls())
}

// TGMSG-SC-005: A blocked bot and a chat that never opened both read as unreachable.
func TestTelegramRefusingTheChatReadsAsUnreachable(t *testing.T) {
	t.Parallel()

	sc := newScene(t)
	sc.api.Refuse(telegramOfNina, 403, "Forbidden: bot was blocked by the user")
	sc.api.Refuse(telegramOfBeka, 400, "Bad Request: chat not found")

	_, ninaErr := sc.service.Send(sc.as(t, sc.nina), pluginP, messageapi.Message{Text: "Hi"})
	_, bekaErr := sc.service.Send(sc.as(t, sc.beka), pluginP, messageapi.Message{Text: "Hi"})

	require.ErrorIs(t, ninaErr, messageapi.ErrUnreachable)
	require.ErrorIs(t, bekaErr, messageapi.ErrUnreachable)
	assert.Empty(t, sc.rows(t, sc.nina))
	assert.Empty(t, sc.rows(t, sc.beka))
}

// TGMSG-FR-004: An error of Telegram other than a refused chat stays a failure.
func TestAnotherFailureOfTelegramIsNoRefusal(t *testing.T) {
	t.Parallel()

	sc := newScene(t)
	sc.api.Refuse(telegramOfNina, 400, "Bad Request: can't parse entities")

	_, err := sc.service.Send(sc.as(t, sc.nina), pluginP, messageapi.Message{Text: "Hi"})

	require.Error(t, err)
	assert.NotErrorIs(t, err, messageapi.ErrUnreachable)
}

// TGMSG-SC-006: Buttons become an inline keyboard, keys a reply keyboard that asks
// for a location, and RemoveKeyboard a markup that removes it.
func TestTheMarkupOfAPluginMessage(t *testing.T) {
	t.Parallel()

	sc := newScene(t)

	sc.send(t, sc.nina, pluginP, messageapi.Message{
		Text: textMood,
		Buttons: [][]messageapi.Button{
			{
				{Label: "1", Value: "1"},
				{Label: "2", Value: "2"},
				{Label: "3", Value: "3"},
				{Label: "4", Value: "4"},
				{Label: "5", Value: "5"},
			},
			{{Label: "Skip", Value: "skip"}},
		},
	})
	sc.send(t, sc.nina, pluginP, messageapi.Message{
		Text: "Where are you?",
		Keyboard: [][]messageapi.Key{
			{{Label: "Send location", RequestLocation: true}},
			{{Label: "Skip"}},
		},
	})
	sc.send(t, sc.nina, pluginP, messageapi.Message{Text: "Saved", RemoveKeyboard: true})

	calls := sc.api.CallsOf(sendMessage)
	require.Len(t, calls, 3)

	inline, _ := calls[0].Markup()["inline_keyboard"].([]any)
	require.Len(t, inline, 2)

	firstRow, _ := inline[0].([]any)
	require.Len(t, firstRow, 5)

	var values []string

	for _, row := range inline {
		buttons, _ := row.([]any)
		for _, button := range buttons {
			fields, _ := button.(map[string]any)
			value, _ := fields["callback_data"].(string)
			values = append(values, value)
		}
	}

	assert.Equal(t, []string{"1", "2", "3", "4", "5", "skip"}, values)

	keyboard := calls[1].Markup()
	rows, _ := keyboard["keyboard"].([]any)
	require.Len(t, rows, 2)

	locationRow, _ := rows[0].([]any)
	locationKey, _ := locationRow[0].(map[string]any)
	assert.Equal(t, true, locationKey["request_location"])
	assert.Equal(t, true, keyboard["one_time_keyboard"])

	assert.Equal(t, true, calls[2].Markup()["remove_keyboard"])
}

// TGMSG-SC-007, TGMSG-FR-013: A message that Telegram would refuse is refused before
// any call. A claim inside the lifetime succeeds.
func TestAnInvalidMessageIsRefusedBeforeTheCall(t *testing.T) {
	t.Parallel()

	sc := newScene(t)
	buttons := func(label, value string) [][]messageapi.Button {
		return [][]messageapi.Button{{{Label: label, Value: value}}}
	}
	yes := buttons("Yes", "yes")

	for name, msg := range map[string]messageapi.Message{
		"an empty text":          {Text: "  "},
		"a text past the limit":  {Text: strings.Repeat("a", messageapi.TextMax+1)},
		"a value of 65 bytes":    {Text: "Q", Buttons: buttons("A", strings.Repeat("v", 65))},
		"an empty value":         {Text: "Q", Buttons: buttons("A", "")},
		"an empty label":         {Text: "Q", Buttons: buttons("", "a")},
		"buttons and a keyboard": {Text: "Q", Buttons: yes, Keyboard: [][]messageapi.Key{{{Label: "K"}}}},
		"buttons and a removal":  {Text: "Q", Buttons: yes, RemoveKeyboard: true},
		"a claim 31 days away":   {Text: "Q", ClaimReplyUntil: time.Now().Add(31 * 24 * time.Hour)},
		"a claim in the past":    {Text: "Q", ClaimReplyUntil: time.Now().Add(-time.Minute)},
	} {
		_, err := sc.service.Send(sc.as(t, sc.nina), pluginP, msg)
		require.ErrorIs(t, err, messageapi.ErrInvalid, name)
	}

	assert.Empty(t, sc.api.Calls())

	sc.send(t, sc.nina, pluginP, messageapi.Message{
		Text:            textWeight,
		ClaimReplyUntil: time.Now().Add(29 * 24 * time.Hour),
	})

	rows := sc.rows(t, sc.nina)
	require.Len(t, rows, 1)
	assert.NotNil(t, rows[0].ClaimUntil)
}

// TGMSG-SC-008, TGMSG-INV-002: A plugin edits its own message for 30 days, and
// never the message of another plugin.
func TestAPluginEditsItsOwnMessage(t *testing.T) {
	t.Parallel()

	sc := newScene(t)
	ctx := sc.as(t, sc.nina)

	mood := sc.send(t, sc.nina, pluginP, messageapi.Message{
		Text:    textMood,
		Buttons: [][]messageapi.Button{{{Label: "4", Value: "4"}}},
	})
	ofQ := sc.send(t, sc.nina, pluginQ, messageapi.Message{Text: "Q asks"})

	require.NoError(t, sc.service.Edit(ctx, pluginP, mood, "Mood: 4", nil))

	edits := sc.api.CallsOf("editMessageText")
	require.Len(t, edits, 1)
	assert.Equal(t, "Mood: 4", edits[0].Text())
	assert.Equal(t, telegramOfNina, edits[0].ChatID())
	assert.Nil(t, edits[0].Markup(), "no buttons removes them")

	require.ErrorIs(
		t,
		sc.service.Edit(ctx, pluginP, ofQ, "taken", nil),
		messageapi.ErrUnknown,
	)

	sc.age(t, sc.nina)
	require.ErrorIs(
		t,
		sc.service.Edit(ctx, pluginP, mood, "late", nil),
		messageapi.ErrUnknown,
	)
	assert.Len(t, sc.api.CallsOf("editMessageText"), 1, "a refused edit calls nothing")
}

// TGMSG-FR-009: Telegram answers that the text did not change. The edit succeeds.
func TestAnEditThatChangesNothingSucceeds(t *testing.T) {
	t.Parallel()

	sc := newScene(t)
	id := sc.send(t, sc.nina, pluginP, messageapi.Message{Text: "Mood: 4"})
	sc.api.RefuseEdits(400, "Bad Request: message is not modified")

	require.NoError(t, sc.service.Edit(sc.as(t, sc.nina), pluginP, id, "Mood: 4", nil))
}

// TGMSG-FR-014: A plugin releases its own claim, and a release of a message that it
// did not send is refused.
func TestAPluginReleasesItsOwnClaim(t *testing.T) {
	t.Parallel()

	sc := newScene(t)
	ctx := sc.as(t, sc.nina)
	id := sc.send(t, sc.nina, pluginP, messageapi.Message{
		Text:            textWeight,
		ClaimReplyUntil: time.Now().Add(time.Hour),
	})

	require.ErrorIs(t, sc.service.Release(ctx, pluginQ, id), messageapi.ErrUnknown)
	require.NoError(t, sc.service.Release(ctx, pluginP, id))
	require.NoError(t, sc.service.Release(ctx, pluginP, id), "a second release succeeds")
	assert.Nil(t, sc.rows(t, sc.nina)[0].ClaimUntil)
}

// TGMSG-SC-019: A send deletes the rows of the acting user older than 30 days.
func TestASendPurgesTheOldRowsOfThePerson(t *testing.T) {
	t.Parallel()

	sc := newScene(t)

	sc.send(t, sc.nina, pluginP, messageapi.Message{Text: "old"})
	sc.send(t, sc.beka, pluginP, messageapi.Message{Text: "old"})
	sc.age(t, sc.nina)
	sc.age(t, sc.beka)

	sc.send(t, sc.nina, pluginP, messageapi.Message{Text: "new"})

	assert.Len(t, sc.rows(t, sc.nina), 1, "the old row of Nina is gone")
	assert.Len(t, sc.rows(t, sc.beka), 1, "the old row of Beka stays")
}
