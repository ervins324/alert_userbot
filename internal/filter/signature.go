package filter

import (
	"regexp"
	"strings"
)

// signatureRE matches the subscription footer that the source channel
// appends to every post, e.g.:
//
//	Підписатись 👉 🚀ППО | РАДАР (https://t.me/mon1tor_ua)
//	@mon1tor_ua
var signatureRE = regexp.MustCompile(`(?ms)^\s*Підписатись.*?@mon1tor_ua\s*$`)

// HasSignature reports whether the text contains the channel footer.
func HasSignature(text string) bool {
	return signatureRE.MatchString(strings.TrimRight(text, "\r\n\t "))
}

// RemoveSignature strips the channel footer from the text. It returns the
// cleaned text and whether the signature was found and removed.
func RemoveSignature(text string) (string, bool) {
	cleaned := signatureRE.ReplaceAllString(strings.TrimRight(text, "\r\n\t "), "")
	cleaned = strings.TrimRight(cleaned, "\r\n\t ")
	if cleaned != text {
		return cleaned, true
	}
	return text, false
}
