package xstrings

import (
	"strings"
	"unicode"
)

// StringClean removes non-graphic characters from the input string.
//
//yake:skip-test
func StringClean(input string) string {
	return strings.TrimFunc(input, func(r rune) bool {
		return !unicode.IsGraphic(r)
	})
}
