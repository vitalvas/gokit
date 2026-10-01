package xflags

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespaces(t *testing.T) {
	t.Run("prefixes long names", func(t *testing.T) {
		var opts struct {
			Server struct {
				Port int    `long:"port" default:"8080"`
				Host string `long:"host" default:"0.0.0.0"`
			} `namespace:"server"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		assert.Contains(t, p.byLong, "server.port")
		assert.Contains(t, p.byLong, "server.host")
		require.NoError(t, p.Parse([]string{"--server.port=9090"}))
		assert.Equal(t, 9090, opts.Server.Port)
		assert.Equal(t, "0.0.0.0", opts.Server.Host)
	})

	t.Run("nested namespaces", func(t *testing.T) {
		var opts struct {
			A struct {
				B struct {
					Val int `long:"val"`
				} `namespace:"b"`
			} `namespace:"a"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"--a.b.val=7"}))
		assert.Equal(t, 7, opts.A.B.Val)
	})

	t.Run("pointer namespace is allocated", func(t *testing.T) {
		type inner struct {
			Val int `long:"val"`
		}
		var opts struct {
			Nested *inner `namespace:"nested"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"--nested.val=3"}))
		require.NotNil(t, opts.Nested)
		assert.Equal(t, 3, opts.Nested.Val)
	})
}

func TestCallbackSignatureValidation(t *testing.T) {
	t.Run("wrong return type", func(t *testing.T) {
		opts := struct {
			Fn func(string) string `long:"fn"`
		}{
			Fn: func(s string) string { return s },
		}
		p := New("app")
		require.Error(t, p.AddGroup("", &opts))
	})

	t.Run("too many args", func(t *testing.T) {
		opts := struct {
			Fn func(string, string) error `long:"fn"`
		}{
			Fn: func(string, string) error { return nil },
		}
		p := New("app")
		require.Error(t, p.AddGroup("", &opts))
	})
}

func TestMergePreservesMetadata(t *testing.T) {
	t.Run("func declared before value keeps short/env/default/required", func(t *testing.T) {
		t.Setenv("XF_MERGE", "")
		o := struct {
			PortFn func(int) error `long:"port"`
			Port   int             `long:"port" short:"p" default:"3000" required:"true"`
		}{
			PortFn: func(int) error { return nil },
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))

		opt := p.byLong["port"]
		assert.Equal(t, "p", opt.Short)
		assert.Equal(t, "3000", opt.Default)
		assert.True(t, opt.Required)
		require.Contains(t, p.byShort, "p")

		require.NoError(t, p.Parse([]string{"-p", "9090"}))
		assert.Equal(t, 9090, o.Port)
	})

	t.Run("conflicting short names error", func(t *testing.T) {
		o := struct {
			A int             `long:"port" short:"a"`
			B func(int) error `long:"port" short:"b"`
		}{B: func(int) error { return nil }}
		p := New("app")
		require.Error(t, p.AddGroup("", &o))
	})
}

func TestMergedValueMetadata(t *testing.T) {
	var seen int
	o := struct {
		Fn func(int) error `long:"n"`
		N  int             `long:"n" short:"n" base:"16" optional-value:"ff" hidden:"true" value-name:"HEX" default:"a"`
	}{Fn: func(n int) error { seen = n; return nil }}
	p := New("app")
	require.NoError(t, p.AddGroup("", &o))
	require.NoError(t, p.Parse([]string{"-n"}))
	assert.Equal(t, 255, o.N)
	assert.Equal(t, 255, seen)
	assert.NotContains(t, p.Help(), "--n")
	assert.Equal(t, "HEX", p.byLong["n"].ValueName)
	require.NoError(t, p.Parse(nil))
	assert.Equal(t, 10, o.N)
}
