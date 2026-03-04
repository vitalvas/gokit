package xsemver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Version
		wantErr error
	}{
		// valid: basic versions
		{"basic", "1.2.3", Version{Major: 1, Minor: 2, Patch: 3}, nil},
		{"zeros", "0.0.0", Version{Major: 0, Minor: 0, Patch: 0}, nil},
		{"large numbers", "999.999.999", Version{Major: 999, Minor: 999, Patch: 999}, nil},

		// valid: v-prefix
		{"v prefix", "v1.2.3", Version{Major: 1, Minor: 2, Patch: 3}, nil},
		{"v prefix zeros", "v0.0.0", Version{Major: 0, Minor: 0, Patch: 0}, nil},

		// valid: pre-release
		{"pre-release alpha", "1.0.0-alpha", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha"}, nil},
		{"pre-release alpha.1", "1.0.0-alpha.1", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha.1"}, nil},
		{"pre-release beta.2", "1.0.0-beta.2", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "beta.2"}, nil},
		{"pre-release rc.1", "1.0.0-rc.1", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "rc.1"}, nil},
		{"pre-release numeric", "1.0.0-0", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "0"}, nil},
		{"pre-release with hyphens", "1.0.0-x-y-z", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "x-y-z"}, nil},

		// valid: build metadata
		{"build metadata", "1.0.0+build", Version{Major: 1, Minor: 0, Patch: 0, Build: "build"}, nil},
		{"build metadata with dots", "1.0.0+build.123", Version{Major: 1, Minor: 0, Patch: 0, Build: "build.123"}, nil},
		{"build leading zeros allowed", "1.0.0+001", Version{Major: 1, Minor: 0, Patch: 0, Build: "001"}, nil},

		// valid: full version
		{"full version", "1.0.0-alpha.1+build.123", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha.1", Build: "build.123"}, nil},
		{"v prefix full", "v2.1.0-beta+exp.sha.5114f85", Version{Major: 2, Minor: 1, Patch: 0, PreRelease: "beta", Build: "exp.sha.5114f85"}, nil},

		// valid: lenient partial versions
		{"two parts", "1.2", Version{Major: 1, Minor: 2, Patch: 0}, nil},
		{"one part", "1", Version{Major: 1, Minor: 0, Patch: 0}, nil},
		{"v prefix two parts", "v1.2", Version{Major: 1, Minor: 2, Patch: 0}, nil},
		{"v prefix one part", "v1", Version{Major: 1, Minor: 0, Patch: 0}, nil},
		{"zero one part", "0", Version{Major: 0, Minor: 0, Patch: 0}, nil},
		{"zero two parts", "0.0", Version{Major: 0, Minor: 0, Patch: 0}, nil},
		{"partial with prerelease", "1.2-alpha", Version{Major: 1, Minor: 2, Patch: 0, PreRelease: "alpha"}, nil},
		{"partial with build", "1.2+build", Version{Major: 1, Minor: 2, Patch: 0, Build: "build"}, nil},
		{"one part with prerelease", "1-rc.1", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "rc.1"}, nil},

		// valid: leading zeros coerced
		{"leading zero major", "01.0.0", Version{Major: 1, Minor: 0, Patch: 0}, nil},
		{"leading zero minor", "1.02.0", Version{Major: 1, Minor: 2, Patch: 0}, nil},
		{"leading zero patch", "1.0.03", Version{Major: 1, Minor: 0, Patch: 3}, nil},
		{"leading zeros all", "001.002.003", Version{Major: 1, Minor: 2, Patch: 3}, nil},

		// invalid: empty / missing parts
		{"empty string", "", Version{}, ErrInvalidVersion},
		{"only v", "v", Version{}, ErrInvalidVersion},
		{"four parts", "1.2.3.4", Version{}, ErrInvalidVersion},

		// invalid: non-numeric core
		{"alpha major", "a.0.0", Version{}, ErrInvalidVersion},
		{"alpha minor", "1.b.0", Version{}, ErrInvalidVersion},
		{"alpha patch", "1.0.c", Version{}, ErrInvalidVersion},
		{"negative major", "-1.0.0", Version{}, ErrInvalidVersion},

		// invalid: pre-release
		{"pre-release empty identifier", "1.0.0-", Version{}, ErrEmptyIdentifier},
		{"pre-release trailing dot", "1.0.0-alpha.", Version{}, ErrEmptyIdentifier},
		{"pre-release leading dot", "1.0.0-.alpha", Version{}, ErrEmptyIdentifier},
		{"pre-release leading zero numeric", "1.0.0-01", Version{}, ErrLeadingZero},
		{"pre-release invalid char", "1.0.0-al$pha", Version{}, ErrInvalidPreRelease},

		// invalid: build metadata
		{"build empty identifier", "1.0.0+", Version{}, ErrEmptyIdentifier},
		{"build trailing dot", "1.0.0+build.", Version{}, ErrEmptyIdentifier},
		{"build invalid char", "1.0.0+bu!ld", Version{}, ErrInvalidBuild},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMustParse(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		v := MustParse("1.2.3")
		assert.Equal(t, Version{Major: 1, Minor: 2, Patch: 3}, v)
	})

	t.Run("panics on invalid", func(t *testing.T) {
		assert.Panics(t, func() {
			MustParse("invalid")
		})
	})
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid basic", "1.2.3", true},
		{"valid with v", "v1.2.3", true},
		{"valid full", "1.0.0-alpha+build", true},
		{"valid partial two", "1.2", true},
		{"valid partial one", "1", true},
		{"valid leading zero", "01.0.0", true},
		{"invalid empty", "", false},
		{"invalid format", "not-a-version", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValid(tt.input))
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"basic", "1.2.3", "1.2.3"},
		{"strips v", "v1.2.3", "1.2.3"},
		{"pre-release", "1.0.0-alpha.1", "1.0.0-alpha.1"},
		{"build", "1.0.0+build.123", "1.0.0+build.123"},
		{"full", "1.0.0-beta+exp.sha", "1.0.0-beta+exp.sha"},
		{"zeros", "0.0.0", "0.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := MustParse(tt.input)
			assert.Equal(t, tt.want, v.String())
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		// major/minor/patch ordering
		{"major less", "1.0.0", "2.0.0", -1},
		{"major greater", "2.0.0", "1.0.0", 1},
		{"minor less", "1.1.0", "1.2.0", -1},
		{"minor greater", "1.2.0", "1.1.0", 1},
		{"patch less", "1.0.1", "1.0.2", -1},
		{"patch greater", "1.0.2", "1.0.1", 1},
		{"equal", "1.2.3", "1.2.3", 0},

		// pre-release precedence (semver spec examples)
		{"alpha < alpha.1", "1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"alpha.1 < alpha.beta", "1.0.0-alpha.1", "1.0.0-alpha.beta", -1},
		{"alpha.beta < beta", "1.0.0-alpha.beta", "1.0.0-beta", -1},
		{"beta < beta.2", "1.0.0-beta", "1.0.0-beta.2", -1},
		{"beta.2 < beta.11", "1.0.0-beta.2", "1.0.0-beta.11", -1},
		{"beta.11 < rc.1", "1.0.0-beta.11", "1.0.0-rc.1", -1},
		{"rc.1 < release", "1.0.0-rc.1", "1.0.0", -1},

		// pre-release: has pre-release < no pre-release
		{"pre-release < release", "1.0.0-alpha", "1.0.0", -1},
		{"release > pre-release", "1.0.0", "1.0.0-alpha", 1},

		// pre-release: numeric < alphanumeric
		{"numeric < alpha", "1.0.0-1", "1.0.0-alpha", -1},

		// build metadata ignored
		{"build ignored equal", "1.0.0+build1", "1.0.0+build2", 0},
		{"build ignored with prerelease", "1.0.0-alpha+build1", "1.0.0-alpha+build2", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := MustParse(tt.a)
			b := MustParse(tt.b)
			assert.Equal(t, tt.want, a.Compare(b))
		})
	}
}

func TestLessThan(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"less", "1.0.0", "2.0.0", true},
		{"equal", "1.0.0", "1.0.0", false},
		{"greater", "2.0.0", "1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := MustParse(tt.a)
			b := MustParse(tt.b)
			assert.Equal(t, tt.want, a.LessThan(b))
		})
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"equal", "1.0.0", "1.0.0", true},
		{"not equal", "1.0.0", "2.0.0", false},
		{"equal ignores build", "1.0.0+a", "1.0.0+b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := MustParse(tt.a)
			b := MustParse(tt.b)
			assert.Equal(t, tt.want, a.Equal(b))
		})
	}
}

func TestGreaterThan(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"greater", "2.0.0", "1.0.0", true},
		{"equal", "1.0.0", "1.0.0", false},
		{"less", "1.0.0", "2.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := MustParse(tt.a)
			b := MustParse(tt.b)
			assert.Equal(t, tt.want, a.GreaterThan(b))
		})
	}
}

func TestLessThanEqual(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"less", "1.0.0", "2.0.0", true},
		{"equal", "1.0.0", "1.0.0", true},
		{"greater", "2.0.0", "1.0.0", false},
		{"pre-release less", "1.0.0-alpha", "1.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := MustParse(tt.a)
			b := MustParse(tt.b)
			assert.Equal(t, tt.want, a.LessThanEqual(b))
		})
	}
}

func TestGreaterThanEqual(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"greater", "2.0.0", "1.0.0", true},
		{"equal", "1.0.0", "1.0.0", true},
		{"less", "1.0.0", "2.0.0", false},
		{"release greater than pre-release", "1.0.0", "1.0.0-alpha", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := MustParse(tt.a)
			b := MustParse(tt.b)
			assert.Equal(t, tt.want, a.GreaterThanEqual(b))
		})
	}
}

func TestIncMajor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Version
	}{
		{"basic", "1.2.3", Version{Major: 2}},
		{"zero", "0.0.0", Version{Major: 1}},
		{"with prerelease", "1.2.3-alpha", Version{Major: 2}},
		{"with build", "1.2.3+build", Version{Major: 2}},
		{"large", "99.88.77", Version{Major: 100}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := MustParse(tt.input)
			assert.Equal(t, tt.want, v.IncMajor())
		})
	}
}

func TestIncMinor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Version
	}{
		{"basic", "1.2.3", Version{Major: 1, Minor: 3}},
		{"zero", "0.0.0", Version{Major: 0, Minor: 1}},
		{"with prerelease", "1.2.3-beta", Version{Major: 1, Minor: 3}},
		{"with build", "1.2.3+build", Version{Major: 1, Minor: 3}},
		{"preserves major", "5.9.1", Version{Major: 5, Minor: 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := MustParse(tt.input)
			assert.Equal(t, tt.want, v.IncMinor())
		})
	}
}

func TestIncPatch(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Version
	}{
		{"basic", "1.2.3", Version{Major: 1, Minor: 2, Patch: 4}},
		{"zero", "0.0.0", Version{Major: 0, Minor: 0, Patch: 1}},
		{"with prerelease", "1.2.3-rc.1", Version{Major: 1, Minor: 2, Patch: 4}},
		{"with build", "1.2.3+build", Version{Major: 1, Minor: 2, Patch: 4}},
		{"preserves major minor", "5.9.1", Version{Major: 5, Minor: 9, Patch: 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := MustParse(tt.input)
			assert.Equal(t, tt.want, v.IncPatch())
		})
	}
}

func TestDiff(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want string
	}{
		{"equal", "1.2.3", "1.2.3", ""},
		{"major diff", "1.0.0", "2.0.0", "major"},
		{"minor diff", "1.1.0", "1.2.0", "minor"},
		{"patch diff", "1.0.1", "1.0.2", "patch"},
		{"prerelease diff", "1.0.0-alpha", "1.0.0-beta", "prerelease"},
		{"prerelease vs release", "1.0.0-alpha", "1.0.0", "prerelease"},
		{"major takes priority", "1.2.3", "2.3.4", "major"},
		{"minor takes priority over patch", "1.1.1", "1.2.2", "minor"},
		{"build ignored equal", "1.0.0+a", "1.0.0+b", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := MustParse(tt.a)
			b := MustParse(tt.b)
			assert.Equal(t, tt.want, Diff(a, b))
		})
	}
}

func TestVersionIsValid(t *testing.T) {
	tests := []struct {
		name string
		v    Version
		want bool
	}{
		{"valid basic", Version{Major: 1, Minor: 0, Patch: 0}, true},
		{"valid with pre-release", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha.1"}, true},
		{"valid with build", Version{Major: 1, Minor: 0, Patch: 0, Build: "build.123"}, true},
		{"invalid pre-release chars", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "al$pha"}, false},
		{"invalid pre-release leading zero", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "01"}, false},
		{"invalid build chars", Version{Major: 1, Minor: 0, Patch: 0, Build: "bu!ld"}, false},
		{"invalid empty pre-release id", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha."}, false},
		{"invalid empty build id", Version{Major: 1, Minor: 0, Patch: 0, Build: "build."}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.v.IsValid())
		})
	}
}

func TestSort(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			"mixed versions",
			[]string{"2.0.0", "1.0.0", "1.1.0", "1.0.1", "0.1.0"},
			[]string{"0.1.0", "1.0.0", "1.0.1", "1.1.0", "2.0.0"},
		},
		{
			"with pre-release",
			[]string{"1.0.0", "1.0.0-alpha", "1.0.0-beta", "1.0.0-alpha.1"},
			[]string{"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-beta", "1.0.0"},
		},
		{
			"already sorted",
			[]string{"1.0.0", "2.0.0", "3.0.0"},
			[]string{"1.0.0", "2.0.0", "3.0.0"},
		},
		{
			"single element",
			[]string{"1.0.0"},
			[]string{"1.0.0"},
		},
		{
			"empty slice",
			[]string{},
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versions := make([]Version, len(tt.in))
			for i, s := range tt.in {
				versions[i] = MustParse(s)
			}

			Sort(versions)

			got := make([]string, len(versions))
			for i, v := range versions {
				got[i] = v.String()
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func BenchmarkParse(b *testing.B) {
	benchmarks := []struct {
		name  string
		input string
	}{
		{"basic", "1.2.3"},
		{"with_prerelease", "1.0.0-alpha.1"},
		{"with_build", "1.0.0+build.123"},
		{"full", "1.0.0-alpha.1+build.123"},
		{"v_prefix", "v1.2.3"},
		{"partial_two", "1.2"},
		{"partial_one", "1"},
		{"leading_zeros", "01.02.03"},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_, _ = Parse(bb.input)
			}
		})
	}
}

func BenchmarkCompare(b *testing.B) {
	benchmarks := []struct {
		name string
		a    string
		b    string
	}{
		{"equal", "1.2.3", "1.2.3"},
		{"major_diff", "1.0.0", "2.0.0"},
		{"prerelease", "1.0.0-alpha.1", "1.0.0-alpha.2"},
		{"prerelease_vs_release", "1.0.0-alpha", "1.0.0"},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			a := MustParse(bb.a)
			bv := MustParse(bb.b)

			b.ReportAllocs()

			for b.Loop() {
				a.Compare(bv)
			}
		})
	}
}

func BenchmarkSort(b *testing.B) {
	input := []string{
		"2.0.0", "1.0.0-alpha", "1.0.0", "3.0.0", "1.0.0-beta",
		"1.1.0", "0.1.0", "1.0.0-alpha.1", "1.0.1", "0.0.1",
	}

	base := make([]Version, len(input))
	for i, s := range input {
		base[i] = MustParse(s)
	}

	b.ReportAllocs()

	for b.Loop() {
		versions := make([]Version, len(base))
		copy(versions, base)
		Sort(versions)
	}
}

func BenchmarkString(b *testing.B) {
	benchmarks := []struct {
		name string
		v    Version
	}{
		{"basic", Version{Major: 1, Minor: 2, Patch: 3}},
		{"prerelease", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha.1"}},
		{"full", Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha.1", Build: "build.123"}},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_ = bb.v.String()
			}
		})
	}
}

func BenchmarkInc(b *testing.B) {
	v := MustParse("1.2.3-alpha+build")

	b.Run("major", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			v.IncMajor()
		}
	})

	b.Run("minor", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			v.IncMinor()
		}
	})

	b.Run("patch", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			v.IncPatch()
		}
	})
}

func BenchmarkDiff(b *testing.B) {
	benchmarks := []struct {
		name string
		a    string
		b    string
	}{
		{"equal", "1.2.3", "1.2.3"},
		{"major", "1.0.0", "2.0.0"},
		{"prerelease", "1.0.0-alpha", "1.0.0-beta"},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			a := MustParse(bb.a)
			bv := MustParse(bb.b)

			b.ReportAllocs()

			for b.Loop() {
				Diff(a, bv)
			}
		})
	}
}

func FuzzParse(f *testing.F) {
	f.Add("1.2.3")
	f.Add("0.0.0")
	f.Add("v1.2.3")
	f.Add("1.0.0-alpha")
	f.Add("1.0.0-alpha.1")
	f.Add("1.0.0+build")
	f.Add("1.0.0-alpha.1+build.123")
	f.Add("")
	f.Add("invalid")
	f.Add("1.2")
	f.Add("01.0.0")
	f.Add("1.0.0-")
	f.Add("1.0.0+")
	f.Add("1.0.0-01")
	f.Add("v")
	f.Add("1.0.0-al$pha")

	f.Fuzz(func(t *testing.T, input string) {
		v, err := Parse(input)
		if err != nil {
			return
		}

		// Round-trip: Parse -> String -> Parse must produce equal version.
		s := v.String()

		v2, err := Parse(s)
		require.NoError(t, err, "re-parse of %q failed", s)
		assert.Equal(t, 0, v.Compare(v2), "round-trip mismatch for %q -> %q", input, s)
	})
}
