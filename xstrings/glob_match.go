package xstrings

import (
	"errors"
	"unicode/utf8"
)

// ErrBadGlobPattern indicates a malformed glob pattern.
var ErrBadGlobPattern = errors.New("syntax error in glob pattern")

// GlobMatch reports whether the string str matches the glob pattern.
//
// Pattern syntax:
//   - '*' matches any sequence of zero or more characters
//   - '**' is equivalent to '*'
//   - '?' matches any single character
//   - '[abc]' matches any single character in the set
//   - '[a-z]' matches any single character in the range
//   - '[!abc]' or '[^abc]' matches any single character NOT in the set
//   - '\\' escapes the next character (literal match)
//
// Unlike filepath.Match, no character is treated as a path separator,
// so '*' matches any character including '/'.
func GlobMatch(pattern, str string) (bool, error) {
	px, sx := 0, 0
	starPx, starSx := -1, -1

	for sx < len(str) || px < len(pattern) {
		if px < len(pattern) {
			switch pattern[px] {
			case '*':
				for px < len(pattern) && pattern[px] == '*' {
					px++
				}

				starPx = px
				starSx = sx

				continue

			case '?':
				if sx < len(str) {
					_, sSize := utf8.DecodeRuneInString(str[sx:])
					px++
					sx += sSize

					continue
				}

			case '[':
				// Always validate the character class, even if string is exhausted.
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
			_, sSize := utf8.DecodeRuneInString(str[starSx:])
			starSx += sSize
			sx = starSx
			px = starPx

			continue
		}

		return false, nil
	}

	return true, nil
}

// globMatchCharClass matches a character class pattern starting at pattern[px]
// (which should be '['). Returns whether the rune matches, the index after the
// closing ']', and any error.
func globMatchCharClass(pattern string, px int, r rune) (bool, int, error) {
	px++ // skip '['

	if px >= len(pattern) {
		return false, 0, ErrBadGlobPattern
	}

	negated := false

	if pattern[px] == '!' || pattern[px] == '^' {
		negated = true
		px++
	}

	matched := false
	nrange := 0

	for px < len(pattern) {
		if pattern[px] == ']' && nrange > 0 {
			px++

			if negated {
				return !matched, px, nil
			}

			return matched, px, nil
		}

		lo, size, err := globDecodePatternRune(pattern, px)
		if err != nil {
			return false, 0, err
		}

		px += size
		nrange++

		if px < len(pattern) && pattern[px] == '-' && px+1 < len(pattern) && pattern[px+1] != ']' {
			px++ // skip '-'

			hi, hiSize, err := globDecodePatternRune(pattern, px)
			if err != nil {
				return false, 0, err
			}

			px += hiSize

			if lo > hi {
				return false, 0, ErrBadGlobPattern
			}

			if r >= lo && r <= hi {
				matched = true
			}
		} else if r == lo {
			matched = true
		}
	}

	return false, 0, ErrBadGlobPattern
}

func globDecodePatternRune(pattern string, px int) (rune, int, error) {
	if px >= len(pattern) {
		return 0, 0, ErrBadGlobPattern
	}

	if pattern[px] == '\\' {
		px++

		if px >= len(pattern) {
			return 0, 0, ErrBadGlobPattern
		}

		r, size := utf8.DecodeRuneInString(pattern[px:])

		return r, size + 1, nil
	}

	r, size := utf8.DecodeRuneInString(pattern[px:])

	return r, size, nil
}
