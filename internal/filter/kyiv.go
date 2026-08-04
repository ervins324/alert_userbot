package filter

import (
	"bytes"
)

// KyivFilter provides zero-allocation matching for Kyiv and Kyiv Oblast threats.
type KyivFilter struct {
	patterns [][]byte
}

// NewKyivFilter initializes a new filter with pre-compiled byte patterns.
func NewKyivFilter() *KyivFilter {
	return &KyivFilter{
		patterns: [][]byte{
			[]byte("Київ"),
			[]byte("київ"),
			[]byte("КИЇВ"),
			[]byte("Kyiv"),
			[]byte("kyiv"),
			[]byte("KYIV"),
			[]byte("Київськ"),
			[]byte("київськ"),
			[]byte("КИЇВСЬК"),
			[]byte("Kiev"),
			[]byte("kiev"),
			[]byte("KIEV"),
			// Common URL encoded or escaped representations if present
			[]byte("%D0%9A%D0%B8%D1%96%D0%B2"),
		},
	}
}

// MatchResult encapsulates zero-allocation filter inspection results.
type MatchResult struct {
	Matched      bool
	IsCity       bool
	IsOblast     bool
	MatchedLabel string
}

// IsKyivTarget returns true if the payload contains any reference to Kyiv or Kyiv Oblast.
// Zero heap allocations.
func (f *KyivFilter) IsKyivTarget(payload []byte) bool {
	if len(payload) == 0 {
		return false
	}

	for _, pattern := range f.patterns {
		if bytes.Contains(payload, pattern) {
			return true
		}
	}
	return false
}

// Inspect analyzes the payload and returns specific location matching metadata.
func (f *KyivFilter) Inspect(payload []byte) MatchResult {
	if !f.IsKyivTarget(payload) {
		return MatchResult{Matched: false}
	}

	isOblast := bytes.Contains(payload, []byte("область")) ||
		bytes.Contains(payload, []byte("обл")) ||
		bytes.Contains(payload, []byte("Oblast")) ||
		bytes.Contains(payload, []byte("oblast")) ||
		bytes.Contains(payload, []byte("Київськ")) ||
		bytes.Contains(payload, []byte("київськ"))

	label := "м. Київ / Київська область"
	if isOblast {
		label = "Київська область"
	} else if bytes.Contains(payload, []byte("м. Київ")) || bytes.Contains(payload, []byte("м.Київ")) || bytes.Contains(payload, []byte("City")) {
		label = "м. Київ"
	}

	return MatchResult{
		Matched:      true,
		IsCity:       !isOblast,
		IsOblast:     isOblast,
		MatchedLabel: label,
	}
}
