package service_test

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/service"
)

func pngOf(t *testing.T, side int) []byte {
	t.Helper()

	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, image.NewGray(image.Rect(0, 0, side, side))))

	return encoded.Bytes()
}

// PARKMAP-SC-031: A tile over 1 MB, of the wrong size, or not a PNG fails the map and
// stays out of the cache.
func TestRenderRejectsABadTile(t *testing.T) {
	t.Parallel()

	bodies := map[string][]byte{
		"too large": bytes.Repeat([]byte{0}, 2<<20),
		"too wide":  pngOf(t, 2*tileSide),
		"not a png": []byte("<html>blocked</html>"),
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write(body)
				}),
			)
			t.Cleanup(server.Close)

			cfg := mapConfig(server.URL)
			cache := service.NewTileCache(cacheLimit, cacheTTL)

			lotMap, err := service.NewLotMap(service.NewTileClient(cfg), cache, cfg)
			require.NoError(t, err)

			_, err = lotMap.Render(t.Context(), originLat, originLng, fixtureLots(t))
			require.ErrorIs(t, err, errs.ErrMapSource)
			assert.Zero(t, cache.Size())
		})
	}
}

// PARKMAP-SC-034: A failed tile request names no address of the map source.
func TestTileErrorNamesNoAddress(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hijacker, _ := w.(http.Hijacker)
		conn, _, _ := hijacker.Hijack()
		_ = conn.Close()
	}))
	t.Cleanup(server.Close)

	cfg := mapConfig(server.URL)
	cfg.TileURL += "?key=secret"

	_, err := service.NewTileClient(cfg).Tile(t.Context(), 19, 1, 2)
	require.ErrorIs(t, err, errs.ErrMapSource)
	assert.NotContains(t, err.Error(), "secret")
	assert.NotContains(t, err.Error(), server.Listener.Addr().String())
}
