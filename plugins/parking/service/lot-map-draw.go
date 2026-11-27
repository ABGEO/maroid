package service

import (
	"image/color"
	"math"
	"slices"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"

	"github.com/abgeo/maroid/plugins/parking/model"
)

const (
	outlineWidth  = 2
	outlineFill   = 0x80
	labelFontSize = 15
	labelPadX     = 6
	labelPadY     = 3
	labelBorder   = 1.5
	labelGridStep = 2
	half          = 0.5
	// bothSides doubles a padding that applies on each side of a box.
	bothSides = 2
	pinBoxPad = 1
	// textMiddle places the anchor of a line of text so that its letters look centered.
	textMiddle     = 0.35
	pinRadius      = 9
	pinLift        = 18
	pinOutline     = 1
	creditFontSize = 11
	creditPadX     = 4
	creditPadY     = 3
	creditFill     = 0xC0
)

//nolint:gochecknoglobals // a fixed palette: Go has no constant of a composite type.
var (
	lotColors = []color.RGBA{
		{0x25, 0x63, 0xEB, 0xFF},
		{0x16, 0xA3, 0x4A, 0xFF},
		{0xD9, 0x77, 0x06, 0xFF},
		{0x93, 0x33, 0xEA, 0xFF},
		{0x0D, 0x94, 0x88, 0xFF},
		{0xDB, 0x27, 0x77, 0xFF},
	}
	pinFill    = color.RGBA{0xDC, 0x26, 0x26, 0xFF}
	pinBorder  = color.RGBA{0x7F, 0x1D, 0x1D, 0xFF}
	creditText = color.RGBA{0x1F, 0x29, 0x37, 0xFF}
)

// box is a rectangle of pixels.
type box struct {
	x, y, w, h float64
}

func (b box) overlaps(other box) bool {
	return b.x < other.x+other.w && other.x < b.x+b.w &&
		b.y < other.y+other.h && other.y < b.y+b.h
}

func (b box) covers(point orb.Point) bool {
	return point.X() >= b.x && point.X() <= b.x+b.w && point.Y() >= b.y && point.Y() <= b.y+b.h
}

func boxAround(x, y, width, height float64) box {
	return box{x: x - width/2, y: y - height/2, w: width, h: height}
}

func pinBox(x, y float64) box {
	return box{
		x: x - pinRadius - pinBoxPad,
		y: y - pinLift - pinRadius - pinBoxPad,
		w: bothSides * (pinRadius + pinBoxPad),
		h: pinLift + pinRadius + bothSides*pinBoxPad,
	}
}

// draw puts the outlines, the labels, the pin, and the attribution on the tiles. The pin
// draws after the labels, so no label hides it.
func (m *LotMap) draw(canvas *gg.Context, view mapView, pin orb.Point, lots []model.NearbyLot) {
	pinX, pinY := view.project(pin)
	outlines := make([][]orb.Point, len(lots))

	for i, lot := range lots {
		outlines[i] = projectRing(view, outerRing(lot.Outline))
		drawOutline(canvas, outlines[i], lotColor(i))
	}

	canvas.SetFontFace(truetype.NewFace(m.labelFont, &truetype.Options{Size: labelFontSize}))

	blocked := make([]box, 0, 1+len(lots))
	blocked = append(blocked, pinBox(pinX, pinY))

	for i, lot := range lots {
		textWidth, textHeight := canvas.MeasureString(lot.Number)
		label := placeLabel(outlines[i], textWidth+2*labelPadX, textHeight+2*labelPadY, blocked)
		blocked = append(blocked, label)

		drawLabel(canvas, label, lot.Number, lotColor(i))
	}

	drawPin(canvas, pinX, pinY)
	m.drawAttribution(canvas)
}

func lotColor(index int) color.RGBA {
	return lotColors[index%len(lotColors)]
}

func projectRing(view mapView, ring orb.Ring) []orb.Point {
	points := make([]orb.Point, 0, len(ring))

	for _, point := range ring {
		x, y := view.project(point)
		points = append(points, orb.Point{x, y})
	}

	return points
}

func drawOutline(canvas *gg.Context, outline []orb.Point, stroke color.RGBA) {
	if len(outline) == 0 {
		return
	}

	canvas.MoveTo(outline[0].X(), outline[0].Y())

	for _, point := range outline[1:] {
		canvas.LineTo(point.X(), point.Y())
	}

	canvas.ClosePath()
	canvas.SetColor(color.NRGBA{R: stroke.R, G: stroke.G, B: stroke.B, A: outlineFill})
	canvas.FillPreserve()
	canvas.SetColor(stroke)
	canvas.SetLineWidth(outlineWidth)
	canvas.Stroke()
}

// placeLabel answers the label nearest to the centroid of the outline, on a grid along
// the longest edge and across it, that covers a point of the outline and meets no
// blocked rectangle. Of two at the same distance, the one nearer the start of the edge
// wins. With no clear position, the label sits at the centroid.
func placeLabel(outline []orb.Point, width, height float64, blocked []box) box {
	center, _ := planar.CentroidArea(orb.Polygon{orb.Ring(outline)})
	atCenter := boxAround(center.X(), center.Y(), width, height)

	alongX, alongY := longestEdgeDirection(outline)
	if alongX == 0 && alongY == 0 {
		return atCenter
	}

	acrossX, acrossY := -alongY, alongX
	samples := sampleOutline(outline)

	lowAlong, highAlong := extent(outline, center, alongX, alongY)
	lowAcross, highAcross := extent(outline, center, acrossX, acrossY)

	best, bestDistance, bestAlong := atCenter, math.Inf(1), math.Inf(1)

	for _, along := range gridSteps(lowAlong-width/2, highAlong+width/2) {
		for _, across := range gridSteps(lowAcross-height/2, highAcross+height/2) {
			distance := math.Hypot(along, across)
			if distance > bestDistance || (distance == bestDistance && along >= bestAlong) {
				continue
			}

			candidate := boxAround(
				center.X()+along*alongX+across*acrossX,
				center.Y()+along*alongY+across*acrossY,
				width, height,
			)

			if overlapsAny(candidate, blocked) || !slices.ContainsFunc(samples, candidate.covers) {
				continue
			}

			best, bestDistance, bestAlong = candidate, distance, along
		}
	}

	return best
}

// extent answers the lowest and the highest offset of the outline from the center,
// along the direction.
func extent(outline []orb.Point, center orb.Point, dirX, dirY float64) (float64, float64) {
	low, high := math.Inf(1), math.Inf(-1)

	for _, point := range outline {
		offset := (point.X()-center.X())*dirX + (point.Y()-center.Y())*dirY
		low, high = min(low, offset), max(high, offset)
	}

	return low, high
}

// gridSteps answers the multiples of labelGridStep from low to high.
func gridSteps(low, high float64) []float64 {
	steps := make([]float64, 0, int((high-low)/labelGridStep)+1)

	for k := math.Ceil(low / labelGridStep); k*labelGridStep <= high; k++ {
		steps = append(steps, k*labelGridStep)
	}

	return steps
}

// sampleOutline answers the points along each edge of the outline, one for each step
// of the grid.
func sampleOutline(outline []orb.Point) []orb.Point {
	samples := make([]orb.Point, 0, len(outline))

	for i := 1; i < len(outline); i++ {
		from, to := outline[i-1], outline[i]
		count := max(1, int(math.Ceil(math.Hypot(to.X()-from.X(), to.Y()-from.Y())/labelGridStep)))

		for step := range count {
			fraction := float64(step) / float64(count)
			samples = append(samples, orb.Point{
				from.X() + fraction*(to.X()-from.X()),
				from.Y() + fraction*(to.Y()-from.Y()),
			})
		}
	}

	return samples
}

func longestEdgeDirection(outline []orb.Point) (float64, float64) {
	var dirX, dirY, longest float64

	for i := 1; i < len(outline); i++ {
		dx := outline[i].X() - outline[i-1].X()
		dy := outline[i].Y() - outline[i-1].Y()

		if length := math.Hypot(dx, dy); length > longest {
			longest, dirX, dirY = length, dx/length, dy/length
		}
	}

	return dirX, dirY
}

func overlapsAny(candidate box, blocked []box) bool {
	return slices.ContainsFunc(blocked, candidate.overlaps)
}

func drawLabel(canvas *gg.Context, label box, text string, fill color.RGBA) {
	outer := box{
		x: label.x - labelBorder,
		y: label.y - labelBorder,
		w: label.w + bothSides*labelBorder,
		h: label.h + bothSides*labelBorder,
	}

	canvas.SetColor(color.White)
	canvas.DrawRoundedRectangle(outer.x, outer.y, outer.w, outer.h, outer.h*half)
	canvas.Fill()

	canvas.SetColor(fill)
	canvas.DrawRoundedRectangle(label.x, label.y, label.w, label.h, label.h*half)
	canvas.Fill()

	canvas.SetColor(color.White)
	canvas.DrawStringAnchored(text, label.x+label.w*half, label.y+label.h*half, half, textMiddle)
}

// drawPin draws a drop whose tip stands on the location: a circle above the tip, joined
// to it by the two tangents.
func drawPin(canvas *gg.Context, tipX, tipY float64) {
	const tangentAngle = math.Pi / 6

	centerY := tipY - pinLift

	canvas.MoveTo(tipX, tipY)
	canvas.DrawArc(tipX, centerY, pinRadius, tangentAngle, tangentAngle-math.Pi*4/3)
	canvas.ClosePath()
	canvas.SetColor(pinFill)
	canvas.FillPreserve()
	canvas.SetColor(pinBorder)
	canvas.SetLineWidth(pinOutline)
	canvas.Stroke()
}

func (m *LotMap) drawAttribution(canvas *gg.Context) {
	canvas.SetFontFace(truetype.NewFace(m.creditFont, &truetype.Options{Size: creditFontSize}))

	textWidth, textHeight := canvas.MeasureString(m.attribution)
	stripHeight := textHeight + bothSides*creditPadY
	stripTop := imageHeight - stripHeight

	canvas.SetColor(
		color.NRGBA{R: math.MaxUint8, G: math.MaxUint8, B: math.MaxUint8, A: creditFill},
	)
	canvas.DrawRectangle(0, stripTop, textWidth+bothSides*creditPadX, stripHeight)
	canvas.Fill()

	canvas.SetColor(creditText)
	canvas.DrawStringAnchored(m.attribution, creditPadX, stripTop+stripHeight/2, 0, textMiddle)
}
