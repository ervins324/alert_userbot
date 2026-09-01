package geomap

import (
	"testing"

	"alert-userbot/internal/geoparse"
)

func TestRenderKyivMapDistrictArea(t *testing.T) {
	// Test rendering district area (e.g. Darnytskyi district polygon)
	loc := &geoparse.LocationResult{
		MatchedRaions: []geoparse.RaionID{geoparse.RaionDarnytskyi},
		Description:   "Дарницький район",
	}

	data, err := RenderKyivMap(loc)
	if err != nil {
		t.Fatalf("RenderKyivMap failed for district area: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("RenderKyivMap returned empty PNG data")
	}
}

func TestRenderKyivMapMultiPoints(t *testing.T) {
	// Test rendering multiple point markers on OpenStreetMap
	loc := &geoparse.LocationResult{
		MatchedRaions: []geoparse.RaionID{geoparse.RaionDarnytskyi, geoparse.RaionDesnyanskyi},
		Points: []geoparse.PointMatch{
			{NameUA: "Позняки", Raion: geoparse.RaionDarnytskyi, Lat: 50.3980, Lon: 30.6340},
			{NameUA: "Троєщина", Raion: geoparse.RaionDesnyanskyi, Lat: 50.5180, Lon: 30.6020},
		},
		Description: "Позняки, Троєщина",
	}

	data, err := RenderKyivMap(loc)
	if err != nil {
		t.Fatalf("RenderKyivMap failed for multi-points: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("RenderKyivMap returned empty PNG data")
	}
}
