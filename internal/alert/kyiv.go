package alert

import (
	"encoding/json"
	"strings"
)

// alertsFrame mirrors the NEPTUN "alerts" WebSocket frame structure.
type alertsFrame struct {
	Type string `json:"type"`
	Data struct {
		Raions []struct {
			Key    string `json:"key"`
			Name   string `json:"name"`
			Oblast string `json:"oblast"`
		} `json:"raions"`
	} `json:"data"`
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
	for _, r := range f.Data.Raions {
		if isKyivCityRaion(r.Key, r.Oblast) {
			return true
		}
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
