package geoparse

import (
	"testing"
)

func TestExtractLocation(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		wantMatchedRaion RaionID
		wantSpecific     bool
		wantNeighborhood string
	}{
		{
			name:             "Darnytskyi raion mention",
			input:            "Увага! Ракета курсом на Дарницький район!",
			wantMatchedRaion: RaionDarnytskyi,
			wantSpecific:     false,
		},
		{
			name:             "Darnytsia neighborhood Pozniaky",
			input:            "БпЛА фіксується в районі Позняків",
			wantMatchedRaion: RaionDarnytskyi,
			wantSpecific:     true,
			wantNeighborhood: "Позняки",
		},
		{
			name:             "TEC-5 power plant",
			input:            "⚠️З Позняків на ТЕЦ-5.",
			wantMatchedRaion: RaionPecherskyi,
			wantSpecific:     true,
			wantNeighborhood: "ТЕЦ-5",
		},
		{
			name:             "TEC-6 power plant",
			input:            "⚠️Реактивний шахед на Троєщину/ТЕЦ-6",
			wantMatchedRaion: RaionDesnyanskyi,
			wantSpecific:     true,
			wantNeighborhood: "ТЕЦ-6",
		},
		{
			name:             "River Mall shopping mall",
			input:            "Рівер мол! ПАДАЄ",
			wantMatchedRaion: RaionDarnytskyi,
			wantSpecific:     true,
			wantNeighborhood: "ТРЦ River Mall",
		},
		{
			name:             "TRC Respublika",
			input:            "❗️2 крилаті ракети на Республіку/Теремки.",
			wantMatchedRaion: RaionHolosiivskyi,
			wantSpecific:     true,
			wantNeighborhood: "ТРЦ Республіка",
		},
		{
			name:             "TRC Lavina",
			input:            "⚠️Реактивний шахед на ТРЦ Лавіна.",
			wantMatchedRaion: RaionSviatoshyn,
			wantSpecific:     true,
			wantNeighborhood: "ТРЦ Лавіна",
		},
		{
			name:             "TRC Retroville",
			input:            "⚠️Реактивний шахед на ТРЦ Ретровіль.",
			wantMatchedRaion: RaionPodilskyi,
			wantSpecific:     true,
			wantNeighborhood: "ТРЦ Retroville",
		},
		{
			name:             "Nyzhni Sady",
			input:            "⚠️Реактивний шахед знову над Бортничі курсом на Нижні Сади.",
			wantMatchedRaion: RaionDarnytskyi,
			wantSpecific:     true,
			wantNeighborhood: "Нижні Сади",
		},
		{
			name:             "Rembaza",
			input:            "Реактивний Шахед курсом на Рембазу.",
			wantMatchedRaion: RaionDarnytskyi,
			wantSpecific:     true,
			wantNeighborhood: "Рембаза",
		},
		{
			name:             "Boryspil and Brovary suburbs",
			input:            "⚠️Реактивний шахед на Бровари.",
			wantMatchedRaion: RaionDesnyanskyi,
			wantSpecific:     true,
			wantNeighborhood: "Бровари",
		},
		{
			name:             "Vyshneve suburb",
			input:            "⚠️Реактивний шахед на Вишневе.",
			wantMatchedRaion: RaionSviatoshyn,
			wantSpecific:     true,
			wantNeighborhood: "Вишневе",
		},
		{
			name:             "Pivnichnyi bridge",
			input:            "⚠️Реактивний шахед падає на північний міст.",
			wantMatchedRaion: RaionDesnyanskyi,
			wantSpecific:     true,
			wantNeighborhood: "Північний міст",
		},
		{
			name:             "Podilskyi bridge",
			input:            "❗️Реактивний шахед падає на Подільський міст.",
			wantMatchedRaion: RaionPodilskyi,
			wantSpecific:     true,
			wantNeighborhood: "Подільський міст",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ExtractLocation(tt.input)
			if res == nil {
				t.Fatalf("ExtractLocation(%q) returned nil", tt.input)
			}
			found := false
			for _, r := range res.MatchedRaions {
				if r == tt.wantMatchedRaion {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("ExtractLocation(%q) matched %v, wanted raion %v", tt.input, res.MatchedRaions, tt.wantMatchedRaion)
			}
			if res.HasSpecificPoint != tt.wantSpecific {
				t.Errorf("HasSpecificPoint = %v, wanted %v", res.HasSpecificPoint, tt.wantSpecific)
			}
			if tt.wantNeighborhood != "" && res.NeighborhoodName != tt.wantNeighborhood {
				t.Errorf("NeighborhoodName = %q, wanted %q", res.NeighborhoodName, tt.wantNeighborhood)
			}
		})
	}
}
