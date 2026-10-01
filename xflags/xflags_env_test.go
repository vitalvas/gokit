package xflags

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrecedence(t *testing.T) {
	t.Run("cli beats env and default", func(t *testing.T) {
		t.Setenv("XF_NAME", "fromenv")
		var opts struct {
			Name string `long:"name" env:"XF_NAME" default:"fromdefault"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"--name=fromcli"}))
		assert.Equal(t, "fromcli", opts.Name)
	})

	t.Run("env beats default", func(t *testing.T) {
		t.Setenv("XF_NAME", "fromenv")
		var opts struct {
			Name string `long:"name" env:"XF_NAME" default:"fromdefault"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse(nil))
		assert.Equal(t, "fromenv", opts.Name)
	})

	t.Run("default used when nothing set", func(t *testing.T) {
		var opts struct {
			Name string `long:"name" env:"XF_UNSET" default:"fromdefault"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse(nil))
		assert.Equal(t, "fromdefault", opts.Name)
	})

	t.Run("empty env falls back to default", func(t *testing.T) {
		t.Setenv("XF_EMPTY", "")
		var opts struct {
			Name string `long:"name" env:"XF_EMPTY" default:"fromdefault"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse(nil))
		assert.Equal(t, "fromdefault", opts.Name)
	})
}

func TestEnvMultiValue(t *testing.T) {
	t.Run("env seeds slice", func(t *testing.T) {
		t.Setenv("XF_TAGS", "a,b,c")
		var opts struct {
			Tags []string `long:"tag" env:"XF_TAGS" choice:"append"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse(nil))
		assert.Equal(t, []string{"a", "b", "c"}, opts.Tags)
	})

	t.Run("env seeds map", func(t *testing.T) {
		t.Setenv("XF_LABELS", "env=prod,tier=web")
		var opts struct {
			Labels map[string]string `long:"label" env:"XF_LABELS"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse(nil))
		assert.Equal(t, map[string]string{"env": "prod", "tier": "web"}, opts.Labels)
	})

	t.Run("blank entries are skipped", func(t *testing.T) {
		t.Setenv("XF_TAGS", "a,,b, ,c")
		var opts struct {
			Tags []string `long:"tag" env:"XF_TAGS" choice:"append"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse(nil))
		assert.Equal(t, []string{"a", "b", "c"}, opts.Tags)
	})
}
