package filter

import "strings"

// GeoFilter skips messages that mention only excluded geographic regions
// and do not mention Kyiv. By default (even with an empty exclusion list),
// it operates in "Kyiv-only" mode: any message that contains a recognized
// non-Kyiv oblast/region name and does NOT mention Kyiv is skipped.
//
// When ExcludedRegions are configured, only those specific regions trigger
// filtering (other non-Kyiv regions pass through).
type GeoFilter struct {
	excludedStems []string // lowercased stems, e.g. "полтав", "сум"
	kyivOnly      bool     // true when no explicit exclusions are set
}

// kyivPassStems are Kyiv-related stems that always pass through the filter.
var kyivPassStems = []string{
	"київ",
	"києв",
	"києві",
	"столиц",
}

// nonKyivOblastStems are stems for Ukrainian oblasts/cities outside Kyiv.
// Used in "Kyiv-only" mode to detect non-Kyiv regional content.
var nonKyivOblastStems = []string{
	"полтав",
	"сумськ", "сумщин", "суми",
	"харків", "харьків",
	"одеськ", "одес",
	"дніпропетровськ", "дніпро",
	"запоріз", "запоріж",
	"львів", "львівськ",
	"вінниц",
	"житомир",
	"черкас",
	"чернігів",
	"чернівц",
	"хмельниц",
	"волинськ", "волині",
	"рівненськ", "рівне",
	"тернопіл",
	"івано-франків",
	"закарпат",
	"миколаїв",
	"херсон",
	"кіровоград", "кропивниц",
	"донеччин", "донецьк",
	"луганськ", "луганщин",
}

// NewGeoFilter builds a geographic filter.
// If regions is empty, it defaults to "Kyiv-only" mode where all non-Kyiv
// oblast mentions are filtered. If regions is non-empty, only those specific
// region stems are filtered.
func NewGeoFilter(regions []string) *GeoFilter {
	var stems []string
	for _, r := range regions {
		r = strings.ToLower(strings.TrimSpace(r))
		if r != "" {
			stems = append(stems, r)
		}
	}

	if len(stems) == 0 {
		return &GeoFilter{
			excludedStems: nonKyivOblastStems,
			kyivOnly:      true,
		}
	}
	return &GeoFilter{
		excludedStems: stems,
		kyivOnly:      false,
	}
}

// ShouldSkip reports whether the message should be filtered out.
// A message is skipped if it mentions an excluded region and does NOT
// mention Kyiv.
func (g *GeoFilter) ShouldSkip(text string) bool {
	if g == nil || len(g.excludedStems) == 0 {
		return false
	}

	lower := strings.ToLower(text)

	// If the message mentions Kyiv, always pass through.
	for _, stem := range kyivPassStems {
		if strings.Contains(lower, stem) {
			return false
		}
	}

	// If the message mentions any excluded region, skip it.
	for _, stem := range g.excludedStems {
		if strings.Contains(lower, stem) {
			return true
		}
	}

	return false
}
