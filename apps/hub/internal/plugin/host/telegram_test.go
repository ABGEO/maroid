package host_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"sync"
	"testing"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/plugin/host"
)

const (
	botToken       = "123456:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	maxUploadBytes = 1 << 20
)

type photoUpload struct {
	method string
	photo  []byte
}

// PARKMAP-SC-024: The host sends the photo of a plugin with sendPhoto.
func TestTelegramBotSendsPhoto(t *testing.T) {
	t.Parallel()

	var (
		mu      sync.Mutex
		uploads []photoUpload
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upload := photoUpload{method: path.Base(r.URL.Path)}
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

		//nolint:gosec // MaxBytesReader on the line above bounds the body.
		if err := r.ParseMultipartForm(maxUploadBytes); err == nil {
			if file, _, fileErr := r.FormFile("photo"); fileErr == nil {
				upload.photo, _ = io.ReadAll(file)
				_ = file.Close()
			}
		}

		mu.Lock()

		uploads = append(uploads, upload)

		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"result": map[string]any{
				"message_id": 1, "date": 0, "chat": map[string]any{"id": 7, "type": "private"},
			},
		})
	}))
	t.Cleanup(server.Close)

	bot, err := telego.NewBot(botToken, telego.WithAPIServer(server.URL))
	require.NoError(t, err)

	plugin := bind(t, host.Dependencies{TelegramBot: bot}, "dev.maroid.parking")
	telegramBot, err := plugin.TelegramBot()
	require.NoError(t, err)

	photo := []byte{0x89, 0x50, 0x4e}
	_, err = telegramBot.SendPhoto(
		t.Context(),
		tu.Photo(tu.ID(7), tu.FileFromBytes(photo, "map.png")),
	)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, uploads, 1)
	require.Equal(t, "sendPhoto", uploads[0].method)
	require.Equal(t, photo, uploads[0].photo)
}
