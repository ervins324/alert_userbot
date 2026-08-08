package alert

import (
	"encoding/json"
	"strings"
)

// alertsFrame mirrors the NEPTUN "alerts" WebSocket frame structure.
type alertsFrame struct {
	Type string `json:"type"`
	Data struct {
		Raions  []alertEntry `json:"raions"`
		Oblasts []alertEntry `json:"oblasts"`
	} `json:"data"`
}

// alertEntry is a single raion or oblast-level alert in the NEPTUN feed.
type alertEntry struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Oblast string `json:"oblast"`
}

// kyivCityRaions are the 10 raions belonging to the city of Kyiv. They are
// unique to the city, which lets us exclude Kyiv Oblast entirely.
var kyivCityRaions = map[string]bool{
	"голосіївський": true,
	"дарницький":    true,
	"деснянський":   true,
	"дніпровський":  true,
	"оболонський":   true,
	"печерський":    true,
	"подільський":   true,
	"святошинський": true,
	"соломянський":  true,
	"шевченківський": true,
}

// KyivAlertActive returns true if the frame indicates an active air alert for
// Kyiv city. Non-"alerts" frames and unparsable frames return false.
func KyivAlertActive(frame []byte) bool {
	var f alertsFrame
	if err := json.Unmarshal(frame, &f); err != nil {
		return false
	}
	if f.Type != "alerts" {
		return false
	}
	// The city of Kyiv is a first-class "oblast" entry in the NEPTUN feed
	// (key "м. київ"), delivered in data.oblasts, not data.raions.
	for _, o := range f.Data.Oblasts {
		if isKyivCityOblast(o.Key, o.Name) {
			return true
		}
	}
	for _, r := range f.Data.Raions {
		if isKyivCityRaion(r.Key, r.Oblast) {
			return true
		}
	}
	return false
}

// isKyivCityOblast reports whether an oblast-level alert entry targets the
// city of Kyiv (as opposed to Kyiv Oblast).
func isKyivCityOblast(key, name string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	n := strings.ToLower(strings.TrimSpace(name))
	switch k {
	case "м. київ", "м.київ", "м київ", "місто київ", "київ":
		return true
	}
	switch n {
	case "м. київ", "м.київ", "м київ", "місто київ", "київ":
		return true
	}
	return false
}

// isKyivCityRaion reports whether a raion belongs to the city of Kyiv
// (not Kyiv Oblast).
func isKyivCityRaion(key, oblast string) bool {
	o := strings.ToLower(strings.TrimSpace(oblast))
	switch o {
	case "місто київ", "м. київ", "м київ", "київ":
		return true
	}
	// Fallback: the raion key is one of the unique city raions and the oblast
	// field does not point to a different (oblast) region.
	if kyivCityRaions[strings.ToLower(strings.TrimSpace(key))] && !strings.Contains(o, "область") {
		return true
	}
	return false
}
