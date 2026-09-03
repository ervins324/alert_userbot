package filter

import "testing"

func TestGeoFilterKyivOnlyMode(t *testing.T) {
	// Default mode (no explicit exclusions) = Kyiv-only
	gf := NewGeoFilter(nil)

	tests := []struct {
		name string
		text string
		skip bool
	}{
		{"Poltava only", "Повітряна тривога у Полтавській області", true},
		{"Sumy only", "Ракетна небезпека Сумщина", true},
		{"Odesa only", "В Одеській області вибухи", true},
		{"Kyiv mention", "Повітряна тривога Київ", false},
		{"Kyiv + Poltava", "Загроза з Полтавської на Київ", false},
		{"No region", "БпЛА в повітрі", false},
		{"Generic drone", "Ударний дрон курсом на захід", false},
		{"Столиця mention", "Загроза для столиці", false},
		{"Empty text", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gf.ShouldSkip(tt.text)
			if got != tt.skip {
				t.Errorf("ShouldSkip(%q) = %v, want %v", tt.text, got, tt.skip)
			}
		})
	}
}

func TestGeoFilterExplicitExclusions(t *testing.T) {
	// Explicit exclusions: only Poltava and Sumy
	gf := NewGeoFilter([]string{"полтав", "сум"})

	tests := []struct {
		name string
		text string
		skip bool
	}{
		{"Poltava", "Тривога у Полтавській області", true},
		{"Sumy", "Ракетна небезпека Сумська область", true},
		{"Odesa passes", "Одеська область тривога", false},
		{"Kyiv always passes", "Загроза Київ з Полтавської", false},
		{"No region", "Вибухи чутно", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gf.ShouldSkip(tt.text)
			if got != tt.skip {
				t.Errorf("ShouldSkip(%q) = %v, want %v", tt.text, got, tt.skip)
			}
		})
	}
}

func TestGeoFilterNilSafe(t *testing.T) {
	var gf *GeoFilter
	if gf.ShouldSkip("Полтава") {
		t.Error("nil GeoFilter should not skip")
	}
}
