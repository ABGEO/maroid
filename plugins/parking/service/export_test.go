package service

import (
	"time"

	"github.com/paulmach/orb"
)

// The map geometry, exported for the tests of the service package.
const (
	ImageWidth  = imageWidth
	ImageHeight = imageHeight
	ViewMargin  = viewMargin
)

// MapView exposes the view of a map to the tests.
type MapView = mapView

// Box exposes a rectangle of pixels to the tests.
type Box = box

// ChooseView exposes chooseView to the tests.
func ChooseView(points []orb.Point, maxZoom int) MapView {
	return chooseView(points, maxZoom)
}

// Project answers the pixel of the point in the view.
func (v mapView) Project(point orb.Point) (float64, float64) {
	return v.project(point)
}

// Zoom answers the zoom of the view.
func (v mapView) Zoom() int {
	return v.zoom
}

// NewBox builds a rectangle of pixels.
func NewBox(x, y, width, height float64) Box {
	return box{x: x, y: y, w: width, h: height}
}

// Center answers the center of the rectangle.
func (b box) Center() (float64, float64) {
	return b.x + b.w/2, b.y + b.h/2
}

// Overlaps exposes overlaps to the tests.
func (b box) Overlaps(other box) bool {
	return b.overlaps(other)
}

// PlaceLabel exposes placeLabel to the tests.
func PlaceLabel(outline []orb.Point, width, height float64, blocked []Box) Box {
	return placeLabel(outline, width, height, blocked)
}

// SetClock replaces the clock of the cache.
func (c *TileCache) SetClock(now func() time.Time) {
	c.now = now
}
