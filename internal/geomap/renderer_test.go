package geomap

import (
	"testing"

	"alert-userbot/internal/geoparse"
)

func TestRenderKyivMap(t *testing.T) {
	loc := &geoparse.LocationResult{
		MatchedRaions:    []geoparse.RaionID{geoparse.RaionDarnytskyi},
		NeighborhoodName: "Позняки",
		Latitude:         50.3980,
		Longitude:        30.6340,
		HasSpecificPoint: true,
		Description:      "Дарницький район (Позняки)",
	}

	data, err := RenderKyivMap(loc, "")
	if err != nil {
		t.Fatalf("RenderKyivMap failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("RenderKyivMap returned empty PNG data")
	}
}
