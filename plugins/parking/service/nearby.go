package service

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strconv"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"

	"github.com/abgeo/maroid/plugins/parking/model"
)

const (
	searchRadiusMeters = 30
	maxOfferedLots     = 6
	earthRadiusMeters  = 6371008.8
	halfTurnDegrees    = 180
	// distanceStepMeters keeps a float noise from deciding the order of two lots.
	distanceStepMeters = 0.01
)

// LotFinder finds the parking lots near a location.
type LotFinder interface {
	FindNearby(ctx context.Context, lat, lng float64) ([]model.NearbyLot, error)
}

// NearbyLots finds the parking lots near a location through the parking service.
type NearbyLots struct {
	client APIClientService
	logger *slog.Logger
}

var _ LotFinder = (*NearbyLots)(nil)

// NewNearbyLots creates a NearbyLots.
func NewNearbyLots(client APIClientService, logger *slog.Logger) *NearbyLots {
	return &NearbyLots{client: client, logger: logger}
}

// FindNearby answers the parking lots whose outline lies within 30 m of the location,
// the nearest first, at most 6.
func (n *NearbyLots) FindNearby(
	ctx context.Context,
	lat, lng float64,
) ([]model.NearbyLot, error) {
	latOffset := degrees(searchRadiusMeters / earthRadiusMeters)
	lngOffset := latOffset / math.Cos(radians(lat))

	answers, err := n.client.GetParkingLots(
		ctx,
		lng-lngOffset, lng+lngOffset,
		lat+latOffset, lat-latOffset,
	)
	if err != nil {
		return nil, fmt.Errorf("searching the parking lots: %w", err)
	}

	plane := localPlane{lat: lat, lng: lng}
	nearby := make([]model.NearbyLot, 0, len(answers))

	for _, answer := range answers {
		lot, convErr := answer.ToModel()
		if convErr != nil {
			n.logger.WarnContext(
				ctx,
				"parking lot outline not read",
				slog.String("lot", answer.UniqueNumber),
				slog.Any("error", convErr),
			)

			continue
		}

		distance := plane.distanceTo(lot.Outline)
		if distance > searchRadiusMeters {
			continue
		}

		nearby = append(nearby, model.NearbyLot{ParkingLot: lot, Distance: distance})
	}

	slices.SortStableFunc(nearby, compareNearby)

	return nearby[:min(len(nearby), maxOfferedLots)], nil
}

// localPlane projects a point to meters east and north of the location. Over 100 m
// its error stays below 1 mm.
type localPlane struct {
	lat, lng float64
}

func (p localPlane) project(point orb.Point) orb.Point {
	return orb.Point{
		radians(point.Lon()-p.lng) * math.Cos(radians(p.lat)) * earthRadiusMeters,
		radians(point.Lat()-p.lat) * earthRadiusMeters,
	}
}

func (p localPlane) distanceTo(outline orb.Polygon) float64 {
	projected := make(orb.Polygon, 0, len(outline))

	for _, ring := range outline {
		points := make(orb.Ring, 0, len(ring))
		for _, point := range ring {
			points = append(points, p.project(point))
		}

		projected = append(projected, points)
	}

	origin := orb.Point{}
	if planar.PolygonContains(projected, origin) {
		return 0
	}

	return planar.DistanceFrom(projected, origin)
}

func compareNearby(a, b model.NearbyLot) int {
	return cmp.Or(
		cmp.Compare(distanceStep(a.Distance), distanceStep(b.Distance)),
		compareLotNumbers(a.Number, b.Number),
	)
}

func distanceStep(distance float64) float64 {
	return math.Round(distance / distanceStepMeters)
}

// compareLotNumbers orders by the zone letter, then by the integer after it, so A12
// comes before A102.
func compareLotNumbers(first, second string) int {
	if first == "" || second == "" {
		return cmp.Compare(first, second)
	}

	firstNumber, firstErr := strconv.Atoi(first[1:])
	secondNumber, secondErr := strconv.Atoi(second[1:])

	if firstErr != nil || secondErr != nil {
		return cmp.Compare(first, second)
	}

	return cmp.Or(cmp.Compare(first[0], second[0]), cmp.Compare(firstNumber, secondNumber))
}

func radians(deg float64) float64 {
	return deg * math.Pi / halfTurnDegrees
}

func degrees(rad float64) float64 {
	return rad * halfTurnDegrees / math.Pi
}
