package filter

import "strings"

// TextFilter skips messages that match any of a set of substrings
// (donation / fundraising posts, etc.).
type TextFilter struct {
	patterns []string
}

// defaultSkipPatterns are common fundraising/donation phrases in Ukrainian.
var defaultSkipPatterns = []string{
	"донат",
	"банка",
	"monobank",
	"send.monobank.ua",
	"збір",
	"збираєм",
	"номер картки",
	"картка банки",
	"картки банки",
	"долучіться",
	"долучитись",
	"підтримаєте",
}

// NewTextFilter builds a filter from the default patterns plus any custom ones.
func NewTextFilter(custom []string) *TextFilter {
	patterns := make([]string, 0, len(defaultSkipPatterns)+len(custom))
	patterns = append(patterns, defaultSkipPatterns...)
	for _, p := range custom {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			patterns = append(patterns, p)
		}
	}
	return &TextFilter{patterns: patterns}
}

// ShouldSkip reports whether the message matches any skip pattern.
func (f *TextFilter) ShouldSkip(text string) bool {
	lower := strings.ToLower(text)
	for _, p := range f.patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
