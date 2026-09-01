package geomap

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"net/http"
	"sync"
	"time"

	"alert-userbot/internal/geoparse"
)

var httpClient = &http.Client{
	Timeout: 8 * time.Second,
}

// RenderKyivMap generates an authentic OpenStreetMap image with red pushpin markers for all detected places.
func RenderKyivMap(loc *geoparse.LocationResult) ([]byte, error) {
	return stitchOSMMap(loc)
}

// stitchOSMMap downloads official OpenStreetMap tiles in parallel and draws red pin markers for all places.
func stitchOSMMap(loc *geoparse.LocationResult) ([]byte, error) {
	centerLat := 50.4501
	centerLon := 30.5234
	zoom := 11

	if loc != nil && len(loc.Points) > 0 {
		if len(loc.Points) == 1 {
			centerLat = loc.Points[0].Lat
			centerLon = loc.Points[0].Lon
			zoom = 12
		} else {
			minLat, maxLat := loc.Points[0].Lat, loc.Points[0].Lat
			minLon, maxLon := loc.Points[0].Lon, loc.Points[0].Lon
			sumLat, sumLon := 0.0, 0.0

			for _, pt := range loc.Points {
				if pt.Lat < minLat {
					minLat = pt.Lat
				}
				if pt.Lat > maxLat {
					maxLat = pt.Lat
				}
				if pt.Lon < minLon {
					minLon = pt.Lon
				}
				if pt.Lon > maxLon {
					maxLon = pt.Lon
				}
				sumLat += pt.Lat
				sumLon += pt.Lon
			}

			centerLat = sumLat / float64(len(loc.Points))
			centerLon = sumLon / float64(len(loc.Points))

			latSpan := maxLat - minLat
			lonSpan := maxLon - minLon
			maxSpan := math.Max(latSpan, lonSpan)

			if maxSpan > 0.30 {
				zoom = 10
			} else if maxSpan > 0.12 {
				zoom = 11
			} else if maxSpan > 0.05 {
				zoom = 12
			} else {
				zoom = 13
			}
		}
	}

	// Target image dimensions
	width := 1000
	height := 700

	centerGlobalX, centerGlobalY := latLonToGlobalPixel(centerLat, centerLon, zoom)
	topLeftGlobalX := centerGlobalX - float64(width)/2.0
	topLeftGlobalY := centerGlobalY - float64(height)/2.0

	tileXMin := int(math.Floor(topLeftGlobalX / 256.0))
	tileYMin := int(math.Floor(topLeftGlobalY / 256.0))
	tileXMax := int(math.Floor((topLeftGlobalX + float64(width)) / 256.0))
	tileYMax := int(math.Floor((topLeftGlobalY + float64(height)) / 256.0))

	cols := tileXMax - tileXMin + 1
	rows := tileYMax - tileYMin + 1

	type tileData struct {
		tx, ty int
		img    image.Image
	}

	tileChan := make(chan tileData, cols*rows)
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	for ty := tileYMin; ty <= tileYMax; ty++ {
		for tx := tileXMin; tx <= tileXMax; tx++ {
			wg.Add(1)
			go func(x, y int) {
				defer wg.Done()
				img := fetchTile(ctx, zoom, x, y)
				if img != nil {
					tileChan <- tileData{tx: x, ty: y, img: img}
				}
			}(tx, ty)
		}
	}

	wg.Wait()
	close(tileChan)

	outImg := image.NewRGBA(image.Rect(0, 0, width, height))
	// Default background
	draw.Draw(outImg, outImg.Bounds(), &image.Uniform{C: color.RGBA{R: 240, G: 240, B: 240, A: 255}}, image.Point{}, draw.Src)

	for td := range tileChan {
		tileGlobalX := float64(td.tx * 256)
		tileGlobalY := float64(td.ty * 256)

		destX := int(math.Round(tileGlobalX - topLeftGlobalX))
		destY := int(math.Round(tileGlobalY - topLeftGlobalY))

		draw.Draw(outImg, image.Rect(destX, destY, destX+256, destY+256), td.img, image.Point{}, draw.Over)
	}

	// Draw Red Pin Markers for all detected points
	if loc != nil {
		for _, pt := range loc.Points {
			gx, gy := latLonToGlobalPixel(pt.Lat, pt.Lon, zoom)
			px := int(math.Round(gx - topLeftGlobalX))
			py := int(math.Round(gy - topLeftGlobalY))

			if px >= -50 && px <= width+50 && py >= -50 && py <= height+50 {
				drawRedPushpin(outImg, px, py)
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, outImg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// fetchTile downloads a tile from the official OpenStreetMap tile mirrors.
func fetchTile(ctx context.Context, zoom, x, y int) image.Image {
	mirrors := []string{
		fmt.Sprintf("https://tile.openstreetmap.org/%d/%d/%d.png", zoom, x, y),
		fmt.Sprintf("https://a.tile.openstreetmap.org/%d/%d/%d.png", zoom, x, y),
		fmt.Sprintf("https://b.tile.openstreetmap.org/%d/%d/%d.png", zoom, x, y),
	}

	for _, u := range mirrors {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "KyivAirAlertMonitor/1.0 (https://github.com/alert-userbot)")

		resp, err := httpClient.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode == http.StatusOK {
			img, err := png.Decode(resp.Body)
			resp.Body.Close()
			if err == nil {
				return img
			}
		} else {
			resp.Body.Close()
		}
	}
	return nil
}

func latLonToGlobalPixel(lat, lon float64, zoom int) (float64, float64) {
	n := math.Pow(2, float64(zoom)) * 256.0
	gx := (lon + 180.0) / 360.0 * n
	latRad := lat * math.Pi / 180.0
	gy := (1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n
	return gx, gy
}

// drawRedPushpin renders a clean red pushpin marker with drop shadow at (x, y).
func drawRedPushpin(img *image.RGBA, x, y int) {
	headY := y - 24
	radius := 12

	// Drop shadow
	drawFilledCircle(img, x+3, y+2, 6, color.RGBA{R: 0, G: 0, B: 0, A: 90})

	// Needle pointer triangle
	fillTriangle(img,
		image.Point{X: x - 6, Y: headY + 5},
		image.Point{X: x + 6, Y: headY + 5},
		image.Point{X: x, Y: y},
		color.RGBA{R: 217, G: 48, B: 37, A: 255})

	// Red head
	drawFilledCircle(img, x, headY, radius, color.RGBA{R: 217, G: 48, B: 37, A: 255})
	drawCircleBorder(img, x, headY, radius, color.RGBA{R: 180, G: 20, B: 20, A: 255})

	// White inner dot
	drawFilledCircle(img, x, headY, 4, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}

func drawFilledCircle(img *image.RGBA, cx, cy, r int, col color.RGBA) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				blendPixel(img, cx+dx, cy+dy, col)
			}
		}
	}
}

func drawCircleBorder(img *image.RGBA, cx, cy, r int, col color.RGBA) {
	for theta := 0.0; theta < 2*math.Pi; theta += 0.03 {
		px := int(float64(cx) + float64(r)*math.Cos(theta))
		py := int(float64(cy) + float64(r)*math.Sin(theta))
		blendPixel(img, px, py, col)
	}
}

func fillTriangle(img *image.RGBA, p1, p2, p3 image.Point, col color.RGBA) {
	minX := int(math.Min(float64(p1.X), math.Min(float64(p2.X), float64(p3.X))))
	maxX := int(math.Max(float64(p1.X), math.Max(float64(p2.X), float64(p3.X))))
	minY := int(math.Min(float64(p1.Y), math.Min(float64(p2.Y), float64(p3.Y))))
	maxY := int(math.Max(float64(p1.Y), math.Max(float64(p2.Y), float64(p3.Y))))

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if pointInTriangle(image.Point{X: x, Y: y}, p1, p2, p3) {
				blendPixel(img, x, y, col)
			}
		}
	}
}

func pointInTriangle(p, a, b, c image.Point) bool {
	d1 := sign(p, a, b)
	d2 := sign(p, b, c)
	d3 := sign(p, c, a)

	hasNeg := (d1 < 0) || (d2 < 0) || (d3 < 0)
	hasPos := (d1 > 0) || (d2 > 0) || (d3 > 0)

	return !(hasNeg && hasPos)
}

func sign(p1, p2, p3 image.Point) int {
	return (p1.X-p3.X)*(p2.Y-p3.Y) - (p2.X-p3.X)*(p1.Y-p3.Y)
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
