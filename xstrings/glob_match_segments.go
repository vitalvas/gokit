package xstrings

import (
	"strings"
	"unicode/utf8"
)

// GlobMatchSegments reports whether str matches pattern using segment-aware
// semantics suitable for matching structured identifiers such as file paths
// or OIDC subject claims.
//
// Pattern syntax:
//   - '*' matches any sequence of zero or more characters that are NOT in seps
//   - '**' matches any sequence of zero or more characters (including seps)
//   - '?' matches any single character that is NOT in seps
//   - '[abc]' matches any single character in the set (may include seps)
//   - '[a-z]' matches any single character in the range
//   - '[!abc]' or '[^abc]' matches any single character NOT in the set
//   - '\\' escapes the next character (literal match)
//
// A run of one '*' is segment-bounded; a run of two or more '*' is treated as
// '**' and crosses separators.
//
// When seps is empty, it defaults to "/". Pass a string of separator runes such
// as "/:" to treat both '/' and ':' as segment boundaries.
//
// Example:
//
//	GlobMatchSegments("repo:*/main", "repo:foo/main", "")          // true
//	GlobMatchSegments("repo:*/main", "repo:foo/bar/main", "")      // false
//	GlobMatchSegments("repo:**/main", "repo:foo/bar/main", "")     // true
//	GlobMatchSegments("repo:*:ref", "repo:a/b:ref", "/:")          // false
func GlobMatchSegments(pattern, str, seps string) (bool, error) {
	if seps == "" {
		seps = "/"
	}

	px, sx := 0, 0
	starPx, starSx := -1, -1
	starDouble := false

	for sx < len(str) || px < len(pattern) {
		if px < len(pattern) {
			switch pattern[px] {
			case '*':
				stars := 0
				for px < len(pattern) && pattern[px] == '*' {
					px++
					stars++
				}

				starPx = px
				starSx = sx
				starDouble = stars >= 2

				continue

			case '?':
				if sx < len(str) {
					sr, sSize := utf8.DecodeRuneInString(str[sx:])
					if !strings.ContainsRune(seps, sr) {
						px++
						sx += sSize

						continue
					}
				}

			case '[':
				var sr rune
				var sSize int

				if sx < len(str) {
					sr, sSize = utf8.DecodeRuneInString(str[sx:])
				}

				matched, newPx, err := globMatchCharClass(pattern, px, sr)
				if err != nil {
					return false, err
				}

				if sx < len(str) && matched {
					px = newPx
					sx += sSize

					continue
				}

			case '\\':
				px++

				if px >= len(pattern) {
					return false, ErrBadGlobPattern
				}

				ec, eSize := utf8.DecodeRuneInString(pattern[px:])

				if sx < len(str) {
					sr, sSize := utf8.DecodeRuneInString(str[sx:])
					if ec == sr {
						px += eSize
						sx += sSize

						continue
					}
				}

			default:
				if sx < len(str) {
					pc, pSize := utf8.DecodeRuneInString(pattern[px:])
					sr, sSize := utf8.DecodeRuneInString(str[sx:])

					if pc == sr {
						px += pSize
						sx += sSize

						continue
					}
				}
			}
		}

		if starPx >= 0 && starSx < len(str) {
			sr, sSize := utf8.DecodeRuneInString(str[starSx:])
			if !starDouble && strings.ContainsRune(seps, sr) {
				return false, nil
			}

			starSx += sSize
			sx = starSx
			px = starPx

			continue
		}

		return false, nil
	}

	return true, nil
}
