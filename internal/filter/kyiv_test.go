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
			name:     "Kyiv city Ukrainian no dot",
			payload:  `{"region": "Київ", "status": "active"}`,
			expected: true,
		},
		{
			name:     "Kyiv English",
			payload:  `{"region": "Kyiv", "status": "alert"}`,
			expected: true,
		},
		{
			name:     "Kyiv lowercase Ukrainian",
			payload:  `{"message": "загроза бпла для київ"}`,
			expected: true,
		},
		{
			name:     "Kyiv Oblast Ukrainian (excluded)",
			payload:  `{"region": "Київська область", "status": "active", "threat": "drone"}`,
			expected: false,
		},
		{
			name:     "Kyiv Oblast English (excluded)",
			payload:  `{"region": "Kyiv Oblast", "type": "air_raid"}`,
			expected: false,
		},
		{
			name:     "Oblast mention even if city present (excluded)",
			payload:  `{"region": "Київська область, м. Київ", "status": "active"}`,
			expected: false,
		},
		{
			name:     "Other region (Lviv)",
			payload:  `{"region": "Львівська область", "status": "active"}`,
			expected: false,
		},
		{
			name:     "Other region city",
			payload:  `{"region": "м. Харків", "status": "active"}`,
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

func BenchmarkIsKyivTarget(b *testing.B) {
	filter := NewKyivFilter()
	payload := []byte(`{"id":12345,"region":"м. Київ","threat_type":"missile","title":"Повітряна тривога в Києві","severity":"high"}`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = filter.IsKyivTarget(payload)
	}
}
