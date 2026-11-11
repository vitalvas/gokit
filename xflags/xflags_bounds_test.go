package xflags

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoundsInt(t *testing.T) {
	t.Run("within range", func(t *testing.T) {
		var o struct {
			N int `long:"n" min:"1" max:"10"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--n=5"}...))
		assert.Equal(t, 5, o.N)
	})

	t.Run("below minimum", func(t *testing.T) {
		var o struct {
			N int `long:"n" min:"1"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		err := p.Parse([]string{"--n=0"}...)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--n")
		assert.Contains(t, err.Error(), "minimum")
	})

	t.Run("above maximum", func(t *testing.T) {
		var o struct {
			N int `long:"n" max:"10"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		err := p.Parse([]string{"--n=11"}...)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maximum")
	})

	t.Run("boundaries are inclusive", func(t *testing.T) {
		var o struct {
			N int `long:"n" min:"1" max:"10"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--n=1"}...))
		assert.Equal(t, 1, o.N)
		require.NoError(t, p.Parse([]string{"--n=10"}...))
		assert.Equal(t, 10, o.N)
	})
}

func TestBoundsUintAndFloat(t *testing.T) {
	t.Run("uint max (semaphore cap style)", func(t *testing.T) {
		var o struct {
			Inserts uint `long:"max-concurrent-inserts" default:"0" max:"2147483647"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--max-concurrent-inserts=1024"}...))
		assert.Equal(t, uint(1024), o.Inserts)
		require.Error(t, p.Parse([]string{"--max-concurrent-inserts=2147483648"}...))
	})

	t.Run("float range", func(t *testing.T) {
		var o struct {
			Ratio float64 `long:"ratio" min:"0" max:"1"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--ratio=0.5"}...))
		assert.Equal(t, 0.5, o.Ratio)
		require.Error(t, p.Parse([]string{"--ratio=1.5"}...))
	})
}

func TestBoundsCustomTypes(t *testing.T) {
	t.Run("ByteSize with suffixed bounds", func(t *testing.T) {
		var o struct {
			Size ByteSize `long:"size" min:"1KiB" max:"1MiB"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--size=512KiB"}...))
		assert.Equal(t, 512*Kibibyte, o.Size)
		require.Error(t, p.Parse([]string{"--size=2MiB"}...))
		require.Error(t, p.Parse([]string{"--size=512B"}...))
	})

	t.Run("Duration with suffixed bounds", func(t *testing.T) {
		var o struct {
			Interval Duration `long:"interval" min:"1s" max:"1h"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--interval=30s"}...))
		assert.Equal(t, Duration(30*time.Second), o.Interval)
		require.Error(t, p.Parse([]string{"--interval=2h"}...))
	})

	t.Run("time.Duration bounds", func(t *testing.T) {
		var o struct {
			Interval time.Duration `long:"interval" min:"1s" max:"1h"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--interval=30s"}...))
		assert.Equal(t, 30*time.Second, o.Interval)
		require.Error(t, p.Parse([]string{"--interval=90m"}...))
	})
}

func TestBoundsDefaultChecked(t *testing.T) {
	var o struct {
		N int `long:"n" min:"5" default:"10"`
	}
	p := New("app")
	require.NoError(t, p.AddGroup("", &o))
	require.NoError(t, p.Parse([]string{}...))
	assert.Equal(t, 10, o.N)

	var bad struct {
		N int `long:"n" min:"5" default:"1"`
	}
	bp := New("app")
	require.NoError(t, bp.AddGroup("", &bad))
	require.Error(t, bp.Parse([]string{}...))
}

func TestBoundsPointer(t *testing.T) {
	t.Run("nil optional skips check", func(t *testing.T) {
		var o struct {
			N *int `long:"n" min:"1"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{}...))
		assert.Nil(t, o.N)
	})

	t.Run("set pointer is checked", func(t *testing.T) {
		var o struct {
			N *int `long:"n" min:"1"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.Error(t, p.Parse([]string{"--n=0"}...))
	})
}

func TestBoundsRegistrationErrors(t *testing.T) {
	t.Run("non-numeric type", func(t *testing.T) {
		var o struct {
			S string `long:"s" min:"1"`
		}
		require.Error(t, New("app").AddGroup("", &o))
	})

	t.Run("unparseable bound", func(t *testing.T) {
		var o struct {
			N int `long:"n" max:"notnum"`
		}
		require.Error(t, New("app").AddGroup("", &o))
	})

	t.Run("min greater than max", func(t *testing.T) {
		var o struct {
			N int `long:"n" min:"10" max:"1"`
		}
		require.Error(t, New("app").AddGroup("", &o))
	})
}
