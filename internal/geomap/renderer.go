package geomap

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"alert-userbot/internal/geoparse"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// RenderKyivMap generates an OpenStreetMap image with native markers for all detected places.
func RenderKyivMap(loc *geoparse.LocationResult) ([]byte, error) {
	return fetchOSMStaticMap(loc)
}

// fetchOSMStaticMap requests a clean OpenStreetMap static map with red pushpin markers for all locations.
func fetchOSMStaticMap(loc *geoparse.LocationResult) ([]byte, error) {
	centerLat := 50.4501
	centerLon := 30.5234
	zoom := 11

	if loc != nil && len(loc.Points) > 0 {
		if len(loc.Points) == 1 {
			centerLat = loc.Points[0].Lat
			centerLon = loc.Points[0].Lon
			zoom = 12
		} else {
			// Compute bounding box and center for multiple locations
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

	var markerParams []string
	if loc != nil {
		for _, pt := range loc.Points {
			markerParams = append(markerParams, fmt.Sprintf("%.5f,%.5f,red-pushpin", pt.Lat, pt.Lon))
		}
	}

	// OpenStreetMap static map endpoints
	endpoints := []string{
		"https://staticmap.openstreetmap.de/staticmap.php",
	}

	var lastErr error
	for _, base := range endpoints {
		reqURL := fmt.Sprintf("%s?center=%.5f,%.5f&zoom=%d&size=1000x800&maptype=mapnik",
			base, centerLat, centerLon, zoom)
		if len(markerParams) > 0 {
			reqURL += "&markers=" + url.QueryEscape(strings.Join(markerParams, "|"))
		}

		req, err := http.NewRequest(http.MethodGet, reqURL, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "KyivAirAlertMonitor/1.0 (https://github.com/alert-userbot)")

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK && len(body) > 0 {
			return body, nil
		}
		lastErr = fmt.Errorf("osm static map returned status %d: %s", resp.StatusCode, string(body))
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("failed to fetch openstreetmap static map")
}
