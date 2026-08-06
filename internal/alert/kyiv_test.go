package alert

import (
	"testing"
)

func TestKyivAlertActive(t *testing.T) {
	cases := []struct {
		name     string
		frame    string
		expected bool
	}{
		{
			name:     "city raion active (місто Київ)",
			frame:    `{"type":"alerts","data":{"raions":[{"key":"оболонський","name":"Оболонський район","oblast":"місто Київ","since":"2026-08-06T05:00:00Z"}]}}`,
			expected: true,
		},
		{
			name:     "city raion active (м. Київ)",
			frame:    `{"type":"alerts","data":{"raions":[{"key":"печерський","name":"Печерський район","oblast":"м. Київ"}]}}`,
			expected: true,
		},
		{
			name:     "city raion key fallback",
			frame:    `{"type":"alerts","data":{"raions":[{"key":"шевченківський","name":"Шевченківський район","oblast":"Київ"}]}}`,
			expected: true,
		},
		{
			name:     "Kyiv Oblast raion excluded",
			frame:    `{"type":"alerts","data":{"raions":[{"key":"бориспільський","name":"Бориспільський район","oblast":"Київська область"}]}}`,
			expected: false,
		},
		{
			name:     "city-like key but different oblast excluded",
			frame:    `{"type":"alerts","data":{"raions":[{"key":"шевченківський","name":"Шевченківський район","oblast":"Харківська область"}]}}`,
			expected: false,
		},
		{
			name:     "no kyiv raions",
			frame:    `{"type":"alerts","data":{"raions":[{"key":"львівський","name":"Львівський район","oblast":"Львівська область"}]}}`,
			expected: false,
		},
		{
			name:     "empty raions",
			frame:    `{"type":"alerts","data":{"version":1785871257,"raions":[]}}`,
			expected: false,
		},
		{
			name:     "heartbeat frame ignored",
			frame:    `{"type":"heartbeat","ts":"2026-08-06T05:02:31Z"}`,
			expected: false,
		},
		{
			name:     "threats frame ignored",
			frame:    `{"type":"threats","data":{"threats":[]}}`,
			expected: false,
		},
		{
			name:     "invalid json",
			frame:    `{not json`,
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := KyivAlertActive([]byte(tc.frame)); got != tc.expected {
				t.Errorf("KyivAlertActive() = %v; want %v", got, tc.expected)
			}
		})
	}
}

func TestAlertStateTransitions(t *testing.T) {
	st := NewKyivAlertState(nil)
	if st.IsActive() {
		t.Fatal("expected inactive initially")
	}

	st.SetActive(true)
	if !st.IsActive() {
		t.Fatal("expected active after SetActive(true)")
	}

	// Duplicate set should not cause issues
	st.SetActive(true)
	if !st.IsActive() {
		t.Fatal("expected still active")
	}

	st.SetActive(false)
	if st.IsActive() {
		t.Fatal("expected inactive after SetActive(false)")
	}
}
