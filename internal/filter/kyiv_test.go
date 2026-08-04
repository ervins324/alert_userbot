package filter

import (
	"testing"
)

func TestKyivFilterMatches(t *testing.T) {
	filter := NewKyivFilter()

	testCases := []struct {
		name     string
		payload  string
		expected bool
	}{
		{
			name:     "Kyiv city Ukrainian",
			payload:  `{"region": "м. Київ", "status": "active", "threat": "missile"}`,
			expected: true,
		},
		{
			name:     "Kyiv Oblast Ukrainian",
			payload:  `{"region": "Київська область", "status": "active", "threat": "drone"}`,
			expected: true,
		},
		{
			name:     "Kyiv English",
			payload:  `{"region": "Kyiv", "status": "alert"}`,
			expected: true,
		},
		{
			name:     "Kyiv Oblast English",
			payload:  `{"region": "Kyiv Oblast", "type": "air_raid"}`,
			expected: true,
		},
		{
			name:     "Kyiv region lowercase Ukrainian",
			payload:  `{"message": "загроза бпла для київщини та м.київ"}`,
			expected: true,
		},
		{
			name:     "Other region (Lviv)",
			payload:  `{"region": "Львівська область", "status": "active"}`,
			expected: false,
		},
		{
			name:     "Ping payload",
			payload:  `{"type": "ping", "ts": 1700000000}`,
			expected: false,
		},
		{
			name:     "Empty payload",
			payload:  "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filter.IsKyivTarget([]byte(tc.payload))
			if result != tc.expected {
				t.Errorf("IsKyivTarget(%q) = %v; want %v", tc.payload, result, tc.expected)
			}
		})
	}
}

func TestInspect(t *testing.T) {
	filter := NewKyivFilter()

	resCity := filter.Inspect([]byte(`{"region": "м. Київ"}`))
	if !resCity.Matched || resCity.MatchedLabel != "м. Київ" {
		t.Errorf("Expected city match, got %+v", resCity)
	}

	resOblast := filter.Inspect([]byte(`{"region": "Київська область"}`))
	if !resOblast.Matched || !resOblast.IsOblast {
		t.Errorf("Expected oblast match, got %+v", resOblast)
	}
}

func BenchmarkIsKyivTarget(b *testing.B) {
	filter := NewKyivFilter()
	payload := []byte(`{"id":12345,"region":"Київська область","threat_type":"drone","title":"Повітряна тривога в Київській області","severity":"high"}`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = filter.IsKyivTarget(payload)
	}
}
