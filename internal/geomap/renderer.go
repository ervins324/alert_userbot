package geomap

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"

	"alert-userbot/internal/geoparse"
)

const (
	CanvasWidth  = 1000
	CanvasHeight = 850
	MarginLeft   = 40
	MarginRight  = 40
	MarginTop    = 80
	MarginBottom = 40
)

// Project converts (lat, lon) to (x, y) canvas pixel coordinates.
func Project(lat, lon float64, width, height int) (int, int) {
	// Linear equirectangular projection for local Kyiv bounds
	usableW := float64(width - MarginLeft - MarginRight)
	usableH := float64(height - MarginTop - MarginBottom)

	normX := (lon - MinLon) / (MaxLon - MinLon)
	// Invert Y because canvas Y grows downwards while Lat grows upwards
	normY := 1.0 - ((lat - MinLat) / (MaxLat - MinLat))

	x := int(float64(MarginLeft) + normX*usableW)
	y := int(float64(MarginTop) + normY*usableH)
	return x, y
}

// RenderKyivMap generates a PNG map of Kyiv highlighting the matched location.
func RenderKyivMap(loc *geoparse.LocationResult) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, CanvasWidth, CanvasHeight))

	// 1. Fill background with dark tactical slate
	bgCol := color.RGBA{R: 15, G: 20, B: 28, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgCol}, image.Point{}, draw.Src)

	// 2. Draw subtle grid lines
	gridCol := color.RGBA{R: 24, G: 32, B: 46, A: 255}
	for x := MarginLeft; x < CanvasWidth-MarginRight; x += 80 {
		drawLine(img, x, MarginTop, x, CanvasHeight-MarginBottom, gridCol, 1)
	}
	for y := MarginTop; y < CanvasHeight-MarginBottom; y += 80 {
		drawLine(img, MarginLeft, y, CanvasWidth-MarginRight, y, gridCol, 1)
	}

	// 3. Highlighted raion set
	highlightedSet := make(map[geoparse.RaionID]bool)
	if loc != nil {
		for _, r := range loc.MatchedRaions {
			highlightedSet[r] = true
		}
	}

	// 4. Draw normal raion polygons
	normalFill := color.RGBA{R: 26, G: 35, B: 50, A: 255}
	normalBorder := color.RGBA{R: 50, G: 68, B: 95, A: 255}

	highlightFill := color.RGBA{R: 180, G: 40, B: 45, A: 210}
	highlightBorder := color.RGBA{R: 255, G: 70, B: 75, A: 255}

	for id, raion := range KyivRaionBoundaries {
		if highlightedSet[id] {
			continue
		}
		polyPoints := coordsToPoints(raion.Boundary, CanvasWidth, CanvasHeight)
		fillPolygon(img, polyPoints, normalFill)
		drawPolyline(img, polyPoints, normalBorder, 2)
	}

	// 5. Draw Dnipro River
	riverPoints := coordsToPoints(DniproRiverPolygon, CanvasWidth, CanvasHeight)
	riverCol := color.RGBA{R: 28, G: 75, B: 125, A: 255}
	riverBorderCol := color.RGBA{R: 40, G: 105, B: 170, A: 255}
	drawPolyline(img, riverPoints, riverCol, 18)
	drawPolyline(img, riverPoints, riverBorderCol, 3)

	// 6. Draw highlighted raions on top
	for id, raion := range KyivRaionBoundaries {
		if !highlightedSet[id] {
			continue
		}
		polyPoints := coordsToPoints(raion.Boundary, CanvasWidth, CanvasHeight)
		fillPolygon(img, polyPoints, highlightFill)
		drawPolyline(img, polyPoints, highlightBorder, 3)
	}

	// 7. Draw District Labels
	labelNormalCol := color.RGBA{R: 130, G: 150, B: 175, A: 255}
	labelHighlightCol := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	for id, raion := range KyivRaionBoundaries {
		cx, cy := Project(raion.Center.Lat, raion.Center.Lon, CanvasWidth, CanvasHeight)
		name := raion.NameUA
		w, h := MeasureText(name, 1)

		isHigh := highlightedSet[id]
		col := labelNormalCol
		if isHigh {
			col = labelHighlightCol
			// Draw badge under highlighted label
			fillBadge(img, cx-w/2-6, cy-h/2-4, w+12, h+8, color.RGBA{R: 120, G: 20, B: 25, A: 220})
			drawRect(img, cx-w/2-6, cy-h/2-4, w+12, h+8, highlightBorder, 1)
		}
		DrawText(img, name, cx-w/2, cy-h/2, 1, col)
	}

	// 8. Draw pinpoint marker if specific coordinates / neighborhood found
	if loc != nil && loc.HasSpecificPoint && loc.Latitude > 0 && loc.Longitude > 0 {
		px, py := Project(loc.Latitude, loc.Longitude, CanvasWidth, CanvasHeight)
		drawPinMarker(img, px, py, loc.NeighborhoodName)
	}

	// 9. Draw Header Info Card
	drawHeaderCard(img, loc)

	// 10. Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func coordsToPoints(coords []Coord, w, h int) []image.Point {
	pts := make([]image.Point, len(coords))
	for i, c := range coords {
		x, y := Project(c.Lat, c.Lon, w, h)
		pts[i] = image.Point{X: x, Y: y}
	}
	return pts
}

func drawHeaderCard(img *image.RGBA, loc *geoparse.LocationResult) {
	cardX := 30
	cardY := 15
	cardW := CanvasWidth - 60
	cardH := 55

	fillBadge(img, cardX, cardY, cardW, cardH, color.RGBA{R: 20, G: 28, B: 40, A: 240})
	drawRect(img, cardX, cardY, cardW, cardH, color.RGBA{R: 60, G: 80, B: 110, A: 255}, 1)

	// Status title
	title := "КАРТА КИЄВА — ЛОКАЛІЗАЦІЯ ОБ'ЄКТА / ПОДІЇ"
	DrawText(img, title, cardX+16, cardY+10, 1, color.RGBA{R: 140, G: 165, B: 195, A: 255})

	// Subtitle with target details
	target := "м. Київ"
	targetCol := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if loc != nil && loc.Description != "" {
		target = "ЛОКАЦІЯ: " + loc.Description
		if len(loc.MatchedRaions) > 0 {
			targetCol = color.RGBA{R: 255, G: 90, B: 95, A: 255}
		}
	}
	DrawText(img, target, cardX+16, cardY+28, 2, targetCol)
}

func drawPinMarker(img *image.RGBA, x, y int, label string) {
	// Marker outer pulsing glow / rings
	drawCircle(img, x, y, 16, color.RGBA{R: 255, G: 50, B: 60, A: 100}, 2)
	drawCircle(img, x, y, 10, color.RGBA{R: 255, G: 30, B: 40, A: 200}, 3)
	drawCircle(img, x, y, 4, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 4)

	// Label badge above pin
	if label != "" {
		w, h := MeasureText(label, 1)
		bx := x - w/2 - 8
		by := y - 32
		fillBadge(img, bx, by, w+16, h+8, color.RGBA{R: 20, G: 20, B: 30, A: 240})
		drawRect(img, bx, by, w+16, h+8, color.RGBA{R: 255, G: 60, B: 70, A: 255}, 1)
		DrawText(img, label, bx+8, by+4, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		// Small connector line to pin
		drawLine(img, x, by+h+8, x, y-10, color.RGBA{R: 255, G: 60, B: 70, A: 255}, 1)
	}
}

// Drawing primitives: fillPolygon, drawLine, drawPolyline, fillBadge, drawRect, drawCircle

func fillPolygon(img *image.RGBA, pts []image.Point, col color.RGBA) {
	if len(pts) < 3 {
		return
	}
	minY, maxY := pts[0].Y, pts[0].Y
	for _, p := range pts {
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	if minY < 0 {
		minY = 0
	}
	if maxY >= CanvasHeight {
		maxY = CanvasHeight - 1
	}

	for y := minY; y <= maxY; y++ {
		var nodeX []int
		j := len(pts) - 1
		for i := 0; i < len(pts); i++ {
			p1, p2 := pts[i], pts[j]
			if (p1.Y < y && p2.Y >= y) || (p2.Y < y && p1.Y >= y) {
				x := p1.X + (y-p1.Y)*(p2.X-p1.X)/(p2.Y-p1.Y)
				nodeX = append(nodeX, x)
			}
			j = i
		}

		// Sort nodeX
		for i := 0; i < len(nodeX); i++ {
			for k := i + 1; k < len(nodeX); k++ {
				if nodeX[i] > nodeX[k] {
					nodeX[i], nodeX[k] = nodeX[k], nodeX[i]
				}
			}
		}

		// Fill between pairs
		for i := 0; i+1 < len(nodeX); i += 2 {
			x1 := nodeX[i]
			x2 := nodeX[i+1]
			if x1 < 0 {
				x1 = 0
			}
			if x2 >= CanvasWidth {
				x2 = CanvasWidth - 1
			}
			for x := x1; x <= x2; x++ {
				blendPixel(img, x, y, col)
			}
		}
	}
}

func blendPixel(img *image.RGBA, x, y int, src color.RGBA) {
	if x < 0 || x >= CanvasWidth || y < 0 || y >= CanvasHeight {
		return
	}
	if src.A == 255 {
		img.SetRGBA(x, y, src)
		return
	}
	dst := img.RGBAAt(x, y)
	alpha := float64(src.A) / 255.0
	invAlpha := 1.0 - alpha

	outR := uint8(float64(src.R)*alpha + float64(dst.R)*invAlpha)
	outG := uint8(float64(src.G)*alpha + float64(dst.G)*invAlpha)
	outB := uint8(float64(src.B)*alpha + float64(dst.B)*invAlpha)
	img.SetRGBA(x, y, color.RGBA{R: outR, G: outG, B: outB, A: 255})
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA, thickness int) {
	dx := int(math.Abs(float64(x1 - x0)))
	dy := int(math.Abs(float64(y1 - y0)))
	sx, sy := 1, 1
	if x0 >= x1 {
		sx = -1
	}
	if y0 >= y1 {
		sy = -1
	}
	err := dx - dy

	r := thickness / 2
	for {
		for ty := -r; ty <= r; ty++ {
			for tx := -r; tx <= r; tx++ {
				blendPixel(img, x0+tx, y0+ty, col)
			}
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func drawPolyline(img *image.RGBA, pts []image.Point, col color.RGBA, thickness int) {
	for i := 0; i < len(pts)-1; i++ {
		drawLine(img, pts[i].X, pts[i].Y, pts[i+1].X, pts[i+1].Y, col, thickness)
	}
}

func fillBadge(img *image.RGBA, x, y, w, h int, col color.RGBA) {
	for cy := y; cy < y+h; cy++ {
		for cx := x; cx < x+w; cx++ {
			blendPixel(img, cx, cy, col)
		}
	}
}

func drawRect(img *image.RGBA, x, y, w, h int, col color.RGBA, thickness int) {
	drawLine(img, x, y, x+w, y, col, thickness)
	drawLine(img, x+w, y, x+w, y+h, col, thickness)
	drawLine(img, x+w, y+h, x, y+h, col, thickness)
	drawLine(img, x, y+h, x, y, col, thickness)
}

func drawCircle(img *image.RGBA, cx, cy, radius int, col color.RGBA, thickness int) {
	for r := radius - thickness/2; r <= radius+thickness/2; r++ {
		for theta := 0.0; theta < 2*math.Pi; theta += 0.02 {
			x := int(float64(cx) + float64(r)*math.Cos(theta))
			y := int(float64(cy) + float64(r)*math.Sin(theta))
			blendPixel(img, x, y, col)
		}
	}
}
