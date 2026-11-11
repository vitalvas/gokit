package xflags

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Duration
	}{
		{"0", 0},
		{"30", Duration(30 * time.Second)},
		{"90", Duration(90 * time.Second)},
		{"500ms", Duration(500 * time.Millisecond)},
		{"5s", Duration(5 * time.Second)},
		{"2m", Duration(2 * time.Minute)},
		{"1h", Duration(time.Hour)},
		{"1d", Day},
		{"14d", 14 * Day},
		{"1w", Week},
		{"2w", 2 * Week},
		{"1.5h", Duration(90 * time.Minute)},
		{"0.5d", Duration(12 * time.Hour)},
		{"1h30m", Duration(90 * time.Minute)},
		{" 10m ", Duration(10 * time.Minute)},
	} {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseDuration(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseDurationErrors(t *testing.T) {
	for _, in := range []string{"", "   ", "abc", "5x", "d", "w", "inf", "nan", "1e400w"} {
		t.Run(in, func(t *testing.T) {
			_, err := ParseDuration(in)
			require.Error(t, err)
		})
	}
}

func TestParseDurationOverflow(t *testing.T) {
	_, err := ParseDuration("100000000000000000w")
	require.Error(t, err)
}

func TestDurationString(t *testing.T) {
	for _, tc := range []struct {
		in   Duration
		want string
	}{
		{0, "0s"},
		{Duration(5 * time.Second), "5s"},
		{Duration(90 * time.Minute), "1h30m0s"},
		{Day, "24h0m0s"},
		{Week, "168h0m0s"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.in.String())
			text, err := tc.in.MarshalText()
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(text))
		})
	}
}

func TestDurationRoundTrip(t *testing.T) {
	for _, d := range []Duration{0, Duration(5 * time.Second), Day, Week, Duration(90 * time.Minute)} {
		parsed, err := ParseDuration(d.String())
		require.NoError(t, err)
		assert.Equal(t, d, parsed)
	}
}

func TestDurationUnmarshalRoundsBareSeconds(t *testing.T) {
	var d Duration
	require.NoError(t, d.UnmarshalText([]byte("45")))
	assert.Equal(t, Duration(45*time.Second), d)
}

func TestDurationAsOption(t *testing.T) {
	t.Run("parses day suffix", func(t *testing.T) {
		var o struct {
			Backfill Duration `long:"ingest-max-backfill-age" default:"14d"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--ingest-max-backfill-age=7d"}...))
		assert.Equal(t, 7*Day, o.Backfill)
	})

	t.Run("applies day-suffixed default", func(t *testing.T) {
		var o struct {
			Backfill Duration `long:"ingest-max-backfill-age" default:"14d"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{}...))
		assert.Equal(t, 14*Day, o.Backfill)
	})

	t.Run("bare number is seconds", func(t *testing.T) {
		var o struct {
			Dedup Duration `long:"dedup-min-interval" default:"0"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--dedup-min-interval=30"}...))
		assert.Equal(t, Duration(30*time.Second), o.Dedup)
	})

	t.Run("invalid value fails with option name", func(t *testing.T) {
		var o struct {
			Backfill Duration `long:"ingest-max-backfill-age"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		err := p.Parse([]string{"--ingest-max-backfill-age=5x"}...)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--ingest-max-backfill-age")
	})
}
