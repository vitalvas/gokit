package xflags

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseByteSize(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want ByteSize
	}{
		{"0", 0},
		{"512", 512},
		{"1024", Kibibyte},
		{"1B", Byte},
		{"8B", 8 * Byte},
		{"1KiB", Kibibyte},
		{"4KiB", 4 * Kibibyte},
		{"32MiB", 32 * Mebibyte},
		{"1GiB", Gibibyte},
		{"2TiB", 2 * Tebibyte},
		{" 16MiB ", 16 * Mebibyte},
		{"16 MiB", 16 * Mebibyte},
	} {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseByteSize(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseByteSizeErrors(t *testing.T) {
	for _, in := range []string{"", "   ", "abc", "1.5MiB", "-1", "10GB", "99999999999999999999TiB"} {
		t.Run(in, func(t *testing.T) {
			_, err := ParseByteSize(in)
			require.Error(t, err)
		})
	}
}

func TestByteSizeOverflow(t *testing.T) {
	// A unit multiplier that pushes the value past uint64 must error, not wrap.
	_, err := ParseByteSize("16TiB")
	require.NoError(t, err)
	_, err = ParseByteSize("16777216TiB")
	require.Error(t, err)
}

func TestByteSizeString(t *testing.T) {
	for _, tc := range []struct {
		in   ByteSize
		want string
	}{
		{0, "0B"},
		{1, "1B"},
		{512, "512B"},
		{Kibibyte, "1KiB"},
		{32 * Mebibyte, "32MiB"},
		{Gibibyte, "1GiB"},
		{Tebibyte, "1TiB"},
		{Kibibyte + 1, "1025B"},
		{ByteSize(math.MaxUint64), "18446744073709551615B"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.in.String())
			text, err := tc.in.MarshalText()
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(text))
		})
	}
}

func TestByteSizeRoundTrip(t *testing.T) {
	for _, in := range []string{"0B", "1B", "4KiB", "32MiB", "1GiB", "2TiB"} {
		parsed, err := ParseByteSize(in)
		require.NoError(t, err)
		assert.Equal(t, in, parsed.String())
	}
}

func TestByteSizeAsOption(t *testing.T) {
	t.Run("parses suffixed value", func(t *testing.T) {
		var o struct {
			MaxInsert ByteSize `long:"max-insert-request-size" default:"32MiB"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--max-insert-request-size=64MiB"}...))
		assert.Equal(t, 64*Mebibyte, o.MaxInsert)
	})

	t.Run("applies suffixed default", func(t *testing.T) {
		var o struct {
			MaxInsert ByteSize `long:"max-insert-request-size" default:"32MiB"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{}...))
		assert.Equal(t, 32*Mebibyte, o.MaxInsert)
	})

	t.Run("invalid value fails with option name", func(t *testing.T) {
		var o struct {
			MaxInsert ByteSize `long:"max-insert-request-size"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		err := p.Parse([]string{"--max-insert-request-size=10GB"}...)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--max-insert-request-size")
	})
}
