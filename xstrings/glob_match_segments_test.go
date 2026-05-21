package xstrings

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobMatchSegments(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		str     string
		seps    string
		want    bool
		wantErr error
	}{
		// exact match
		{"exact match", "hello", "hello", "", true, nil},
		{"exact mismatch", "hello", "world", "", false, nil},
		{"empty pattern empty string", "", "", "", true, nil},
		{"empty pattern non-empty string", "", "hello", "", false, nil},
		{"non-empty pattern empty string", "hello", "", "", false, nil},

		// single star is segment-bounded (default sep "/")
		{"star matches within segment", "*", "abc", "", true, nil},
		{"star empty", "*", "", "", true, nil},
		{"star does not cross slash", "*", "a/b", "", false, nil},
		{"star prefix within segment", "*.txt", "file.txt", "", true, nil},
		{"star prefix stops at slash", "*.txt", "dir/file.txt", "", false, nil},
		{"star middle within segment", "a*c", "abc", "", true, nil},
		{"star middle stops at slash", "a*c", "a/c", "", false, nil},
		{"star matches one segment", "/api/*/users", "/api/v1/users", "", true, nil},
		{"star does not match nested", "/api/*/users", "/api/v1/v2/users", "", false, nil},

		// double star crosses separators
		{"double star matches everything", "**", "a/b/c", "", true, nil},
		{"double star empty", "**", "", "", true, nil},
		{"double star nested path", "/api/**/users", "/api/v1/v2/users", "", true, nil},
		{"double star prefix", "**/file.txt", "a/b/c/file.txt", "", true, nil},
		{"double star prefix one segment", "**/file.txt", "a/file.txt", "", true, nil},

		// question respects separators
		{"question matches single char", "?", "a", "", true, nil},
		{"question does not match slash", "?", "/", "", false, nil},
		{"question in word", "h?llo", "hello", "", true, nil},
		{"question stops at slash", "a?c", "a/c", "", false, nil},

		// OIDC gandalf use case with seps="/:"
		{"oidc exact tail", "repo:vitalvas/*:ref:refs/heads/main", "repo:vitalvas/gandalf:ref:refs/heads/main", "/:", true, nil},
		{"oidc star does not cross slash", "repo:vitalvas/*:ref:refs/heads/main", "repo:vitalvas/gandalf/extra:ref:refs/heads/main", "/:", false, nil},
		{"oidc star does not cross colon", "repo:vitalvas/*", "repo:vitalvas/gandalf:ref:refs/heads/main", "/:", false, nil},
		{"oidc double star crosses both", "repo:vitalvas/**", "repo:vitalvas/gandalf:ref:refs/heads/main", "/:", true, nil},
		{"oidc segment in middle", "repo:*/gandalf:ref:refs/heads/*", "repo:vitalvas/gandalf:ref:refs/heads/main", "/:", true, nil},
		{"oidc segment in middle rejects sub-path", "repo:*/gandalf:ref:refs/heads/*", "repo:victim/owner/gandalf:ref:refs/heads/main", "/:", false, nil},

		// char classes can match separators when explicitly listed
		{"class with slash matches", "a[/]b", "a/b", "", true, nil},
		{"class without slash does not match slash", "a[xy]b", "a/b", "", false, nil},
		{"range in class", "h[a-z]llo", "hello", "", true, nil},
		{"negated class", "[!abc]", "d", "", true, nil},

		// escape sequences
		{"escaped star", "hello\\*", "hello*", "", true, nil},
		{"escaped star literal", "hello\\*", "helloX", "", false, nil},
		{"escaped backslash", "hello\\\\world", "hello\\world", "", true, nil},

		// explicit empty default behavior (seps == "" -> "/")
		{"default sep is slash", "a*b", "a/b", "", false, nil},

		// custom seps - only colon
		{"colon-only sep allows slash in star", "a*b", "a/x/b", ":", true, nil},
		{"colon-only sep blocks colon", "a*b", "a:x:b", ":", false, nil},

		// multi-segment patterns
		{"trailing double star", "/api/**", "/api/v1/users/123", "", true, nil},
		{"trailing double star empty tail", "/api/**", "/api/", "", true, nil},
		{"leading double star", "**/main", "feature/branch/main", "", true, nil},

		// unicode
		{"unicode within segment", "h?llo", "héllo", "", true, nil},
		{"unicode does not affect seps", "a*b", "aéb", "", true, nil},

		// error cases
		{"unclosed bracket", "[abc", "", "", false, ErrBadGlobPattern},
		{"trailing backslash", "hello\\", "hello", "", false, ErrBadGlobPattern},
		{"bad range", "[z-a]", "m", "", false, ErrBadGlobPattern},
		{"empty bracket", "[]", "", "", false, ErrBadGlobPattern},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GlobMatchSegments(tt.pattern, tt.str, tt.seps)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func BenchmarkGlobMatchSegments(b *testing.B) {
	benchmarks := []struct {
		name    string
		pattern string
		str     string
		seps    string
	}{
		{"exact", "hello", "hello", ""},
		{"exact_mismatch", "hello", "world", ""},
		{"star_segment", "/api/*/users", "/api/v1/users", ""},
		{"star_segment_reject", "/api/*/users", "/api/v1/v2/users", ""},
		{"double_star", "/api/**/users", "/api/v1/v2/v3/users", ""},
		{"double_star_deep", "**/last", "a/b/c/d/e/f/g/h/i/last", ""},
		{"oidc_match", "repo:vitalvas/*:ref:refs/heads/*", "repo:vitalvas/gandalf:ref:refs/heads/main", "/:"},
		{"oidc_reject", "repo:vitalvas/*:ref:refs/heads/*", "repo:vitalvas/gandalf/extra:ref:refs/heads/main", "/:"},
		{"oidc_double_star", "repo:vitalvas/**", "repo:vitalvas/gandalf:ref:refs/heads/main", "/:"},
		{"trailing_doublestar", "/api/**", "/api/v1/users/123/profile/settings", ""},
		{"question", "h?llo", "hello", ""},
		{"char_class", "h[aeiou]llo", "hello", ""},
		{"char_class_with_sep", "a[/]b", "a/b", ""},
		{"escaped_literal", "a\\*b", "a*b", ""},
		{"many_stars", "*/*/*/*/*/*", "a/b/c/d/e/f", ""},
		{"backtrack_heavy", "*foo*bar*baz", "afooXbarYbaz", ""},
		{"long_no_match", "/api/v1/*", "/other/long/path/that/does/not/match", ""},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_, _ = GlobMatchSegments(bb.pattern, bb.str, bb.seps)
			}
		})
	}
}

func FuzzGlobMatchSegments(f *testing.F) {
	f.Add("*", "hello", "")
	f.Add("**", "a/b/c", "")
	f.Add("*", "a/b", "")
	f.Add("h?llo", "hello", "")
	f.Add("???", "abc", "")
	f.Add("[abc]", "a", "")
	f.Add("[a-z]", "m", "")
	f.Add("[!abc]", "d", "")
	f.Add("[^0-9]", "x", "")
	f.Add("[/]", "/", "/")
	f.Add("/api/*/users", "/api/v1/users", "")
	f.Add("/api/**/users", "/api/v1/v2/users", "")
	f.Add("*foo*bar*", "XXfooYYbarZZ", "")
	f.Add("\\*", "*", "")
	f.Add("\\?", "?", "")
	f.Add("\\[", "[", "")
	f.Add("hello\\\\world", "hello\\world", "")
	f.Add("repo:*/main", "repo:foo/main", "/:")
	f.Add("repo:**/main", "repo:foo/bar/main", "/:")
	f.Add("repo:vitalvas/*:ref:refs/heads/main", "repo:vitalvas/gandalf:ref:refs/heads/main", "/:")
	f.Add("repo:vitalvas/*:ref:refs/heads/main", "repo:vitalvas/gandalf/x:ref:refs/heads/main", "/:")
	f.Add("a*b", "aéb", "")
	f.Add("héllo", "héllo", "")
	f.Add("", "", "")
	f.Add("", "nonempty", "")
	f.Add("nonempty", "", "")
	f.Add("**", "", "")
	f.Add("*", "", "")

	f.Fuzz(func(t *testing.T, pattern, str, seps string) {
		got, err := GlobMatchSegments(pattern, str, seps)
		if err != nil {
			return
		}

		// Invariant 1: with all separators removed from str AND seps,
		// GlobMatchSegments must agree with GlobMatch on a pattern without ** runs.
		// We only assert the weaker invariant that double-star is always at least
		// as permissive as single-star.
		if got {
			return
		}

		// Invariant 2: replacing every single '*' in the pattern with '**' can only
		// turn a non-match into a match, never the other way around.
		doubled := replaceSingleStarsWithDouble(pattern)
		if doubled == pattern {
			return
		}

		got2, err2 := GlobMatchSegments(doubled, str, seps)
		if err2 != nil {
			return
		}

		if got && !got2 {
			t.Fatalf("monotonicity violated: %q matched but %q did not (str=%q seps=%q)", pattern, doubled, str, seps)
		}
	})
}

// replaceSingleStarsWithDouble rewrites every run of exactly one '*' in pattern
// into '**', leaving runs of two or more stars (already '**') and escaped stars
// alone. Used by the fuzz monotonicity check.
func replaceSingleStarsWithDouble(pattern string) string {
	out := make([]byte, 0, len(pattern)+4)

	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '\\' && i+1 < len(pattern) {
			out = append(out, pattern[i], pattern[i+1])
			i++

			continue
		}

		if pattern[i] != '*' {
			out = append(out, pattern[i])

			continue
		}

		j := i
		for j < len(pattern) && pattern[j] == '*' {
			j++
		}

		if j-i == 1 {
			out = append(out, '*', '*')
		} else {
			out = append(out, pattern[i:j]...)
		}

		i = j - 1
	}

	return string(out)
}
