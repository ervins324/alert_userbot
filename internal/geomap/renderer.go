package geomap

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"
	"math"

	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"

	"alert-userbot/internal/geoparse"
)

// RenderKyivMap generates an OpenStreetMap static map using go-staticmaps.
// - Markers are omitted; the map zooms in and frames the requested location.
// - District areas are drawn with border outline only (no background fill).
func RenderKyivMap(loc *geoparse.LocationResult) ([]byte, error) {
	ctx := sm.NewContext()
	ctx.SetTileProvider(sm.NewTileProviderOpenStreetMaps())
	ctx.SetSize(1000, 750)
	ctx.SetUserAgent("KyivAirAlertMonitor/1.0 (https://github.com/alert-userbot)")

	hasObjects := false

	// 1. Highlight District Areas: Border outline only (no fill)
	if loc != nil && len(loc.MatchedRaions) > 0 {
		borderColor := color.RGBA{R: 217, G: 48, B: 37, A: 245} // Bold Red border outline

		for _, rID := range loc.MatchedRaions {
			if poly, ok := KyivRaionBoundaries[rID]; ok && len(poly.Boundary) > 0 {
				var positions []s2.LatLng
				for _, pt := range poly.Boundary {
					positions = append(positions, s2.LatLngFromDegrees(pt.Lat, pt.Lon))
				}
				// Draw outline path with 3.5px width and zero fill
				path := sm.NewPath(positions, borderColor, 3.5)
				ctx.AddObject(path)
				hasObjects = true
			}
		}
	}

	// 2. Zoom to requested specific locations (without drawing pin markers)
	if loc != nil && len(loc.Points) > 0 {
		if len(loc.Points) == 1 && len(loc.MatchedRaions) == 0 {
			// Single specific location: zoom directly into the neighborhood (zoom 14)
			pt := loc.Points[0]
			ctx.SetCenter(s2.LatLngFromDegrees(pt.Lat, pt.Lon))
			ctx.SetZoom(14)
		} else if len(loc.Points) > 1 && len(loc.MatchedRaions) == 0 {
			// Multiple specific locations: calculate bounding center and zoom to frame all places
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

			centerLat := sumLat / float64(len(loc.Points))
			centerLon := sumLon / float64(len(loc.Points))
			ctx.SetCenter(s2.LatLngFromDegrees(centerLat, centerLon))

			latSpan := maxLat - minLat
			lonSpan := maxLon - minLon
			maxSpan := math.Max(latSpan, lonSpan)

			if maxSpan > 0.30 {
				ctx.SetZoom(10)
			} else if maxSpan > 0.12 {
				ctx.SetZoom(11)
			} else if maxSpan > 0.05 {
				ctx.SetZoom(12)
			} else {
				ctx.SetZoom(13)
			}
		}
	} else if !hasObjects {
		// General overview of Kyiv
		ctx.SetCenter(s2.LatLngFromDegrees(50.4501, 30.5234))
		ctx.SetZoom(11)
	}

	// 3. Render map using go-staticmaps
	img, err := ctx.Render()
	if err != nil {
		return nil, fmt.Errorf("go-staticmaps render failed: %w", err)
	}

	// 4. Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("png encode failed: %w", err)
	}

	return buf.Bytes(), nil
}
