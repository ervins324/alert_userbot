package geomap

import (
	"bytes"
	"fmt"
	"image/png"
	"math"

	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"

	"alert-userbot/internal/geoparse"
)

// RenderKyivMap generates an OpenStreetMap static map using go-staticmaps.
// The map zooms in and frames the requested location without drawing markers or overlays.
func RenderKyivMap(loc *geoparse.LocationResult) ([]byte, error) {
	ctx := sm.NewContext()
	ctx.SetTileProvider(sm.NewTileProviderOpenStreetMaps())
	ctx.SetSize(1000, 750)
	ctx.SetUserAgent("KyivAirAlertMonitor/1.0 (https://github.com/alert-userbot)")

	// 1. Zoom to requested specific locations (without drawing pin markers)
	if loc != nil && len(loc.Points) > 0 {
		if len(loc.Points) == 1 {
			// Single specific location: zoom directly into the neighborhood (zoom 14)
			pt := loc.Points[0]
			ctx.SetCenter(s2.LatLngFromDegrees(pt.Lat, pt.Lon))
			ctx.SetZoom(14)
		} else {
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
	} else if loc != nil && len(loc.MatchedRaions) > 0 {
		// Raions matched but no specific points: center on raion polygon centers
		sumLat, sumLon := 0.0, 0.0
		count := 0
		for _, rID := range loc.MatchedRaions {
			if poly, ok := KyivRaionBoundaries[rID]; ok {
				sumLat += poly.Center.Lat
				sumLon += poly.Center.Lon
				count++
			}
		}
		if count > 0 {
			ctx.SetCenter(s2.LatLngFromDegrees(sumLat/float64(count), sumLon/float64(count)))
			if count == 1 {
				ctx.SetZoom(13)
			} else {
				ctx.SetZoom(11)
			}
		} else {
			ctx.SetCenter(s2.LatLngFromDegrees(50.4501, 30.5234))
			ctx.SetZoom(11)
		}
	} else {
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
