package xstrings

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		str     string
		want    bool
		wantErr error
	}{
		// exact match
		{"exact match", "hello", "hello", true, nil},
		{"exact mismatch", "hello", "world", false, nil},
		{"empty pattern empty string", "", "", true, nil},
		{"empty pattern non-empty string", "", "hello", false, nil},
		{"non-empty pattern empty string", "hello", "", false, nil},

		// star wildcard
		{"star matches everything", "*", "anything", true, nil},
		{"star matches empty", "*", "", true, nil},
		{"star prefix", "*.txt", "file.txt", true, nil},
		{"star suffix", "hello*", "hello world", true, nil},
		{"star middle", "he*lo", "hello", true, nil},
		{"star multiple chars", "he*lo", "heyolo", true, nil},
		{"star no match", "he*lo", "hero", false, nil},
		{"multiple stars", "*foo*bar*", "XXfooYYbarZZ", true, nil},
		{"multiple stars no match", "*foo*bar*", "XXfooYY", false, nil},

		// double star (equivalent to single star)
		{"double star matches everything", "**", "anything/here", true, nil},
		{"double star matches empty", "**", "", true, nil},
		{"double star prefix", "**/*.txt", "some/path/file.txt", true, nil},

		// star matches slash (generic string mode)
		{"star matches slash", "*", "a/b/c", true, nil},
		{"star in segment matches slash", "a*c", "a/b/c", true, nil},

		// question mark
		{"question matches single char", "?", "a", true, nil},
		{"question no match empty", "?", "", false, nil},
		{"question no match two chars", "?", "ab", false, nil},
		{"question in pattern", "h?llo", "hello", true, nil},
		{"question mismatch", "h?llo", "hllo", false, nil},
		{"multiple questions", "???", "abc", true, nil},
		{"multiple questions short", "???", "ab", false, nil},

		// character classes
		{"char class match", "[abc]", "a", true, nil},
		{"char class match second", "[abc]", "b", true, nil},
		{"char class no match", "[abc]", "d", false, nil},
		{"char class in word", "h[ae]llo", "hello", true, nil},
		{"char class in word alt", "h[ae]llo", "hallo", true, nil},
		{"char class in word no match", "h[ae]llo", "hullo", false, nil},

		// character ranges
		{"range match", "[a-z]", "m", true, nil},
		{"range no match", "[a-z]", "M", false, nil},
		{"range digit", "[0-9]", "5", true, nil},
		{"range digit no match", "[0-9]", "a", false, nil},

		// negated character classes
		{"negated class match", "[!abc]", "d", true, nil},
		{"negated class no match", "[!abc]", "a", false, nil},
		{"negated caret match", "[^abc]", "d", true, nil},
		{"negated caret no match", "[^abc]", "a", false, nil},
		{"negated range match", "[!a-z]", "A", true, nil},
		{"negated range no match", "[!a-z]", "m", false, nil},

		// escape sequences
		{"escaped star", "hello\\*", "hello*", true, nil},
		{"escaped star no match", "hello\\*", "helloX", false, nil},
		{"escaped question", "hello\\?", "hello?", true, nil},
		{"escaped bracket", "hello\\[", "hello[", true, nil},
		{"escaped backslash", "hello\\\\world", "hello\\world", true, nil},

		// HTTP request matching use cases
		{"http any path", "/api/v1/*", "/api/v1/users", true, nil},
		{"http nested path", "/api/v1/*", "/api/v1/users/123/profile", true, nil},
		{"http any version", "/api/*/users", "/api/v1/users", true, nil},
		{"http any version nested", "/api/*/users", "/api/v2/users", true, nil},
		{"http method pattern", "GET /api/*", "GET /api/v1/users", true, nil},
		{"http no match", "/api/v1/users", "/api/v2/users", false, nil},
		{"http wildcard middle", "/api/*/users/*/edit", "/api/v1/users/42/edit", true, nil},

		// unicode
		{"unicode match", "hel*", "hello", true, nil},
		{"unicode question", "h?llo", "h\u00e9llo", true, nil},
		{"unicode class", "[\u00e0-\u00ff]", "\u00e9", true, nil},

		// error cases
		{"unclosed bracket", "[abc", "", false, ErrBadGlobPattern},
		{"trailing backslash", "hello\\", "hello", false, ErrBadGlobPattern},
		{"bad range", "[z-a]", "m", false, ErrBadGlobPattern},
		{"empty bracket", "[]", "", false, ErrBadGlobPattern},
		{"bracket only open", "[", "", false, ErrBadGlobPattern},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GlobMatch(tt.pattern, tt.str)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func BenchmarkGlobMatch(b *testing.B) {
	benchmarks := []struct {
		name    string
		pattern string
		str     string
	}{
		{"exact", "hello", "hello"},
		{"star_suffix", "/api/v1/*", "/api/v1/users/123/profile"},
		{"star_middle", "/api/*/users/*/edit", "/api/v1/users/42/edit"},
		{"multi_star", "*foo*bar*baz*", "XXfooYYbarZZbazQQ"},
		{"question", "h?l?o", "hello"},
		{"char_class", "h[aeiou]llo", "hello"},
		{"char_range", "[a-z][0-9][a-z]", "a5z"},
		{"no_match", "/api/v1/*", "/other/path/here"},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_, _ = GlobMatch(bb.pattern, bb.str)
			}
		})
	}
}

func FuzzGlobMatch(f *testing.F) {
	f.Add("*", "hello")
	f.Add("**", "a/b/c")
	f.Add("h?llo", "hello")
	f.Add("???", "abc")
	f.Add("[abc]", "a")
	f.Add("[a-z]", "m")
	f.Add("[!abc]", "d")
	f.Add("[^0-9]", "x")
	f.Add("/api/*/users", "/api/v1/users")
	f.Add("*foo*bar*", "XXfooYYbarZZ")
	f.Add("\\*", "*")
	f.Add("\\?", "?")
	f.Add("\\[", "[")
	f.Add("hello\\\\world", "hello\\world")
	f.Add("", "")
	f.Add("", "nonempty")
	f.Add("nonempty", "")

	f.Fuzz(func(_ *testing.T, pattern, str string) {
		_, _ = GlobMatch(pattern, str)
	})
}
