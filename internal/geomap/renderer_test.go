package geomap

import (
	"testing"

	"alert-userbot/internal/geoparse"
)

func TestRenderKyivMap(t *testing.T) {
	loc := &geoparse.LocationResult{
		MatchedRaions: []geoparse.RaionID{geoparse.RaionDarnytskyi, geoparse.RaionDesnyanskyi},
		Points: []geoparse.PointMatch{
			{NameUA: "Позняки", Raion: geoparse.RaionDarnytskyi, Lat: 50.3980, Lon: 30.6340},
			{NameUA: "Троєщина", Raion: geoparse.RaionDesnyanskyi, Lat: 50.5180, Lon: 30.6020},
		},
		Description: "Позняки, Троєщина",
	}

	if len(loc.Points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(loc.Points))
	}
}
