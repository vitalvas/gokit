package xsemver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConstraint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// valid: basic operators
		{"equal", "=1.2.3", false},
		{"double equal", "==1.2.3", false},
		{"not equal", "!=1.2.3", false},
		{"greater than", ">1.2.3", false},
		{"greater than equal", ">=1.2.3", false},
		{"less than", "<1.2.3", false},
		{"less than equal", "<=1.2.3", false},

		// valid: tilde/caret
		{"tilde", "~1.2.3", false},
		{"tilde minor", "~1.2", false},
		{"tilde major", "~1", false},
		{"tilde pessimistic", "~>1.2.3", false},
		{"caret", "^1.2.3", false},
		{"caret zero major", "^0.2.3", false},
		{"caret zero minor", "^0.0.3", false},

		// valid: wildcards
		{"wildcard star", "*", false},
		{"wildcard x", "1.x", false},
		{"wildcard X", "1.X", false},
		{"wildcard patch", "1.2.*", false},

		// valid: compound
		{"and comma", ">=1.0.0, <2.0.0", false},
		{"or pipe", ">=1.0.0 || >=2.0.0", false},
		{"hyphen range", "1.2.3 - 2.3.4", false},
		{"bare partial", "1.2", false},
		{"bare major", "1", false},

		// valid: with spaces
		{"spaces around op", ">= 1.2.3", false},

		// invalid
		{"empty", "", true},
		{"empty or group", ">=1.0.0 ||", true},
		{"empty and group", ">=1.0.0,", true},
		{"invalid version", ">abc", true},
		{"wildcard then non-wildcard", "1.*.3", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewConstraint(tt.input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConstraintsCheck(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		version    string
		want       bool
	}{
		// basic operators
		{"eq match", "=1.2.3", "1.2.3", true},
		{"eq no match", "=1.2.3", "1.2.4", false},
		{"neq match", "!=1.2.3", "1.2.4", true},
		{"neq no match", "!=1.2.3", "1.2.3", false},
		{"gt match", ">1.2.3", "1.3.0", true},
		{"gt no match", ">1.2.3", "1.2.3", false},
		{"gte match equal", ">=1.2.3", "1.2.3", true},
		{"gte match greater", ">=1.2.3", "2.0.0", true},
		{"gte no match", ">=1.2.3", "1.2.2", false},
		{"lt match", "<1.2.3", "1.2.2", true},
		{"lt no match", "<1.2.3", "1.2.3", false},
		{"lte match equal", "<=1.2.3", "1.2.3", true},
		{"lte match less", "<=1.2.3", "1.0.0", true},
		{"lte no match", "<=1.2.3", "1.2.4", false},

		// bare version (exact match for full version)
		{"bare full match", "1.2.3", "1.2.3", true},
		{"bare full no match", "1.2.3", "1.2.4", false},

		// tilde
		{"tilde patch match", "~1.2.3", "1.2.5", true},
		{"tilde patch lower bound", "~1.2.3", "1.2.3", true},
		{"tilde patch upper exclude", "~1.2.3", "1.3.0", false},
		{"tilde minor match", "~1.2", "1.2.9", true},
		{"tilde minor upper exclude", "~1.2", "1.3.0", false},
		{"tilde major match", "~1", "1.9.9", true},
		{"tilde major upper exclude", "~1", "2.0.0", false},

		// caret
		{"caret major match", "^1.2.3", "1.9.9", true},
		{"caret major lower bound", "^1.2.3", "1.2.3", true},
		{"caret major upper exclude", "^1.2.3", "2.0.0", false},
		{"caret zero major", "^0.2.3", "0.2.9", true},
		{"caret zero major upper exclude", "^0.2.3", "0.3.0", false},
		{"caret zero minor", "^0.0.3", "0.0.3", true},
		{"caret zero minor upper exclude", "^0.0.3", "0.0.4", false},

		// wildcards
		{"star matches all", "*", "99.99.99", true},
		{"x matches all", "x", "1.2.3", true},
		{"major wildcard", "1.*", "1.9.9", true},
		{"major wildcard upper exclude", "1.*", "2.0.0", false},
		{"minor wildcard", "1.2.x", "1.2.9", true},
		{"minor wildcard upper exclude", "1.2.x", "1.3.0", false},

		// hyphen ranges
		{"hyphen full match", "1.2.3 - 2.3.4", "1.5.0", true},
		{"hyphen full lower bound", "1.2.3 - 2.3.4", "1.2.3", true},
		{"hyphen full upper bound", "1.2.3 - 2.3.4", "2.3.4", true},
		{"hyphen full below", "1.2.3 - 2.3.4", "1.2.2", false},
		{"hyphen full above", "1.2.3 - 2.3.4", "2.3.5", false},
		{"hyphen partial upper", "1.2.3 - 2.3", "2.3.9", true},
		{"hyphen partial upper exclude", "1.2.3 - 2.3", "2.4.0", false},
		{"hyphen partial lower", "1.2 - 2.3.4", "1.2.0", true},

		// bare partial versions (range expansion)
		{"bare minor match", "1.2", "1.2.5", true},
		{"bare minor lower bound", "1.2", "1.2.0", true},
		{"bare minor upper exclude", "1.2", "1.3.0", false},
		{"bare major match", "1", "1.5.5", true},
		{"bare major upper exclude", "1", "2.0.0", false},

		// AND (comma)
		{"and match", ">=1.0.0, <2.0.0", "1.5.0", true},
		{"and lower fail", ">=1.0.0, <2.0.0", "0.9.0", false},
		{"and upper fail", ">=1.0.0, <2.0.0", "2.0.0", false},

		// OR (||)
		{"or first match", ">=1.0.0 || >=3.0.0", "1.5.0", true},
		{"or second match", "<1.0.0 || >=3.0.0", "3.0.0", true},
		{"or no match", ">2.0.0 || <1.0.0", "1.5.0", false},

		// pre-release
		{"pre-release match gte", ">=1.0.0-alpha", "1.0.0-alpha", true},
		{"pre-release match release", ">=1.0.0-alpha", "1.0.0", true},
		{"pre-release below", ">=1.0.0-beta", "1.0.0-alpha", false},

		// build metadata ignored
		{"build ignored in constraint", "=1.2.3", "1.2.3+build", true},

		// pessimistic operator
		{"pessimistic match", "~>1.2.3", "1.2.5", true},
		{"pessimistic upper exclude", "~>1.2.3", "1.3.0", false},

		// double equal
		{"double equal match", "==1.2.3", "1.2.3", true},
		{"double equal no match", "==1.2.3", "1.2.4", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewConstraint(tt.constraint)
			require.NoError(t, err)

			v := MustParse(tt.version)
			assert.Equal(t, tt.want, c.Check(v))
		})
	}
}

func TestConstraintsString(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple", ">=1.2.3"},
		{"compound", ">=1.0.0, <2.0.0"},
		{"or", ">=1.0.0 || >=2.0.0"},
		{"tilde", "~1.2.3"},
		{"caret", "^1.2.3"},
		{"wildcard", "1.*"},
		{"hyphen", "1.2.3 - 2.3.4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewConstraint(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.input, c.String())
		})
	}
}

func BenchmarkNewConstraint(b *testing.B) {
	benchmarks := []struct {
		name  string
		input string
	}{
		{"simple", ">=1.2.3"},
		{"tilde", "~1.2.3"},
		{"caret", "^1.2.3"},
		{"compound", ">=1.0.0, <2.0.0"},
		{"or", ">=1.0.0 || <0.5.0"},
		{"hyphen", "1.2.3 - 2.3.4"},
		{"wildcard", "1.*"},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_, _ = NewConstraint(bb.input)
			}
		})
	}
}

func BenchmarkConstraintsCheck(b *testing.B) {
	benchmarks := []struct {
		name       string
		constraint string
		version    string
	}{
		{"simple_gte", ">=1.2.3", "1.5.0"},
		{"tilde", "~1.2.3", "1.2.5"},
		{"compound_and", ">=1.0.0, <2.0.0", "1.5.0"},
		{"compound_or", ">=1.0.0 || >=3.0.0", "3.5.0"},
		{"wildcard", "*", "1.2.3"},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			c, _ := NewConstraint(bb.constraint)
			v := MustParse(bb.version)

			b.ReportAllocs()

			for b.Loop() {
				c.Check(v)
			}
		})
	}
}

func FuzzNewConstraint(f *testing.F) {
	f.Add(">=1.2.3")
	f.Add("~1.2.3")
	f.Add("^1.2.3")
	f.Add("*")
	f.Add("1.*")
	f.Add("1.2.x")
	f.Add(">=1.0.0, <2.0.0")
	f.Add(">=1.0.0 || <0.5.0")
	f.Add("1.2.3 - 2.3.4")
	f.Add("!=1.2.3")
	f.Add("")
	f.Add("invalid")
	f.Add(">=")
	f.Add("1.*.3")

	f.Fuzz(func(t *testing.T, input string) {
		c, err := NewConstraint(input)
		if err != nil {
			return
		}

		// String round-trip preserves original input.
		assert.Equal(t, input, c.String())

		// Check must not panic on any valid version.
		v := Version{Major: 1, Minor: 2, Patch: 3}
		c.Check(v)
	})
}
