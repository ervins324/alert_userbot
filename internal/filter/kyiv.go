package filter

import (
	"bytes"
)

// KyivFilter provides zero-allocation matching for Kyiv city threats only.
type KyivFilter struct {
	patterns   [][]byte // city indicators
	exclusions [][]byte // oblast indicators to reject
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
			[]byte("Kiev"),
			[]byte("kiev"),
			[]byte("KIEV"),
			// Common URL encoded or escaped representations if present
			[]byte("%D0%9A%D0%B8%D1%96%D0%B2"),
		},
		exclusions: [][]byte{
			[]byte("Київськ"), // "Київська область"
			[]byte("київськ"),
			[]byte("КИЇВСЬК"),
			[]byte("область"),
			[]byte("Oblast"),
			[]byte("oblast"),
			[]byte("обл"),
		},
	}
}

// IsKyivTarget returns true if the payload references Kyiv city and not Kyiv Oblast.
// Zero heap allocations.
func (f *KyivFilter) IsKyivTarget(payload []byte) bool {
	if len(payload) == 0 {
		return false
	}

	for _, excl := range f.exclusions {
		if bytes.Contains(payload, excl) {
			return false
		}
	}

	for _, pattern := range f.patterns {
		if bytes.Contains(payload, pattern) {
			return true
		}
	}
	return false
}
