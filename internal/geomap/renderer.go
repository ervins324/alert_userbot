package geomap

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	_ "image/jpeg"
	"io"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"

	"alert-userbot/internal/geoparse"
)

// MapConfig holds configuration for the map renderer.
type MapConfig struct {
	GoogleMapsAPIKey string
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// RenderKyivMap generates a high-resolution map image using Google Maps Static API (if key present)
// or by stitching real OpenStreetMap / CartoDB street map tiles.
func RenderKyivMap(loc *geoparse.LocationResult, apiKey string) ([]byte, error) {
	// 1. Try Google Maps Static API first if API key is configured
	if apiKey != "" {
		imgBytes, err := fetchGoogleStaticMap(loc, apiKey)
		if err == nil && len(imgBytes) > 0 {
			return imgBytes, nil
		}
	}

	// 2. Fallback to real street map tile stitcher
	return renderRealTileMap(loc)
}

// fetchGoogleStaticMap requests a high-resolution roadmap from Google Maps Static API.
func fetchGoogleStaticMap(loc *geoparse.LocationResult, apiKey string) ([]byte, error) {
	baseURL := "https://maps.googleapis.com/maps/api/staticmap"
	params := url.Values{}

	// Center on Kyiv or target point
	centerLat := 50.4501
	centerLon := 30.5234
	zoom := "11" // Zoom 11 shows entire Kyiv city

	if loc != nil && loc.Latitude > 0 && loc.Longitude > 0 {
		centerLat = loc.Latitude
		centerLon = loc.Longitude
		zoom = "11"
	}

	params.Set("center", fmt.Sprintf("%.5f,%.5f", centerLat, centerLon))
	params.Set("zoom", zoom)
	params.Set("size", "640x500")
	params.Set("scale", "2") // High-DPI 1280x1000 image
	params.Set("maptype", "roadmap")
	params.Set("format", "png")
	params.Set("key", apiKey)

	// Add marker pin for target location
	if loc != nil && loc.Latitude > 0 && loc.Longitude > 0 {
		markerParam := fmt.Sprintf("color:red|size:mid|%.5f,%.5f", loc.Latitude, loc.Longitude)
		params.Add("markers", markerParam)
	}

	// Add polygon path for highlighted district
	if loc != nil && len(loc.MatchedRaions) > 0 {
		for _, rID := range loc.MatchedRaions {
			if poly, ok := KyivRaionBoundaries[rID]; ok && len(poly.Boundary) > 0 {
				pathParam := "color:0xff1744ff|weight:3|fillcolor:0xff174428"
				for _, pt := range poly.Boundary {
					pathParam += fmt.Sprintf("|%.5f,%.5f", pt.Lat, pt.Lon)
				}
				params.Add("path", pathParam)
			}
		}
	}

	reqURL := baseURL + "?" + params.Encode()
	resp, err := httpClient.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google static map returned %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// renderRealTileMap fetches and stitches real Web Mercator map tiles (CartoDB / OSM)
// and overlays district boundary highlights, Google Maps-style pin marker, and header banner.
func renderRealTileMap(loc *geoparse.LocationResult) ([]byte, error) {
	zoom := 11

	// Bounding coordinates covering Kyiv city and immediate agglomeration
	latMin, latMax := 50.260, 50.590
	lonMin, lonMax := 30.220, 30.820

	tileXMin, tileYMin := latLonToTile(latMax, lonMin, zoom)
	tileXMax, tileYMax := latLonToTile(latMin, lonMax, zoom)

	cols := tileXMax - tileXMin + 1
	rows := tileYMax - tileYMin + 1
	tileSize := 256

	canvasW := cols * tileSize
	canvasH := rows * tileSize

	stitched := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))

	// Download and stitch tiles in parallel
	var wg sync.WaitGroup
	type tileResult struct {
		x, y int
		img  image.Image
	}
	resChan := make(chan tileResult, cols*rows)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	for ty := tileYMin; ty <= tileYMax; ty++ {
		for tx := tileXMin; tx <= tileXMax; tx++ {
			wg.Add(1)
			go func(tileX, tileY int) {
				defer wg.Done()
				img := fetchMapTile(ctx, zoom, tileX, tileY)
				if img != nil {
					resChan <- tileResult{x: tileX, y: tileY, img: img}
				}
			}(tx, ty)
		}
	}

	wg.Wait()
	close(resChan)

	tileCount := 0
	for res := range resChan {
		destX := (res.x - tileXMin) * tileSize
		destY := (res.y - tileYMin) * tileSize
		draw.Draw(stitched, image.Rect(destX, destY, destX+tileSize, destY+tileSize), res.img, image.Point{}, draw.Over)
		tileCount++
	}

	// Projection helper for overlaying onto the stitched tile canvas
	projectTile := func(lat, lon float64) (int, int) {
		gx, gy := latLonToGlobalPixel(lat, lon, zoom)
		minGX, minGY := float64(tileXMin*tileSize), float64(tileYMin*tileSize)
		px := int(gx - minGX)
		py := int(gy - minGY)
		return px, py
	}

	// If no online tiles were downloaded (offline fallback), draw clean slate basemap
	if tileCount == 0 {
		bgCol := color.RGBA{R: 24, G: 32, B: 44, A: 255}
		draw.Draw(stitched, stitched.Bounds(), &image.Uniform{C: bgCol}, image.Point{}, draw.Src)
	}

	// Draw highlighted raion polygons onto the real map
	highlightedSet := make(map[geoparse.RaionID]bool)
	if loc != nil {
		for _, r := range loc.MatchedRaions {
			highlightedSet[r] = true
		}
	}

	highlightFill := color.RGBA{R: 235, G: 50, B: 65, A: 75}
	highlightBorder := color.RGBA{R: 235, G: 50, B: 65, A: 240}
	normalBorder := color.RGBA{R: 80, G: 110, B: 150, A: 160}

	for id, raion := range KyivRaionBoundaries {
		pts := make([]image.Point, len(raion.Boundary))
		for i, c := range raion.Boundary {
			px, py := projectTile(c.Lat, c.Lon)
			pts[i] = image.Point{X: px, Y: py}
		}

		if highlightedSet[id] {
			fillPolygon(stitched, pts, highlightFill)
			drawPolyline(stitched, pts, highlightBorder, 4)
		} else {
			drawPolyline(stitched, pts, normalBorder, 2)
		}
	}

	// Draw district labels
	for id, raion := range KyivRaionBoundaries {
		cx, cy := projectTile(raion.Center.Lat, raion.Center.Lon)
		name := raion.NameUA
		w, h := MeasureText(name, 1)

		if highlightedSet[id] {
			fillBadge(stitched, cx-w/2-6, cy-h/2-4, w+12, h+8, color.RGBA{R: 200, G: 30, B: 45, A: 230})
			drawRect(stitched, cx-w/2-6, cy-h/2-4, w+12, h+8, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 1)
			DrawText(stitched, name, cx-w/2, cy-h/2, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		} else {
			fillBadge(stitched, cx-w/2-4, cy-h/2-3, w+8, h+6, color.RGBA{R: 20, G: 25, B: 35, A: 180})
			DrawText(stitched, name, cx-w/2, cy-h/2, 1, color.RGBA{R: 220, G: 230, B: 245, A: 255})
		}
	}

	// Draw Google Maps-style pinpoint marker 📍
	if loc != nil && loc.Latitude > 0 && loc.Longitude > 0 {
		px, py := projectTile(loc.Latitude, loc.Longitude)
		drawGoogleStylePin(stitched, px, py, loc.NeighborhoodName)
	}

	// Draw Header Info Card
	drawTopHeader(stitched, loc)

	var buf bytes.Buffer
	if err := png.Encode(&buf, stitched); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// fetchMapTile downloads a map tile from CartoDB Voyager or OpenStreetMap.
func fetchMapTile(ctx context.Context, zoom, x, y int) image.Image {
	urls := []string{
		fmt.Sprintf("https://a.basemaps.cartocdn.com/rastertiles/voyager/%d/%d/%d.png", zoom, x, y),
		fmt.Sprintf("https://tile.openstreetmap.org/%d/%d/%d.png", zoom, x, y),
	}

	for _, u := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "KyivAirAlertMonitor/1.0")

		resp, err := httpClient.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			img, _, err := image.Decode(resp.Body)
			if err == nil {
				return img
			}
		}
	}
	return nil
}

// latLonToTile converts lat/lon to Web Mercator tile index at given zoom.
func latLonToTile(lat, lon float64, zoom int) (int, int) {
	n := math.Pow(2, float64(zoom))
	x := int((lon + 180.0) / 360.0 * n)
	latRad := lat * math.Pi / 180.0
	y := int((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n)
	return x, y
}

// latLonToGlobalPixel converts lat/lon to global pixel coordinates at given zoom.
func latLonToGlobalPixel(lat, lon float64, zoom int) (float64, float64) {
	n := math.Pow(2, float64(zoom)) * 256.0
	gx := (lon + 180.0) / 360.0 * n
	latRad := lat * math.Pi / 180.0
	gy := (1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n
	return gx, gy
}

// drawGoogleStylePin renders an authentic Google Maps-style red drop pin with shadow.
func drawGoogleStylePin(img *image.RGBA, x, y int, label string) {
	pinHeadY := y - 22
	radius := 12

	// Drop shadow
	drawCircle(img, x+3, y+2, 8, color.RGBA{R: 0, G: 0, B: 0, A: 90}, 6)

	// Pin needle tip (triangle connecting head to point x, y)
	fillPolygon(img, []image.Point{
		{X: x - 6, Y: pinHeadY + 4},
		{X: x + 6, Y: pinHeadY + 4},
		{X: x, Y: y},
	}, color.RGBA{R: 217, G: 48, B: 37, A: 255})

	// Red circular pin head (Google Maps Red #D93025)
	drawCircle(img, x, pinHeadY, radius, color.RGBA{R: 217, G: 48, B: 37, A: 255}, radius)
	drawCircle(img, x, pinHeadY, radius+1, color.RGBA{R: 180, G: 20, B: 20, A: 255}, 1)

	// White inner dot
	drawCircle(img, x, pinHeadY, 4, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 4)

	// Pin label badge above marker
	if label != "" {
		w, h := MeasureText(label, 1)
		bx := x - w/2 - 8
		by := y - 48
		fillBadge(img, bx, by, w+16, h+8, color.RGBA{R: 255, G: 255, B: 255, A: 245})
		drawRect(img, bx, by, w+16, h+8, color.RGBA{R: 217, G: 48, B: 37, A: 255}, 2)
		DrawText(img, label, bx+8, by+4, 1, color.RGBA{R: 30, G: 30, B: 30, A: 255})
		// Pointer stem
		drawLine(img, x, by+h+8, x, pinHeadY-radius-1, color.RGBA{R: 217, G: 48, B: 37, A: 255}, 2)
	}
}

func drawTopHeader(img *image.RGBA, loc *geoparse.LocationResult) {
	cardX := 20
	cardY := 12
	cardW := img.Rect.Dx() - 40
	cardH := 52

	fillBadge(img, cardX, cardY, cardW, cardH, color.RGBA{R: 255, G: 255, B: 255, A: 245})
	drawRect(img, cardX, cardY, cardW, cardH, color.RGBA{R: 217, G: 48, B: 37, A: 255}, 2)

	title := "КАРТА КИЄВА — ЛОКАЛІЗАЦІЯ ОБ'ЄКТА"
	DrawText(img, title, cardX+14, cardY+8, 1, color.RGBA{R: 100, G: 100, B: 100, A: 255})

	target := "м. Київ"
	if loc != nil && loc.Description != "" {
		target = "📍 " + loc.Description
	}
	DrawText(img, target, cardX+14, cardY+24, 2, color.RGBA{R: 217, G: 48, B: 37, A: 255})
}

// Basic raster drawing helpers

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
	if maxY >= img.Rect.Dy() {
		maxY = img.Rect.Dy() - 1
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

		for i := 0; i < len(nodeX); i++ {
			for k := i + 1; k < len(nodeX); k++ {
				if nodeX[i] > nodeX[k] {
					nodeX[i], nodeX[k] = nodeX[k], nodeX[i]
				}
			}
		}

		for i := 0; i+1 < len(nodeX); i += 2 {
			x1 := nodeX[i]
			x2 := nodeX[i+1]
			if x1 < 0 {
				x1 = 0
			}
			if x2 >= img.Rect.Dx() {
				x2 = img.Rect.Dx() - 1
			}
			for x := x1; x <= x2; x++ {
				blendPixel(img, x, y, col)
			}
		}
	}
}

func blendPixel(img *image.RGBA, x, y int, src color.RGBA) {
	if x < 0 || x >= img.Rect.Dx() || y < 0 || y >= img.Rect.Dy() {
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
