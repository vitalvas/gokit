package xflags

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddGroup(t *testing.T) {
	t.Run("requires pointer to struct", func(t *testing.T) {
		p := New("app")
		require.Error(t, p.AddGroup("", struct{}{}))
		require.Error(t, p.AddGroup("", (*struct{})(nil)))

		i := 0
		require.Error(t, p.AddGroup("", &i))
	})

	t.Run("registers options", func(t *testing.T) {
		var opts struct {
			Name string `long:"name"`
			V    int    `short:"v"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		assert.Len(t, p.options, 2)
		assert.Contains(t, p.byLong, "name")
		assert.Contains(t, p.byShort, "v")
	})

	t.Run("skips fields without option tags", func(t *testing.T) {
		var opts struct {
			Name    string `long:"name"`
			Ignored string
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		assert.Len(t, p.options, 1)
	})

	t.Run("rejects duplicate long names", func(t *testing.T) {
		var opts struct {
			A string `long:"dup"`
			B string `long:"dup"`
		}
		p := New("app")
		require.Error(t, p.AddGroup("", &opts))
	})

	t.Run("rejects duplicate short names", func(t *testing.T) {
		var opts struct {
			A string `short:"x"`
			B string `short:"x"`
		}
		p := New("app")
		require.Error(t, p.AddGroup("", &opts))
	})
}

func TestValidate(t *testing.T) {
	t.Run("unknown option", func(t *testing.T) {
		p := New("app")
		require.Error(t, p.Validate("missing", func(any) error { return nil }))
	})

	t.Run("attaches callback", func(t *testing.T) {
		var opts struct {
			Port int `long:"port"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Validate("port", func(any) error { return nil }))
		assert.NotNil(t, p.byLong["port"].validate)
	})
}
