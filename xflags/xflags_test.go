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

func TestRegistrationRollback(t *testing.T) {
	p := New("app")
	var a, b string
	require.NoError(t, p.StringVar(&a, "a", "x", "", ""))
	require.Error(t, p.StringVar(&b, "b", "x", "", ""))
	require.Error(t, p.Parse([]string{"--b=leaked"}))
	assert.Empty(t, b)

	var bad struct {
		Fn       func(string) error `long:"a" short:"x" hidden:"true"`
		Sub      struct{}           `command:"sub"`
		New      string             `long:"new"`
		Conflict string             `short:"x"`
	}
	require.Error(t, p.AddGroup("bad", &bad))
	assert.False(t, p.byLong["a"].callback.IsValid())
	assert.False(t, p.byLong["a"].Hidden)
	assert.NotContains(t, p.byCommand, "sub")
	assert.NotContains(t, p.byLong, "new")
	require.NoError(t, p.StringVar(&b, "b", "b", "", ""))
	require.NoError(t, p.Parse([]string{"-b", "ok"}))
	assert.Equal(t, "ok", b)
}

func TestInvalidRegistrationNames(t *testing.T) {
	for _, name := range []string{"ab", "-", "=", " ", "\n"} {
		p := New("app")
		var b bool
		require.Error(t, p.BoolVar(&b, "flag", name, false, ""))
		assert.Empty(t, p.options)
	}
	for _, name := range []string{"-flag", "flag=value", "two words", "\n"} {
		p := New("app")
		var b bool
		require.Error(t, p.BoolVar(&b, name, "", false, ""))
		_, err := p.AddCommand(name, nil)
		require.Error(t, err)
	}
	var n int
	p := New("app")
	require.NoError(t, p.IntVar(&n, "n", "", 0, ""))
	for _, base := range []int{-1, 1, 37} {
		require.Error(t, p.SetBase("n", base))
	}
}
