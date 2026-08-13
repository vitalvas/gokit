package xflags

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegerBase(t *testing.T) {
	t.Run("hex via base tag", func(t *testing.T) {
		var o struct {
			Mask int `long:"mask" base:"16"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--mask=ff"}))
		assert.Equal(t, 255, o.Mask)
	})

	t.Run("base 0 detects prefix", func(t *testing.T) {
		var o struct {
			A int `long:"a" base:"0"`
			B int `long:"b" base:"0"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--a=0x10", "--b=0o17"}))
		assert.Equal(t, 16, o.A)
		assert.Equal(t, 15, o.B)
	})

	t.Run("invalid base tag errors", func(t *testing.T) {
		var o struct {
			A int `long:"a" base:"xx"`
		}
		p := New("app")
		require.Error(t, p.AddGroup("", &o))
	})

	t.Run("default base 10", func(t *testing.T) {
		var o struct {
			A int `long:"a"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--a=42"}))
		assert.Equal(t, 42, o.A)
	})
}

func TestNilCallbackNoPanic(t *testing.T) {
	t.Run("nil func field is a no-op", func(t *testing.T) {
		o := struct {
			Fn func(string) error `long:"fn"`
		}{}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NotPanics(t, func() {
			require.NoError(t, p.Parse([]string{"--fn=x"}))
		})
	})
}
