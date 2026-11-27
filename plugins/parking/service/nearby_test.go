package service_test

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/parking/dto"
	"github.com/abgeo/maroid/plugins/parking/service"
)

const (
	originLat = 41.708870
	originLng = 44.764440

	earthRadius = 6371008.8
	// boxTolerance bounds the error of a box edge in degrees.
	boxTolerance = 1e-7
	// distanceTolerance bounds the error of a distance in meters.
	distanceTolerance = 0.1
)

// lotsClient plays the parking service for GetParkingLots, and records the box.
type lotsClient struct {
	service.APIClientService

	lots                     []dto.ParkingLot
	err                      error
	left, right, top, bottom float64
}

func (c *lotsClient) GetParkingLots(
	_ context.Context,
	left, right, top, bottom float64,
) ([]dto.ParkingLot, error) {
	c.left, c.right, c.top, c.bottom = left, right, top, bottom

	return c.lots, c.err
}

// rectangle answers a lot whose outline spans the given meters east and north of
// the origin.
func rectangle(number string, west, east, south, north float64) dto.ParkingLot {
	point := func(x, y float64) string {
		lat := originLat + y/earthRadius*180/math.Pi
		lng := originLng + x/(earthRadius*math.Cos(originLat*math.Pi/180))*180/math.Pi

		return fmt.Sprintf("%.12f %.12f", lng, lat)
	}

	corners := []string{
		point(west, south), point(east, south), point(east, north), point(west, north),
		point(west, south),
	}

	return dto.ParkingLot{
		Polygon:      "POLYGON ((" + strings.Join(corners, ", ") + "))",
		Address:      "zaqaria faliaSvilis quCa #60",
		UniqueNumber: number,
	}
}

func findNearby(t *testing.T, client *lotsClient) []string {
	t.Helper()

	finder := service.NewNearbyLots(client, slog.New(slog.DiscardHandler))

	lots, err := finder.FindNearby(t.Context(), originLat, originLng)
	require.NoError(t, err)

	numbers := make([]string, 0, len(lots))
	for _, lot := range lots {
		numbers = append(numbers, lot.Number)
	}

	return numbers
}

// PARKMAP-SC-001: The search offers each lot within 30 m.
func TestFindNearbyOffersLotsWithin30Meters(t *testing.T) {
	t.Parallel()

	client := &lotsClient{lots: []dto.ParkingLot{
		rectangle("A306", 8, 11, -1.5, 1.5),
		rectangle("A302", -1.5, 1.5, 25, 40),
	}}

	assert.Equal(t, []string{"A306", "A302"}, findNearby(t, client))
}

// PARKMAP-SC-002: A lot whose outline contains the location lies at 0 m.
func TestFindNearbyMeasuresZeroInsideTheOutline(t *testing.T) {
	t.Parallel()

	client := &lotsClient{lots: []dto.ParkingLot{rectangle("A306", -2, 20, -1.5, 1.5)}}
	finder := service.NewNearbyLots(client, slog.New(slog.DiscardHandler))

	lots, err := finder.FindNearby(t.Context(), originLat, originLng)
	require.NoError(t, err)
	require.Len(t, lots, 1)
	assert.Zero(t, lots[0].Distance)
}

// PARKMAP-SC-003: A lot at 35 m in a corner of the box is not offered.
func TestFindNearbyDropsALotBeyond30Meters(t *testing.T) {
	t.Parallel()

	client := &lotsClient{lots: []dto.ParkingLot{rectangle("A308", 24.75, 28, 24.75, 28)}}

	assert.Empty(t, findNearby(t, client))
}

// PARKMAP-SC-004: The search asks for the box of 30 m on each side of the location.
func TestFindNearbyAsksForTheBoxOf30Meters(t *testing.T) {
	t.Parallel()

	client := &lotsClient{}
	findNearby(t, client)

	assert.InDelta(t, 0.00026980, client.top-originLat, boxTolerance)
	assert.InDelta(t, 0.00026980, originLat-client.bottom, boxTolerance)
	assert.InDelta(t, 0.00036140, client.right-originLng, boxTolerance)
	assert.InDelta(t, 0.00036140, originLng-client.left, boxTolerance)
}

// PARKMAP-SC-005: The distance runs to the nearest point of the outline, not to its centroid.
func TestFindNearbyMeasuresToTheNearestPoint(t *testing.T) {
	t.Parallel()

	client := &lotsClient{lots: []dto.ParkingLot{rectangle("A307", 5, 65, -1.5, 1.5)}}
	finder := service.NewNearbyLots(client, slog.New(slog.DiscardHandler))

	lots, err := finder.FindNearby(t.Context(), originLat, originLng)
	require.NoError(t, err)
	require.Len(t, lots, 1)
	assert.InDelta(t, 5, lots[0].Distance, distanceTolerance)
}

// PARKMAP-SC-006: The search offers at most the 6 nearest lots.
func TestFindNearbyKeepsTheSixNearest(t *testing.T) {
	t.Parallel()

	lots := make([]dto.ParkingLot, 0, 9)

	for i := 9; i >= 1; i-- {
		near := float64(3 * i)
		lots = append(lots, rectangle(fmt.Sprintf("A%d", i), near, near+2, -1, 1))
	}

	client := &lotsClient{lots: lots}

	assert.Equal(t, []string{"A1", "A2", "A3", "A4", "A5", "A6"}, findNearby(t, client))
}

// PARKMAP-SC-007: The lots come nearest first, then by zone, then by number.
func TestFindNearbyOrdersByDistanceThenNumber(t *testing.T) {
	t.Parallel()

	client := &lotsClient{lots: []dto.ParkingLot{
		rectangle("B12", 12, 14, -1, 1),
		rectangle("A302", -1, 1, 9, 11),
		rectangle("A102", -14, -12, -1, 1),
		rectangle("A306", -1, 1, -6, -4),
		rectangle("A12", -1, 1, 12, 14),
	}}

	assert.Equal(t, []string{"A306", "A302", "A12", "A102", "B12"}, findNearby(t, client))
}
