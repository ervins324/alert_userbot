package geoparse

import (
	"testing"
)

func TestExtractLocation(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantPoints   int
		wantContains []string
	}{
		{
			name:         "Single place: Pozniaky",
			input:        "БпЛА фіксується в районі Позняків",
			wantPoints:   1,
			wantContains: []string{"Позняки"},
		},
		{
			name:         "Multiple places: Pozniaky and Troieshchyna",
			input:        "Шахеди на Позняках та Троєщині!",
			wantPoints:   2,
			wantContains: []string{"Позняки", "Троєщина"},
		},
		{
			name:         "Multiple places from alert history: Obolon, Lukianivka, Syrets",
			input:        "⚠️1 реактивний шахед з Оболоні на Лук'янівку, Сирець.",
			wantPoints:   3,
			wantContains: []string{"Оболонь", "Лук'янівка", "Сирець"},
		},
		{
			name:         "Multiple places: TEC-5 and Berezniaky",
			input:        "⚠️1 реактивний шахед над ТЕЦ-5 курсом на Позняки, Березняки.",
			wantPoints:   3,
			wantContains: []string{"ТЕЦ-5", "Позняки", "Березняки"},
		},
		{
			name:         "Multiple places: Vynohradar, Nyvky, Berkovets",
			input:        "⚠️Реактивний шахед з Виноградар на Нивки, Берковець.",
			wantPoints:   3,
			wantContains: []string{"Виноградар", "Нивки", "Берковець"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ExtractLocation(tt.input)
			if res == nil {
				t.Fatalf("ExtractLocation(%q) returned nil", tt.input)
			}
			if len(res.Points) < tt.wantPoints {
				t.Errorf("ExtractLocation(%q) got %d points, wanted at least %d", tt.input, len(res.Points), tt.wantPoints)
			}
			for _, expected := range tt.wantContains {
				found := false
				for _, pt := range res.Points {
					if pt.NameUA == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ExtractLocation(%q) did not contain point %q (got: %v)", tt.input, expected, res.Description)
				}
			}
		})
	}
}
