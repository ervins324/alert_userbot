package geomap

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"alert-userbot/internal/geoparse"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// RenderKyivMap generates a clean Google Maps static map with markers placed via the Google Maps API.
// If the message contains multiple places, all of them are marked on the same map.
func RenderKyivMap(loc *geoparse.LocationResult, apiKey string) ([]byte, error) {
	if apiKey != "" {
		return fetchGoogleStaticMap(loc, apiKey)
	}

	// Fallback to pure OpenStreetMap static API without custom overlays or top banners
	return fetchOSMStaticMap(loc)
}

// fetchGoogleStaticMap requests a clean, official Google Maps image with native markers added through API parameters.
func fetchGoogleStaticMap(loc *geoparse.LocationResult, apiKey string) ([]byte, error) {
	baseURL := "https://maps.googleapis.com/maps/api/staticmap"
	params := url.Values{}

	params.Set("size", "640x500")
	params.Set("scale", "2") // High-resolution 1280x1000 retina
	params.Set("maptype", "roadmap")
	params.Set("format", "png")
	params.Set("key", apiKey)

	if loc == nil || len(loc.Points) == 0 {
		// Default to Kyiv city overview
		params.Set("center", "50.4501,30.5234")
		params.Set("zoom", "11")
	} else if len(loc.Points) == 1 {
		pt := loc.Points[0]
		params.Set("center", fmt.Sprintf("%.5f,%.5f", pt.Lat, pt.Lon))
		params.Set("zoom", "12")
		params.Add("markers", fmt.Sprintf("color:red|size:mid|%.5f,%.5f", pt.Lat, pt.Lon))
	} else {
		// Multiple places detected: add a marker for each place on the single image.
		// Google Maps automatically computes the optimal center and zoom level to display all markers!
		var coords []string
		for _, pt := range loc.Points {
			coords = append(coords, fmt.Sprintf("%.5f,%.5f", pt.Lat, pt.Lon))
		}
		params.Add("markers", "color:red|size:mid|"+strings.Join(coords, "|"))
	}

	reqURL := baseURL + "?" + params.Encode()
	resp, err := httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("google maps request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google maps api returned status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// fetchOSMStaticMap provides a clean map when no Google API key is configured.
func fetchOSMStaticMap(loc *geoparse.LocationResult) ([]byte, error) {
	centerLat := 50.4501
	centerLon := 30.5234
	zoom := 11

	if loc != nil && len(loc.Points) == 1 {
		centerLat = loc.Points[0].Lat
		centerLon = loc.Points[0].Lon
		zoom = 12
	}

	var markerParams []string
	if loc != nil {
		for _, pt := range loc.Points {
			markerParams = append(markerParams, fmt.Sprintf("%.5f,%.5f,red-pushpin", pt.Lat, pt.Lon))
		}
	}

	reqURL := fmt.Sprintf("https://staticmap.openstreetmap.de/staticmap.php?center=%.5f,%.5f&zoom=%d&size=1000x800&maptype=mapnik",
		centerLat, centerLon, zoom)
	if len(markerParams) > 0 {
		reqURL += "&markers=" + url.QueryEscape(strings.Join(markerParams, "|"))
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "KyivAirAlertMonitor/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("static map returned status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}
