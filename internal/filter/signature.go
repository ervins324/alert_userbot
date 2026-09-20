package filter

import (
	"regexp"
	"strings"
)

// subHeaderRE matches subscription / source call-to-action lines (case-insensitive).
// e.g.:
//   Підписатись 👉 🚀ППО | РАДАР (https://t.me/mon1tor_ua)
//   Підписатися на канал
//   Підпишись 👉 @kyiv_monitor1
//   Джерело: t.me/mon1tor_ua
var subHeaderRE = regexp.MustCompile(`(?mi)^\s*(?:[👉🚀🔔📡📍➡️▶️🔗\s]*)(?:підписатис[яь]|підпишис[яь]|підпишіться|підписка|подписат[ьс]я|подпишись|подписка|наш канал|наш чат|джерело|источник)[^\n]*$`)

// trailingHandleRE matches a line that consists solely of a channel handle,
// optionally preceded by an emoji or pointer: e.g. "@kyiv_monitor1", "👉 @mon1tor_ua", "🚀 @channel".
var trailingHandleRE = regexp.MustCompile(`(?i)^\s*(?:[👉🚀🔔📡📍➡️▶️🔗\s]*)@[\w_]{3,}\s*$`)

// trailingLinkRE matches a line that is solely a Telegram link:
// e.g. "t.me/kyiv_monitor1", "https://t.me/mon1tor_ua", "telegram.me/mon1tor_ua".
var trailingLinkRE = regexp.MustCompile(`(?i)^\s*(?:[👉🚀🔔📡📍➡️▶️🔗\s]*)(?:\(?)?(?:https?://)?(?:t\.me|telegram\.me)/[\w_+-]+(?:\)?)\s*$`)

// isSignatureLine reports whether a single line is a channel signature/footer line.
func isSignatureLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	if subHeaderRE.MatchString(trimmed) {
		return true
	}
	if trailingHandleRE.MatchString(trimmed) {
		return true
	}
	if trailingLinkRE.MatchString(trimmed) {
		return true
	}
	return false
}

// HasSignature reports whether the text contains any channel footer or handle signature.
func HasSignature(text string) bool {
	lines := strings.Split(strings.TrimRight(text, "\r\n\t "), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimRight(lines[i], "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		return isSignatureLine(line)
	}
	return false
}

// RemoveSignature strips channel footers, subscription prompts, trailing handles (@channel),
// and telegram links from the bottom of the post.
// It returns the cleaned text and whether any signature was found and removed.
func RemoveSignature(text string) (string, bool) {
	trimmed := strings.TrimRight(text, "\r\n\t ")
	if trimmed == "" {
		return text, false
	}

	lines := strings.Split(trimmed, "\n")
	removedAny := false

	// Scan from the bottom of the message upwards, stripping trailing signature lines
	cutoff := len(lines)
	for cutoff > 0 {
		line := strings.TrimRight(lines[cutoff-1], "\r")
		if isSignatureLine(line) {
			removedAny = true
			cutoff--
		} else {
			break
		}
	}

	if !removedAny {
		return text, false
	}

	cleaned := strings.TrimRight(strings.Join(lines[:cutoff], "\n"), "\r\n\t ")
	return cleaned, true
}
