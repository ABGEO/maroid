package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"math"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/paulmach/orb"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/sync/errgroup"

	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/model"
)

const (
	imageWidth  = 800
	imageHeight = 600
	tileSize    = 256
	viewMargin  = 48
	minZoom     = 15
	// parallelTiles is the limit of the usage policy of the default map source.
	parallelTiles   = 2
	fullTurnDegrees = 2 * halfTurnDegrees
)

// LotMapRenderer draws the map of the nearby parking lots as a PNG image.
type LotMapRenderer interface {
	Render(ctx context.Context, lat, lng float64, lots []model.NearbyLot) ([]byte, error)
}

// LotMap draws the outlines of the parking lots and the pin of the location on the
// tiles of the map source.
type LotMap struct {
	tiles       TileSource
	cache       *TileCache
	maxZoom     int
	attribution string
	labelFont   *truetype.Font
	creditFont  *truetype.Font
}

var _ LotMapRenderer = (*LotMap)(nil)

// NewLotMap creates a LotMap.
func NewLotMap(tiles TileSource, cache *TileCache, cfg config.MapConfig) (*LotMap, error) {
	labelFont, err := truetype.Parse(gobold.TTF)
	if err != nil {
		return nil, fmt.Errorf("parsing the label font: %w", err)
	}

	creditFont, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("parsing the attribution font: %w", err)
	}

	return &LotMap{
		tiles:       tiles,
		cache:       cache,
		maxZoom:     cfg.MaxZoom,
		attribution: cfg.Attribution,
		labelFont:   labelFont,
		creditFont:  creditFont,
	}, nil
}

// Render draws the lots and the pin of the location, and answers the PNG bytes.
func (m *LotMap) Render(
	ctx context.Context,
	lat, lng float64,
	lots []model.NearbyLot,
) ([]byte, error) {
	pin := orb.Point{lng, lat}

	count := 1
	for _, lot := range lots {
		count += len(outerRing(lot.Outline))
	}

	points := make([]orb.Point, 0, count)
	points = append(points, pin)

	for _, lot := range lots {
		points = append(points, outerRing(lot.Outline)...)
	}

	view := chooseView(points, m.maxZoom)

	tiles, err := m.fetchTiles(ctx, view)
	if err != nil {
		return nil, err
	}

	canvas := gg.NewContext(imageWidth, imageHeight)

	for _, tile := range tiles {
		canvas.DrawImage(tile.img, tile.x, tile.y)
	}

	m.draw(canvas, view, pin, lots)

	var encoded bytes.Buffer
	if err = png.Encode(&encoded, canvas.Image()); err != nil {
		return nil, fmt.Errorf("encoding the map: %w", err)
	}

	return encoded.Bytes(), nil
}

type placedTile struct {
	img  image.Image
	x, y int
}

func (m *LotMap) fetchTiles(ctx context.Context, view mapView) ([]placedTile, error) {
	keys := view.tileKeys()
	placed := make([]placedTile, len(keys))

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(parallelTiles)

	for i, key := range keys {
		group.Go(func() error {
			data, err := m.tile(groupCtx, key)
			if err != nil {
				return err
			}

			img, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				return fmt.Errorf("decoding the tile %d/%d/%d: %w", key.Zoom, key.X, key.Y, err)
			}

			placed[i] = placedTile{
				img: img,
				x:   key.X*tileSize - int(math.Round(view.left)),
				y:   key.Y*tileSize - int(math.Round(view.top)),
			}

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("reading the map tiles: %w", err)
	}

	return placed, nil
}

func (m *LotMap) tile(ctx context.Context, key TileKey) ([]byte, error) {
	if data, found := m.cache.Get(key); found {
		return data, nil
	}

	data, err := m.tiles.Tile(ctx, key.Zoom, key.X, key.Y)
	if err != nil {
		return nil, fmt.Errorf("reading a map tile: %w", err)
	}

	m.cache.Put(key, data)

	return data, nil
}

// mapView places the image on the world of Web Mercator at one zoom. left and top
// hold the world pixel of the top left corner of the image.
type mapView struct {
	zoom      int
	left, top float64
}

// chooseView answers the most detailed view at which every point fits inside the
// margin, down to minZoom.
func chooseView(points []orb.Point, maxZoom int) mapView {
	for zoom := maxZoom; zoom > minZoom; zoom-- {
		if view, fits := fitView(points, zoom); fits {
			return view
		}
	}

	view, _ := fitView(points, minZoom)

	return view
}

func fitView(points []orb.Point, zoom int) (mapView, bool) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)

	for _, point := range points {
		x, y := worldPixel(point, zoom)
		minX, maxX = min(minX, x), max(maxX, x)
		minY, maxY = min(minY, y), max(maxY, y)
	}

	fits := maxX-minX <= imageWidth-2*viewMargin && maxY-minY <= imageHeight-2*viewMargin

	return mapView{
		zoom: zoom,
		left: (minX+maxX)/2 - imageWidth/2,
		top:  (minY+maxY)/2 - imageHeight/2,
	}, fits
}

func (v mapView) project(point orb.Point) (float64, float64) {
	x, y := worldPixel(point, v.zoom)

	return x - v.left, y - v.top
}

func (v mapView) tileKeys() []TileKey {
	first := func(edge float64) int { return int(math.Floor(edge / tileSize)) }
	last := func(edge float64, size int) int { return int(math.Floor((edge + float64(size) - 1) / tileSize)) }

	keys := make([]TileKey, 0)

	for y := first(v.top); y <= last(v.top, imageHeight); y++ {
		for x := first(v.left); x <= last(v.left, imageWidth); x++ {
			keys = append(keys, TileKey{Zoom: v.zoom, X: x, Y: y})
		}
	}

	return keys
}

// worldPixel answers the pixel of the point on the world of Web Mercator at the zoom.
func worldPixel(point orb.Point, zoom int) (float64, float64) {
	scale := tileSize * math.Exp2(float64(zoom))
	lat := radians(point.Lat())

	x := (point.Lon() + halfTurnDegrees) / fullTurnDegrees * scale
	y := (1 - math.Log(math.Tan(lat)+1/math.Cos(lat))/math.Pi) * half * scale

	return x, y
}

func outerRing(outline orb.Polygon) orb.Ring {
	if len(outline) == 0 {
		return nil
	}

	return outline[0]
}
