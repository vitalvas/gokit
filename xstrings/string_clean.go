package xstrings

import (
	"strings"
	"unicode"
)

// clean up the non-ASCII characters
func StringClean(input string) string {
	return strings.TrimFunc(input, func(r rune) bool {
		return !unicode.IsGraphic(r)
	})
}
