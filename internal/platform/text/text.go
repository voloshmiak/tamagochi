package text

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func Clean(s string) (string, bool) {
	if !utf8.ValidString(s) {
		return "", false
	}
	s = strings.TrimSpace(s)
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return "", false
		}
	}
	return s, true
}
