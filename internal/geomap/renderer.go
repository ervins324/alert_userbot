package geomap

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"

	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"

	"alert-userbot/internal/geoparse"
)

// RenderKyivMap generates an OpenStreetMap static map using go-staticmaps.
// - When a district/region is requested (e.g. Darnytskyi), highlights its boundary area.
// - When specific landmarks or locations are requested, places marker pins.
// - When multiple places are detected, displays all areas and markers on the same map.
func RenderKyivMap(loc *geoparse.LocationResult) ([]byte, error) {
	ctx := sm.NewContext()
	ctx.SetTileProvider(sm.NewTileProviderOpenStreetMaps())
	ctx.SetSize(1000, 750)
	ctx.SetUserAgent("KyivAirAlertMonitor/1.0 (https://github.com/alert-userbot)")

	hasObjects := false

	// 1. Highlight District Areas (Polygons) on OpenStreetMap
	if loc != nil && len(loc.MatchedRaions) > 0 {
		areaFill := color.RGBA{R: 235, G: 50, B: 65, A: 70}
		areaBorder := color.RGBA{R: 217, G: 48, B: 37, A: 240}

		for _, rID := range loc.MatchedRaions {
			if poly, ok := KyivRaionBoundaries[rID]; ok && len(poly.Boundary) > 0 {
				var positions []s2.LatLng
				for _, pt := range poly.Boundary {
					positions = append(positions, s2.LatLngFromDegrees(pt.Lat, pt.Lon))
				}
				area := sm.NewArea(positions, areaFill, areaBorder, 3.0)
				ctx.AddObject(area)
				hasObjects = true
			}
		}
	}

	// 2. Add Marker Pins for specific landmarks / points
	if loc != nil && len(loc.Points) > 0 {
		pinColor := color.RGBA{R: 217, G: 48, B: 37, A: 255}

		for _, pt := range loc.Points {
			marker := sm.NewMarker(s2.LatLngFromDegrees(pt.Lat, pt.Lon), pinColor, 16.0)
			ctx.AddObject(marker)
			hasObjects = true
		}
	}

	// 3. If no specific objects, center on Kyiv city overview
	if !hasObjects {
		ctx.SetCenter(s2.LatLngFromDegrees(50.4501, 30.5234))
		ctx.SetZoom(11)
	}

	// 4. Render map using go-staticmaps
	img, err := ctx.Render()
	if err != nil {
		return nil, fmt.Errorf("go-staticmaps render failed: %w", err)
	}

	// 5. Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("png encode failed: %w", err)
	}

	return buf.Bytes(), nil
}
