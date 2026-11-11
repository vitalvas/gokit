package xflags

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseListenAddr(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want netip.AddrPort
	}{
		{":8428", netip.MustParseAddrPort("[::]:8428")},
		{":0", netip.MustParseAddrPort("[::]:0")},
		{"127.0.0.1:9000", netip.MustParseAddrPort("127.0.0.1:9000")},
		{"0.0.0.0:8428", netip.MustParseAddrPort("0.0.0.0:8428")},
		{"[::1]:9000", netip.MustParseAddrPort("[::1]:9000")},
		{"[::]:8428", netip.MustParseAddrPort("[::]:8428")},
	} {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseListenAddr(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.AddrPort())
		})
	}
}

func TestParseListenAddrErrors(t *testing.T) {
	for _, in := range []string{"", ":", "not-an-address", "localhost:8428", ":notaport", "1.2.3.4"} {
		t.Run(in, func(t *testing.T) {
			_, err := ParseListenAddr(in)
			require.Error(t, err)
		})
	}
}

func TestListenAddrString(t *testing.T) {
	addr, err := ParseListenAddr(":8428")
	require.NoError(t, err)
	assert.Equal(t, "[::]:8428", addr.String())

	text, err := addr.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, "[::]:8428", string(text))
}

func TestListenAddrRoundTrip(t *testing.T) {
	for _, in := range []string{"[::]:8428", "127.0.0.1:9000", "[::1]:9000"} {
		parsed, err := ParseListenAddr(in)
		require.NoError(t, err)
		again, err := ParseListenAddr(parsed.String())
		require.NoError(t, err)
		assert.Equal(t, parsed, again)
	}
}

func TestListenAddrAsOption(t *testing.T) {
	t.Run("port shorthand default", func(t *testing.T) {
		var o struct {
			HTTP ListenAddr `long:"http-listen-addr" default:":8428"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{}...))
		assert.Equal(t, netip.MustParseAddrPort("[::]:8428"), o.HTTP.AddrPort())
	})

	t.Run("explicit value overrides default", func(t *testing.T) {
		var o struct {
			HTTP ListenAddr `long:"http-listen-addr" default:":8428"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--http-listen-addr=127.0.0.1:9000"}...))
		assert.Equal(t, netip.MustParseAddrPort("127.0.0.1:9000"), o.HTTP.AddrPort())
	})

	t.Run("invalid value fails with option name", func(t *testing.T) {
		var o struct {
			HTTP ListenAddr `long:"http-listen-addr"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		err := p.Parse([]string{"--http-listen-addr=localhost:9000"}...)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--http-listen-addr")
	})
}
