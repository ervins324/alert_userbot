package geomap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestLookupStatic(t *testing.T) {
	tests := []struct {
		query       string
		wantFound   bool
		minExpected float64
	}{
		{"Оболонь", true, 50.5},
		{"позняки", true, 50.39},
		{"Троєщина", true, 50.51},
		{"Лук'янівка", true, 50.46},
		{"лукянівка", true, 50.46},
		{"Видубичі", true, 50.40},
		{"Печерськ", true, 50.43},
		{"на Позняках", true, 50.39},
		{"вулиця невідома 12345XYZ", false, 0},
	}

	for _, tt := range tests {
		coords, _, ok := LookupStatic(tt.query)
		if ok != tt.wantFound {
			t.Errorf("LookupStatic(%q) ok = %v, want %v", tt.query, ok, tt.wantFound)
		}
		if ok && coords.Lat < tt.minExpected {
			t.Errorf("LookupStatic(%q) lat = %f, want >= %f", tt.query, coords.Lat, tt.minExpected)
		}
	}
}

func TestPrepareNominatimQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Хрещатик", "Хрещатик, Київ, Україна"},
		{"вул. Саксаганського", "вул. Саксаганського, Київ, Україна"},
		{"вул. Хрещатик, Київ", "вул. Хрещатик, Київ, Україна"},
		{"Оболонь, Київ, Україна", "Оболонь, Київ, Україна"},
		{"/map цирк", "цирк, Київ, Україна"},
	}

	for _, tt := range tests {
		got := prepareNominatimQuery(tt.input)
		if got != tt.want {
			t.Errorf("prepareNominatimQuery(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGeocoderWithNominatimMockAndSyncMapCache(t *testing.T) {
	var requestCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		q := r.URL.Query().Get("q")
		if q == "" {
			http.Error(w, "missing q", http.StatusBadRequest)
			return
		}

		if r.Header.Get("User-Agent") == "" {
			http.Error(w, "missing User-Agent", http.StatusBadRequest)
			return
		}

		// Mock responses
		if q == "вул. невідома 9999, Київ, Україна" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]any{})
			return
		}

		resp := []map[string]any{
			{
				"place_id":     12345,
				"lat":          "50.4350",
				"lon":          "30.5050",
				"display_name": "вулиця Саксаганського, 15, Київ, Україна",
				"name":         "вулиця Саксаганського",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	geocoder := NewGeocoder(ts.Client(), nil)
	geocoder.baseURL = ts.URL
	geocoder.minReqGap = 10 * time.Millisecond // fast testing

	ctx := context.Background()

	// 1. Static dictionary lookup should NOT make HTTP requests
	staticCoords, _, err := geocoder.Geocode(ctx, "Позняки")
	if err != nil {
		t.Fatalf("Geocode(Позняки) error = %v", err)
	}
	if staticCoords == nil || staticCoords.Lat == 0 {
		t.Fatalf("expected valid coordinates for Pozniaky")
	}
	if atomic.LoadInt32(&requestCount) != 0 {
		t.Errorf("expected 0 HTTP requests for static dictionary match, got %d", requestCount)
	}

	// 2. Query not in static dictionary should hit mock Nominatim
	coords, name, err := geocoder.Geocode(ctx, "вулиця Саксаганського 15")
	if err != nil {
		t.Fatalf("Geocode(вулиця Саксаганського 15) error = %v", err)
	}
	if coords.Lat < 50.40 || coords.Lon < 30.40 {
		t.Errorf("unexpected coords: %v", coords)
	}
	if name != "вулиця Саксаганського" {
		t.Errorf("unexpected name: %s", name)
	}
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected 1 HTTP request, got %d", requestCount)
	}

	// 3. Second query for the same location must be served from sync.Map cache without HTTP request
	coordsCached, _, err := geocoder.Geocode(ctx, "вулиця Саксаганського 15")
	if err != nil {
		t.Fatalf("Geocode cached error = %v", err)
	}
	if coordsCached.Lat != coords.Lat || coordsCached.Lon != coords.Lon {
		t.Errorf("cached coords mismatch: %v vs %v", coordsCached, coords)
	}
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected still 1 HTTP request due to sync.Map cache, got %d", requestCount)
	}

	// 4. Test negative lookup caching
	_, _, err = geocoder.Geocode(ctx, "вул. невідома 9999")
	if err != ErrLocationNotFound {
		t.Errorf("expected ErrLocationNotFound, got %v", err)
	}
	if atomic.LoadInt32(&requestCount) != 2 {
		t.Errorf("expected 2 HTTP requests, got %d", requestCount)
	}

	// Repeated negative query should also be served from cache
	_, _, err = geocoder.Geocode(ctx, "вул. невідома 9999")
	if err != ErrLocationNotFound {
		t.Errorf("expected ErrLocationNotFound from cache, got %v", err)
	}
	if atomic.LoadInt32(&requestCount) != 2 {
		t.Errorf("expected still 2 HTTP requests due to negative cache, got %d", requestCount)
	}
}

func TestGeocoderRateLimiting(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := []map[string]any{
			{
				"place_id":     1,
				"lat":          "50.45",
				"lon":          "30.50",
				"display_name": "Test",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	geocoder := NewGeocoder(ts.Client(), nil)
	geocoder.baseURL = ts.URL
	geocoder.minReqGap = 150 * time.Millisecond // 150ms test gap

	start := time.Now()
	ctx := context.Background()

	_, _, _ = geocoder.Geocode(ctx, "test query one 123")
	_, _, _ = geocoder.Geocode(ctx, "test query two 456")

	elapsed := time.Since(start)
	if elapsed < 140*time.Millisecond {
		t.Errorf("expected rate limiter to enforce gap, elapsed %v", elapsed)
	}
}
