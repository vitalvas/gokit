package xflags

import (
	"fmt"
	"net/netip"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// upperString is a named string type whose UnmarshalText uppercases its input,
// proving that a custom parser takes precedence over the primitive string kind.
type upperString string

func (u *upperString) UnmarshalText(text []byte) error {
	*u = upperString(strings.ToUpper(string(text)))
	return nil
}

// failingType always fails to unmarshal, used to assert error propagation.
type failingType struct{}

func (failingType) UnmarshalText([]byte) error {
	return fmt.Errorf("always fails")
}

func TestTextUnmarshaler(t *testing.T) {
	t.Run("stdlib netip.AddrPort via pointer receiver", func(t *testing.T) {
		var o struct {
			Addr netip.AddrPort `long:"addr"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--addr=127.0.0.1:8428"}...))
		assert.Equal(t, netip.MustParseAddrPort("127.0.0.1:8428"), o.Addr)
	})

	t.Run("custom type overrides primitive kind", func(t *testing.T) {
		var o struct {
			Name upperString `long:"name"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--name=hello"}...))
		assert.Equal(t, upperString("HELLO"), o.Name)
	})

	t.Run("pointer destination allocates and parses", func(t *testing.T) {
		var o struct {
			Addr *netip.AddrPort `long:"addr"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--addr=[::1]:9000"}...))
		require.NotNil(t, o.Addr)
		assert.Equal(t, netip.MustParseAddrPort("[::1]:9000"), *o.Addr)
	})

	t.Run("default value is unmarshaled", func(t *testing.T) {
		var o struct {
			Addr netip.AddrPort `long:"addr" default:"0.0.0.0:8428"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{}...))
		assert.Equal(t, netip.MustParseAddrPort("0.0.0.0:8428"), o.Addr)
	})

	t.Run("append slice of unmarshaler elements", func(t *testing.T) {
		var o struct {
			Addrs []netip.AddrPort `long:"addr"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--addr=10.0.0.1:1", "--addr=10.0.0.2:2"}...))
		assert.Equal(t, []netip.AddrPort{
			netip.MustParseAddrPort("10.0.0.1:1"),
			netip.MustParseAddrPort("10.0.0.2:2"),
		}, o.Addrs)
	})

	t.Run("merge map with unmarshaler values", func(t *testing.T) {
		var o struct {
			Peers map[string]netip.AddrPort `long:"peer"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--peer=a=10.0.0.1:1"}...))
		assert.Equal(t, map[string]netip.AddrPort{
			"a": netip.MustParseAddrPort("10.0.0.1:1"),
		}, o.Peers)
	})

	t.Run("invalid value reports unmarshal error", func(t *testing.T) {
		var o struct {
			Addr netip.AddrPort `long:"addr"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		err := p.Parse([]string{"--addr=not-an-address"}...)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--addr")
	})

	t.Run("unmarshaler error is wrapped", func(t *testing.T) {
		var o struct {
			V failingType `long:"v"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		err := p.Parse([]string{"--v=x"}...)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "always fails")
	})
}
