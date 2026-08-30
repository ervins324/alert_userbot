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
			name:             "Troieshchyna Desnyanskyi",
			input:            "Вибухи на Троєщині, працює ППО",
			wantMatchedRaion: RaionDesnyanskyi,
			wantSpecific:     true,
			wantNeighborhood: "Троєщина",
		},
		{
			name:             "Obolon district",
			input:            "Ціль рухається у напрямку Оболоні",
			wantMatchedRaion: RaionObolonskyi,
			wantSpecific:     true,
			wantNeighborhood: "Оболонь",
		},
		{
			name:             "Borshchahivka Sviatoshynskyi",
			input:            "Шахед над Борщагівкою!",
			wantMatchedRaion: RaionSviatoshyn,
			wantSpecific:     true,
			wantNeighborhood: "Борщагівка",
		},
		{
			name:             "Pechersk district",
			input:            "Печерський район - в укриття!",
			wantMatchedRaion: RaionPecherskyi,
			wantSpecific:     false,
		},
		{
			name:             "Solomianskyi Vidradnyi",
			input:            "Гучно на Відрадному",
			wantMatchedRaion: RaionSolomyanskyi,
			wantSpecific:     true,
			wantNeighborhood: "Відрадний",
		},
		{
			name:             "Left bank generic",
			input:            "Декілька цілей заходять на лівий берег",
			wantMatchedRaion: RaionDarnytskyi,
			wantSpecific:     false,
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
