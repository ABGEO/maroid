package step_test

import (
	"context"
	"testing"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	conversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
	"github.com/abgeo/maroid/plugins/parking/dto"
	"github.com/abgeo/maroid/plugins/parking/model"
	"github.com/abgeo/maroid/plugins/parking/service"
	"github.com/abgeo/maroid/plugins/parking/telegram/conversation/step"
	"github.com/abgeo/maroid/plugins/parking/telegram/notice"
)

const (
	typeOnlyPriced = "ONLY_PRICED_PARKING"
	stepConfirm    = "confirm"
	stepVehicle    = "select_vehicle"
	lotNumber      = "A123"
	accountURL     = "https://deck.example/w/W/plugins/dev.maroid.parking/account"
)

// fakeBot records the text and the keyboard of each edit of the control message.
type fakeBot struct {
	texts     []string
	keyboards []*telego.InlineKeyboardMarkup
}

func (b *fakeBot) SendMessage(context.Context, *telego.SendMessageParams) (*telego.Message, error) {
	return &telego.Message{}, nil
}

func (b *fakeBot) AnswerCallbackQuery(context.Context, *telego.AnswerCallbackQueryParams) error {
	return nil
}

func (b *fakeBot) EditMessageText(
	_ context.Context,
	params *telego.EditMessageTextParams,
) (*telego.Message, error) {
	b.texts = append(b.texts, params.Text)
	b.keyboards = append(b.keyboards, params.ReplyMarkup)

	return &telego.Message{}, nil
}

// fakeClient answers the vehicles that the test gives, and records the vehicle of a start.
type fakeClient struct {
	service.APIClientService

	vehicles      []model.Vehicle
	vehicleErr    error
	vehicleReads  int
	startedWithID int
}

func (c *fakeClient) ListVehicles(context.Context) ([]model.Vehicle, error) {
	c.vehicleReads++

	return c.vehicles, c.vehicleErr
}

func (c *fakeClient) StartParking(
	_ context.Context,
	_ string,
	_ string,
	vehicleID int,
) (*dto.ParkingSession, error) {
	c.startedWithID = vehicleID

	return &dto.ParkingSession{}, nil
}

// fakePages answers one address for every page.
type fakePages struct{}

func (fakePages) PageURL(context.Context, *pluginapi.PluginID, string) (string, error) {
	return accountURL, nil
}

func newNotice() *notice.Notice {
	return notice.New(fakePages{}, pluginapi.ParsePluginID("dev.maroid.parking"))
}

func newContext() *conversationapi.Context {
	return &conversationapi.Context{
		Context: context.Background(),
		Data: map[string]any{
			"ctrl_msg_id":  1,
			"ctrl_chat_id": int64(2),
			"lot_number":   lotNumber,
		},
	}
}

func press(data string) telego.Update {
	return telego.Update{CallbackQuery: &telego.CallbackQuery{ID: "q", Data: data}}
}

func chevyAndFiat() []model.Vehicle {
	return []model.Vehicle{
		{ID: 1, Name: "Chevy", Plate: "AB000EO"},
		{ID: 2, Name: "Fiat", Plate: "CD111EF"},
	}
}

// PARKACC-SC-023: One vehicle goes to the confirmation with no question, and the
// start carries its identifier.
func TestOneVehicleIsSelectedWithNoQuestion(t *testing.T) {
	t.Parallel()

	bot := &fakeBot{}
	client := &fakeClient{vehicles: []model.Vehicle{{ID: 584842, Name: "Chevy", Plate: "AB000EO"}}}
	ctx := newContext()

	next, err := step.NewSelectType(bot, client, newNotice()).OnMessage(ctx, press(typeOnlyPriced))
	require.NoError(t, err)
	require.Equal(t, stepConfirm, next)
	require.Equal(t, 1, client.vehicleReads)

	confirm := step.NewConfirm(bot, client, newNotice())
	require.NoError(t, confirm.OnEnter(ctx, telego.Update{}))
	require.Contains(t, bot.texts[len(bot.texts)-1], "Chevy (AB000EO)")

	_, err = confirm.OnMessage(ctx, press(stepConfirm))
	require.NoError(t, err)
	require.Equal(t, 584842, client.startedWithID)
}

// PARKACC-SC-024: Two vehicles ask for a choice, and the start carries the chosen one.
// An identifier outside the list keeps the step.
func TestTwoVehiclesAskForAChoice(t *testing.T) {
	t.Parallel()

	bot := &fakeBot{}
	client := &fakeClient{vehicles: chevyAndFiat()}
	ctx := newContext()

	next, err := step.NewSelectType(bot, client, newNotice()).OnMessage(ctx, press(typeOnlyPriced))
	require.NoError(t, err)
	require.Equal(t, stepVehicle, next)

	selectVehicle := step.NewSelectVehicle(bot)
	require.NoError(t, selectVehicle.OnEnter(ctx, telego.Update{}))

	keyboard := bot.keyboards[len(bot.keyboards)-1]
	require.Len(t, keyboard.InlineKeyboard, 3)
	require.Equal(t, "Chevy (AB000EO)", keyboard.InlineKeyboard[0][0].Text)
	require.Equal(t, "vehicle:1", keyboard.InlineKeyboard[0][0].CallbackData)
	require.Equal(t, "Fiat (CD111EF)", keyboard.InlineKeyboard[1][0].Text)
	require.Equal(t, "Cancel", keyboard.InlineKeyboard[2][0].Text)

	next, err = selectVehicle.OnMessage(ctx, press("vehicle:3"))
	require.NoError(t, err)
	require.Equal(t, stepVehicle, next)

	next, err = selectVehicle.OnMessage(ctx, press("vehicle:2"))
	require.NoError(t, err)
	require.Equal(t, stepConfirm, next)

	_, err = step.NewConfirm(bot, client, newNotice()).OnMessage(ctx, press(stepConfirm))
	require.NoError(t, err)
	require.Equal(t, 2, client.startedWithID)
}

// PARKACC-SC-025: A parking account with no vehicle ends the conversation with the reason.
func TestNoVehicleEndsTheConversation(t *testing.T) {
	t.Parallel()

	bot := &fakeBot{}
	client := &fakeClient{}

	next, err := step.NewSelectType(bot, client, newNotice()).
		OnMessage(newContext(), press(typeOnlyPriced))

	require.NoError(t, err)
	require.Empty(t, next)
	require.Contains(t, bot.texts[len(bot.texts)-1], "no vehicle")
	require.Zero(t, client.startedWithID)
}

// PARKACC-SC-018: A conversation whose token the parking service rejects ends with the
// answer of a user with no connection, and reports no failure.
func TestARejectedTokenEndsTheConversationWithTheLink(t *testing.T) {
	t.Parallel()

	bot := &fakeBot{}
	client := &fakeClient{vehicleErr: service.ErrNotConnected}

	next, err := step.NewSelectType(bot, client, newNotice()).
		OnMessage(newContext(), press(typeOnlyPriced))

	require.NoError(t, err)
	require.Empty(t, next)
	require.Contains(t, bot.texts[len(bot.texts)-1], `href="`+accountURL+`"`)
}
