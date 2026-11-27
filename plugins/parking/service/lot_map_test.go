package service_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/dto"
	"github.com/abgeo/maroid/plugins/parking/model"
	"github.com/abgeo/maroid/plugins/parking/service"
)

const (
	tileSide     = 256
	colorSlack   = 8
	cacheLimit   = 50 << 20
	cacheTTL     = 7 * 24 * time.Hour
	attribution  = "© OpenStreetMap contributors"
	userAgent    = "Maroid/1.0 (parking map)"
	stripHeight  = 16
	darkChannel  = 128
	pinLift      = 18
	firstLotRGB  = 0x2563EB
	secondLotRGB = 0x16A34A
	pinRGB       = 0xDC2626
)

// tileServer plays the map source with a white tile, and records each request.
type tileServer struct {
	*httptest.Server

	mu       sync.Mutex
	counts   map[string]int
	requests []*http.Request
	status   int
}

func newTileServer(t *testing.T) *tileServer {
	t.Helper()

	var tile bytes.Buffer

	white := image.NewRGBA(image.Rect(0, 0, tileSide, tileSide))
	for i := range white.Pix {
		white.Pix[i] = 0xFF
	}

	require.NoError(t, png.Encode(&tile, white))

	server := &tileServer{counts: map[string]int{}, status: http.StatusOK}
	server.Server = httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			server.mu.Lock()

			server.counts[r.URL.Path]++
			server.requests = append(server.requests, r)
			status := server.status

			server.mu.Unlock()

			if status != http.StatusOK {
				w.WriteHeader(status)

				return
			}

			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(tile.Bytes())
		}),
	)
	t.Cleanup(server.Close)

	return server
}

func (s *tileServer) countsByPath() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return maps.Clone(s.counts)
}

func mapConfig(url string) config.MapConfig {
	return config.MapConfig{
		TileURL:     url + "/{z}/{x}/{y}.png",
		Attribution: attribution,
		MaxZoom:     19,
		Timeout:     3 * time.Second,
	}
}

func newLotMap(t *testing.T, server *tileServer) (*service.LotMap, *service.TileCache) {
	t.Helper()

	cfg := mapConfig(server.URL)
	cache := service.NewTileCache(cacheLimit, cacheTTL)

	lotMap, err := service.NewLotMap(service.NewTileClient(cfg), cache, cfg)
	require.NoError(t, err)

	return lotMap, cache
}

// fixtureLots reads the lots of the parking service near the origin.
func fixtureLots(t *testing.T) []model.NearbyLot {
	t.Helper()

	raw, err := os.ReadFile("testdata/paliashvili.json")
	require.NoError(t, err)

	var answer dto.APIResponse[[]dto.ParkingLot]
	require.NoError(t, json.Unmarshal(raw, &answer))

	return nearbyOf(t, answer.Result.Data...)
}

func nearbyOf(t *testing.T, lots ...dto.ParkingLot) []model.NearbyLot {
	t.Helper()

	nearby := make([]model.NearbyLot, 0, len(lots))

	for _, lot := range lots {
		parkingLot, err := lot.ToModel()
		require.NoError(t, err)

		nearby = append(nearby, model.NearbyLot{ParkingLot: parkingLot})
	}

	return nearby
}

func metersToPoint(east, north float64) orb.Point {
	lat := originLat + north/earthRadius*180/math.Pi
	lng := originLng + east/(earthRadius*math.Cos(originLat*math.Pi/180))*180/math.Pi

	return orb.Point{lng, lat}
}

func pointsOf(lots []model.NearbyLot) []orb.Point {
	points := []orb.Point{{originLng, originLat}}
	for _, lot := range lots {
		points = append(points, lot.Outline[0]...)
	}

	return points
}

func render(t *testing.T, lotMap *service.LotMap, lots []model.NearbyLot) image.Image {
	t.Helper()

	data, err := lotMap.Render(t.Context(), originLat, originLng, lots)
	require.NoError(t, err)

	img, err := png.Decode(bytes.NewReader(data))
	require.NoError(t, err)

	return img
}

func assertColor(t *testing.T, want uint32, got color.Color) {
	t.Helper()

	r, g, b, _ := got.RGBA()
	wants := []uint32{want >> 16 & 0xFF, want >> 8 & 0xFF, want & 0xFF}
	gots := []uint32{r >> 8, g >> 8, b >> 8}

	for i := range wants {
		assert.InDelta(t, wants[i], gots[i], colorSlack, "channel %d of %v", i, gots)
	}
}

func isWhite(c color.Color) bool {
	r, g, b, _ := c.RGBA()

	return r>>8 == 0xFF && g>>8 == 0xFF && b>>8 == 0xFF
}

// PARKMAP-SC-016: Each outline shows in its own color.
func TestRenderDrawsEachOutlineInItsColor(t *testing.T) {
	t.Parallel()

	lotMap, _ := newLotMap(t, newTileServer(t))
	lots := nearbyOf(t,
		rectangle("A306", -20, 20, 6, 9),
		rectangle("A302", -20, 20, 12, 15),
	)

	img := render(t, lotMap, lots)
	view := service.ChooseView(pointsOf(lots), 19)

	for i, want := range []uint32{firstLotRGB, secondLotRGB} {
		south := []float64{6, 12}[i]
		x, y := view.Project(metersToPoint(-18, south))

		assertColor(t, want, img.At(int(x), int(math.Ceil(y-1))))

		_, inside := view.Project(metersToPoint(-18, south+1.5))
		assert.False(t, isWhite(img.At(int(x), int(inside))), "the inside of lot %d", i)
	}
}

// PARKMAP-SC-017: A label leaves the pin clear.
func TestPlaceLabelAvoidsThePin(t *testing.T) {
	t.Parallel()

	outline := []orb.Point{{100, 300}, {280, 300}, {280, 313}, {100, 313}, {100, 300}}

	const width, height = 50, 22

	pin := service.NewBox(180, 290, 20, 29)

	placed := service.PlaceLabel(outline, width, height, []service.Box{pin})
	centerX, centerY := placed.Center()
	assert.InDelta(t, 154, centerX, 0.5)
	assert.InDelta(t, 306.5, centerY, 0.5)
	assert.False(t, placed.Overlaps(pin))

	centerX, centerY = service.PlaceLabel(outline, width, height, nil).Center()
	assert.InDelta(t, 190, centerX, 0.5)
	assert.InDelta(t, 306.5, centerY, 0.5)

	everywhere := service.NewBox(0, 0, 400, 400)
	centerX, centerY = service.PlaceLabel(outline, width, height, []service.Box{everywhere}).
		Center()
	assert.InDelta(t, 190, centerX, 0.5)
	assert.InDelta(t, 306.5, centerY, 0.5)
}

// PARKMAP-SC-018: The map shows the shared location as a pin.
func TestRenderDrawsThePin(t *testing.T) {
	t.Parallel()

	lotMap, _ := newLotMap(t, newTileServer(t))
	lots := fixtureLots(t)

	img := render(t, lotMap, lots)
	x, y := service.ChooseView(pointsOf(lots), 19).Project(orb.Point{originLng, originLat})

	assertColor(t, pinRGB, img.At(int(x), int(y-pinLift)))
}

// PARKMAP-SC-019: Each outline and the pin fit inside the margin of the image.
func TestChooseViewKeepsEveryPointInside(t *testing.T) {
	t.Parallel()

	lots := append(fixtureLots(t), nearbyOf(t, rectangle("A399", 10, 100, -1.5, 1.5))...)
	points := pointsOf(lots)
	view := service.ChooseView(points, 19)

	for _, point := range points {
		x, y := view.Project(point)

		assert.GreaterOrEqual(t, x, float64(service.ViewMargin))
		assert.LessOrEqual(t, x, float64(service.ImageWidth-service.ViewMargin))
		assert.GreaterOrEqual(t, y, float64(service.ViewMargin))
		assert.LessOrEqual(t, y, float64(service.ImageHeight-service.ViewMargin))
	}
}

// PARKMAP-SC-020: The map uses the most detailed zoom that fits.
func TestChooseViewUsesTheMostDetailedZoom(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 19, service.ChooseView(pointsOf(fixtureLots(t)), 19).Zoom())

	apart := []orb.Point{metersToPoint(0, 0), metersToPoint(400, 0)}
	view := service.ChooseView(apart, 19)

	west, _ := view.Project(apart[0])
	east, _ := view.Project(apart[1])
	span := east - west
	room := float64(service.ImageWidth - 2*service.ViewMargin)

	assert.Less(t, view.Zoom(), 19)
	assert.LessOrEqual(t, span, room)
	assert.Greater(t, 2*span, room)
}

// PARKMAP-SC-021: The map names the source of the street data.
func TestRenderDrawsTheAttribution(t *testing.T) {
	t.Parallel()

	lotMap, _ := newLotMap(t, newTileServer(t))
	img := render(t, lotMap, fixtureLots(t))

	dark := false

	for y := service.ImageHeight - stripHeight; y < service.ImageHeight; y++ {
		for x := range 200 {
			if r, _, _, _ := img.At(x, y).RGBA(); r>>8 < darkChannel {
				dark = true
			}
		}
	}

	assert.True(t, dark, "the attribution text")
	assert.True(t, isWhite(img.At(service.ImageWidth-10, service.ImageHeight-5)))
}

// PARKMAP-SC-022: The map source comes from the configuration.
func TestRenderReadsTheConfiguredMapSource(t *testing.T) {
	t.Parallel()

	server := newTileServer(t)
	lotMap, _ := newLotMap(t, server)

	render(t, lotMap, fixtureLots(t))

	assert.NotEmpty(t, server.countsByPath())
}

// PARKMAP-SC-023: A tile request names Maroid and carries no credential.
func TestTileRequestsNameMaroidAndCarryNoCredential(t *testing.T) {
	t.Parallel()

	server := newTileServer(t)
	lotMap, _ := newLotMap(t, server)

	render(t, lotMap, fixtureLots(t))

	server.mu.Lock()
	defer server.mu.Unlock()

	require.NotEmpty(t, server.requests)

	for _, request := range server.requests {
		assert.Equal(t, userAgent, request.Header.Get("User-Agent"))
		assert.Equal(t, "image/png", request.Header.Get("Accept"))
		assert.Empty(t, request.Header.Get("Authorization"))
		assert.Empty(t, request.Header.Get("Cookie"))
		assert.Regexp(t, `^/\d+/\d+/\d+\.png$`, request.URL.Path)
		assert.Empty(t, request.URL.RawQuery)
	}
}

// PARKMAP-SC-027: With every tile held, the map takes less than 1 second.
func TestRenderWithHeldTilesTakesLessThanOneSecond(t *testing.T) {
	t.Parallel()

	server := newTileServer(t)
	lotMap, _ := newLotMap(t, server)
	lots := fixtureLots(t)

	render(t, lotMap, lots)

	before := server.countsByPath()
	start := time.Now()

	render(t, lotMap, lots)

	assert.Less(t, time.Since(start), time.Second)
	assert.Equal(t, before, server.countsByPath())
}

// PARKMAP-SC-028: A tile goes to the map source at most once in 7 days.
func TestRenderRequestsATileOnceIn7Days(t *testing.T) {
	t.Parallel()

	server := newTileServer(t)
	lotMap, cache := newLotMap(t, server)
	lots := fixtureLots(t)

	day0 := time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)
	now := day0

	cache.SetClock(func() time.Time { return now })
	render(t, lotMap, lots)

	now = day0.Add(6 * 24 * time.Hour)

	render(t, lotMap, lots)

	for path, count := range server.countsByPath() {
		assert.Equal(t, 1, count, path)
	}

	now = day0.Add(cacheTTL + time.Second)

	render(t, lotMap, lots)

	for path, count := range server.countsByPath() {
		assert.Equal(t, 2, count, path)
	}
}

// PARKMAP-SC-029: The cache holds at most 50 MB, and drops the least recently used tile.
func TestTileCacheHoldsAtMost50Megabytes(t *testing.T) {
	t.Parallel()

	const megabyte = 1 << 20

	cache := service.NewTileCache(cacheLimit, cacheTTL)

	for i := range 55 {
		cache.Put(service.TileKey{Zoom: 19, X: i}, make([]byte, megabyte))

		if i == 9 {
			_, found := cache.Get(service.TileKey{Zoom: 19, X: 0})
			require.True(t, found)
		}
	}

	assert.LessOrEqual(t, cache.Size(), cacheLimit)

	_, first := cache.Get(service.TileKey{Zoom: 19, X: 0})
	assert.True(t, first, "the first tile")

	_, second := cache.Get(service.TileKey{Zoom: 19, X: 1})
	assert.False(t, second, "the second tile")
}
