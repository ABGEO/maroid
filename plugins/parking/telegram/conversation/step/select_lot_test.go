package step_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/dto"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/model"
	"github.com/abgeo/maroid/plugins/parking/service"
	"github.com/abgeo/maroid/plugins/parking/telegram/conversation/step"
)

const (
	locationLat  = 41.708870
	locationLng  = 44.764440
	chatID       = 7
	threadID     = 3
	topicID      = 11
	tileSide     = 256
	foundText    = "I found a few parking lots near you."
	noLotText    = "No parking lots found near your location."
	mapTimeout   = 3 * time.Second
	nearestLot   = "A306"
	manualButton = "Enter manually"
	cancelButton = "Cancel"
)

// fakeFinder answers the lots that the test gives, after the delay.
type fakeFinder struct {
	lots  []model.NearbyLot
	err   error
	delay time.Duration
}

func (f *fakeFinder) FindNearby(context.Context, float64, float64) ([]model.NearbyLot, error) {
	time.Sleep(f.delay)

	return f.lots, f.err
}

// fakeRenderer records the lots of each map, and answers a PNG of one byte.
type fakeRenderer struct {
	calls [][]model.NearbyLot
}

func (r *fakeRenderer) Render(
	_ context.Context,
	_, _ float64,
	lots []model.NearbyLot,
) ([]byte, error) {
	r.calls = append(r.calls, lots)

	return []byte{1}, nil
}

func lotsNamed(numbers ...string) []model.NearbyLot {
	lots := make([]model.NearbyLot, 0, len(numbers))
	for _, number := range numbers {
		lots = append(lots, model.NearbyLot{Number: number})
	}

	return lots
}

// fixtureLots reads the lots of the parking service near the location.
func fixtureLots(t *testing.T) []model.NearbyLot {
	t.Helper()

	raw, err := os.ReadFile("../../../service/testdata/paliashvili.json")
	require.NoError(t, err)

	var answer dto.APIResponse[[]dto.ParkingLot]
	require.NoError(t, json.Unmarshal(raw, &answer))

	lots := make([]model.NearbyLot, 0, len(answer.Result.Data))

	for _, lot := range answer.Result.Data {
		parkingLot, convErr := lot.ToModel()
		require.NoError(t, convErr)

		lots = append(lots, model.NearbyLot{ParkingLot: parkingLot})
	}

	return lots
}

// mapSource plays the map source. A status other than 200 answers with no tile, and a
// negative delay never answers.
func mapSource(t *testing.T, status int, delay time.Duration) string {
	t.Helper()

	var tile bytes.Buffer
	require.NoError(t, png.Encode(&tile, image.NewRGBA(image.Rect(0, 0, tileSide, tileSide))))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if delay < 0 {
			<-r.Context().Done()

			return
		}

		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return
		}

		if status != http.StatusOK {
			w.WriteHeader(status)

			return
		}

		_, _ = w.Write(tile.Bytes())
	}))
	t.Cleanup(server.Close)

	return server.URL
}

func realLotMap(t *testing.T, url string) *service.LotMap {
	t.Helper()

	cfg := config.MapConfig{
		TileURL:     url + "/{z}/{x}/{y}.png",
		Attribution: "© OpenStreetMap contributors",
		MaxZoom:     19,
		Timeout:     mapTimeout,
	}

	lotMap, err := service.NewLotMap(
		service.NewTileClient(cfg),
		service.NewTileCache(50<<20, time.Hour),
		cfg,
	)
	require.NoError(t, err)

	return lotMap
}

func locationContext(t *testing.T) *conversationapi.Context {
	t.Helper()

	return &conversationapi.Context{
		Context: t.Context(),
		Data:    map[string]any{"latitude": locationLat, "longitude": locationLng},
	}
}

func locationUpdate() telego.Update {
	return telego.Update{Message: &telego.Message{
		Chat:                telego.Chat{ID: chatID},
		MessageThreadID:     threadID,
		DirectMessagesTopic: &telego.DirectMessagesTopic{TopicID: topicID},
		Location:            &telego.Location{Latitude: locationLat, Longitude: locationLng},
	}}
}

type selectLotCase struct {
	bot      *fakeBot
	finder   *fakeFinder
	renderer service.LotMapRenderer
	timeout  time.Duration
	logger   *slog.Logger
}

func (c selectLotCase) enter(t *testing.T) error {
	t.Helper()

	if c.timeout == 0 {
		c.timeout = mapTimeout
	}

	if c.logger == nil {
		c.logger = slog.New(slog.DiscardHandler)
	}

	selectLot := step.NewSelectLot(
		c.bot, &fakeClient{}, c.finder, c.renderer, c.timeout, newNotice(), c.logger,
	)

	if err := selectLot.OnEnter(locationContext(t), locationUpdate()); err != nil {
		return fmt.Errorf("entering the lot step: %w", err)
	}

	return nil
}

func lotList(t *testing.T, bot *fakeBot) *telego.SendMessageParams {
	t.Helper()

	require.NotEmpty(t, bot.messages)

	return bot.messages[len(bot.messages)-1]
}

func buttonsOf(params *telego.SendMessageParams) []string {
	keyboard, _ := params.ReplyMarkup.(*telego.InlineKeyboardMarkup)

	var labels []string

	for _, row := range keyboard.InlineKeyboard {
		for _, button := range row {
			labels = append(labels, button.Text)
		}
	}

	return labels
}

// PARKMAP-SC-008: The map comes before the lot list, and both carry the same lots.
func TestSelectLotSendsTheMapBeforeTheList(t *testing.T) {
	t.Parallel()

	bot := &fakeBot{}
	renderer := &fakeRenderer{}
	lots := lotsNamed(nearestLot, "A302")

	require.NoError(
		t,
		selectLotCase{bot: bot, finder: &fakeFinder{lots: lots}, renderer: renderer}.enter(t),
	)

	assert.Equal(t, []string{"SendMessage", "SendPhoto", "SendMessage"}, bot.calls)
	require.Len(t, renderer.calls, 1)
	assert.Equal(t, lots, renderer.calls[0])
	assert.Equal(
		t,
		[]string{nearestLot, "A302", manualButton, cancelButton},
		buttonsOf(lotList(t, bot)),
	)
}

// PARKMAP-SC-009: One lot gives one map and one lot button.
func TestSelectLotSendsOneMapForOneLot(t *testing.T) {
	t.Parallel()

	bot := &fakeBot{}

	require.NoError(t, selectLotCase{
		bot: bot, finder: &fakeFinder{lots: lotsNamed(nearestLot)}, renderer: &fakeRenderer{},
	}.enter(t))

	assert.Len(t, bot.photos, 1)
	assert.Equal(t, []string{nearestLot, manualButton, cancelButton}, buttonsOf(lotList(t, bot)))
}

// PARKMAP-SC-010: No lot gives no map.
func TestSelectLotSendsNoMapForNoLot(t *testing.T) {
	t.Parallel()

	bot := &fakeBot{}
	renderer := &fakeRenderer{}

	require.NoError(t, selectLotCase{bot: bot, finder: &fakeFinder{}, renderer: renderer}.enter(t))

	assert.Empty(t, renderer.calls)
	assert.Empty(t, bot.photos)
	assert.Contains(t, lotList(t, bot).Text, noLotText)
	assert.Equal(t, []string{manualButton, cancelButton}, buttonsOf(lotList(t, bot)))
}

// PARKMAP-SC-011: A failed map source gives the lot list alone, with no note.
func TestSelectLotSendsTheListWhenTheMapSourceFails(t *testing.T) {
	t.Parallel()

	bot := &fakeBot{}
	renderer := realLotMap(t, mapSource(t, http.StatusServiceUnavailable, 0))

	require.NoError(t, selectLotCase{
		bot: bot, finder: &fakeFinder{lots: fixtureLots(t)}, renderer: renderer,
	}.enter(t))

	assert.Empty(t, bot.photos)
	assert.True(t, strings.HasPrefix(lotList(t, bot).Text, foundText))
	assert.NotContains(t, strings.ToLower(lotList(t, bot).Text), "map")
}

// PARKMAP-SC-012: A slow map source gives the lot list alone, at the time limit.
func TestSelectLotSendsTheListAtTheTimeLimit(t *testing.T) {
	t.Parallel()

	bot := &fakeBot{}
	renderer := realLotMap(t, mapSource(t, http.StatusOK, 2*time.Second))
	start := time.Now()

	require.NoError(t, selectLotCase{
		bot:      bot,
		finder:   &fakeFinder{lots: fixtureLots(t)},
		renderer: renderer,
		timeout:  300 * time.Millisecond,
	}.enter(t))

	assert.Empty(t, bot.photos)
	assert.Less(t, bot.sentAt[len(bot.sentAt)-1].Sub(start), 500*time.Millisecond)
}

// PARKMAP-SC-013: A failure of the parking service makes no map.
func TestSelectLotMakesNoMapWhenTheSearchFails(t *testing.T) {
	t.Parallel()

	renderer := &fakeRenderer{}

	err := selectLotCase{
		bot: &fakeBot{}, finder: &fakeFinder{err: errs.ErrRequestFailed}, renderer: renderer,
	}.enter(t)

	require.ErrorIs(t, err, errs.ErrRequestFailed)
	assert.Empty(t, renderer.calls)
}

// PARKMAP-SC-014: A failed photo send still gives the lot list.
func TestSelectLotSendsTheListWhenThePhotoFails(t *testing.T) {
	t.Parallel()

	bot := &fakeBot{photoErr: errs.ErrRequestFailed}

	require.NoError(t, selectLotCase{
		bot: bot, finder: &fakeFinder{lots: lotsNamed(nearestLot)}, renderer: &fakeRenderer{},
	}.enter(t))

	assert.Equal(t, []string{nearestLot, manualButton, cancelButton}, buttonsOf(lotList(t, bot)))
}

// PARKMAP-SC-015: A failed map leaves one warning with the reason and no location.
func TestSelectLotLogsAFailedMap(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer

	renderer := realLotMap(t, mapSource(t, http.StatusServiceUnavailable, 0))

	require.NoError(t, selectLotCase{
		bot:      &fakeBot{},
		finder:   &fakeFinder{lots: fixtureLots(t)},
		renderer: renderer,
		logger:   slog.New(slog.NewJSONHandler(&logs, nil)),
	}.enter(t))

	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	require.Len(t, lines, 1)

	var record map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &record))

	assert.Equal(t, "WARN", record["level"])
	assert.Equal(t, "parking map not sent", record["msg"])
	assert.Contains(t, record["error"], "503")
	assert.InDelta(t, 2, record["lots"], 0)
	assert.NotContains(t, logs.String(), "41.70887")
	assert.NotContains(t, logs.String(), "44.76444")
}

// PARKMAP-SC-025: The map goes to the chat, the thread, and the topic of the update.
func TestSelectLotSendsTheMapToTheThreadOfTheUpdate(t *testing.T) {
	t.Parallel()

	bot := &fakeBot{}

	require.NoError(t, selectLotCase{
		bot: bot, finder: &fakeFinder{lots: lotsNamed(nearestLot)}, renderer: &fakeRenderer{},
	}.enter(t))

	require.Len(t, bot.photos, 1)
	assert.Equal(t, int64(chatID), bot.photos[0].ChatID.ID)
	assert.Equal(t, threadID, bot.photos[0].MessageThreadID)
	assert.Equal(t, int64(topicID), bot.photos[0].DirectMessagesTopicID)
}

// PARKMAP-SC-026: The lot list arrives less than 5 seconds after the location.
func TestSelectLotSendsTheListWithin5Seconds(t *testing.T) {
	t.Parallel()

	bot := &fakeBot{}
	renderer := realLotMap(t, mapSource(t, http.StatusOK, -1))
	start := time.Now()

	require.NoError(t, selectLotCase{
		bot:      bot,
		finder:   &fakeFinder{lots: fixtureLots(t), delay: 900 * time.Millisecond},
		renderer: renderer,
	}.enter(t))

	assert.Less(t, bot.sentAt[len(bot.sentAt)-1].Sub(start), 5*time.Second)
}
