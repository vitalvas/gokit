package xflags

import (
	"os"
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

func TestIsSet(t *testing.T) {
	newParser := func(t *testing.T) (*Parser, *string) {
		t.Helper()
		var name string
		p := New("app")
		require.NoError(t, p.StringVar(&name, "name", "", "", ""))
		return p, &name
	}

	t.Run("set on command line", func(t *testing.T) {
		p, _ := newParser(t)
		require.NoError(t, p.Parse([]string{"--name=x"}))
		assert.True(t, p.IsSet("name"))
	})

	t.Run("set from environment", func(t *testing.T) {
		var name string
		p := New("app")
		require.NoError(t, p.StringVar(&name, "name", "", "", ""))
		p.byLong["name"].Env = "XF_ISSET"
		t.Setenv("XF_ISSET", "y")
		require.NoError(t, p.Parse(nil))
		assert.True(t, p.IsSet("name"))
	})

	t.Run("default only does not count", func(t *testing.T) {
		var name string
		p := New("app")
		require.NoError(t, p.StringVar(&name, "name", "", "fallback", ""))
		require.NoError(t, p.Parse(nil))
		assert.Equal(t, "fallback", name)
		assert.False(t, p.IsSet("name"))
	})

	t.Run("unknown name", func(t *testing.T) {
		p, _ := newParser(t)
		require.NoError(t, p.Parse(nil))
		assert.False(t, p.IsSet("nope"))
	})
}

func TestCount(t *testing.T) {
	newParser := func(t *testing.T) *Parser {
		t.Helper()
		var v int
		p := New("app")
		require.NoError(t, p.CountVar(&v, "verbose", "v", ""))
		return p
	}

	t.Run("counts occurrences", func(t *testing.T) {
		p := newParser(t)
		require.NoError(t, p.Parse([]string{"-vvv"}))
		assert.Equal(t, 3, p.Count("verbose"))
	})

	t.Run("unset is zero", func(t *testing.T) {
		p := newParser(t)
		require.NoError(t, p.Parse(nil))
		assert.Equal(t, 0, p.Count("verbose"))
	})

	t.Run("unknown name is zero", func(t *testing.T) {
		p := newParser(t)
		require.NoError(t, p.Parse(nil))
		assert.Equal(t, 0, p.Count("nope"))
	})
}

func TestSetArgsExecute(t *testing.T) {
	t.Run("execute uses configured args", func(t *testing.T) {
		var name string
		p := New("app")
		require.NoError(t, p.StringVar(&name, "name", "", "", ""))
		p.SetArgs([]string{"--name=configured"})
		require.NoError(t, p.Execute())
		assert.Equal(t, "configured", name)
		assert.True(t, p.IsSet("name"))
	})

	t.Run("execute surfaces parse error", func(t *testing.T) {
		p := New("app")
		p.SetArgs([]string{"does-not-exist"})
		require.Error(t, p.Execute())
	})

	t.Run("execute falls back to os.Args", func(t *testing.T) {
		saved := os.Args
		t.Cleanup(func() { os.Args = saved })
		var name string
		p := New("app")
		require.NoError(t, p.StringVar(&name, "name", "", "", ""))
		os.Args = []string{"app", "--name=fromargs"}
		require.NoError(t, p.Execute())
		assert.Equal(t, "fromargs", name)
	})
}

func TestFind(t *testing.T) {
	buildRoot := func(t *testing.T) *Parser {
		t.Helper()
		p := New("app")
		_, err := p.AddCommand("serve", nil)
		require.NoError(t, err)
		return p
	}

	t.Run("finds subcommand", func(t *testing.T) {
		root := buildRoot(t)
		cmd, rest, err := root.Find([]string{"serve"})
		require.NoError(t, err)
		assert.Equal(t, "serve", cmd.Name())
		assert.Empty(t, rest)
	})

	t.Run("returns remaining args", func(t *testing.T) {
		root := buildRoot(t)
		cmd, rest, err := root.Find([]string{"serve", "--addr", ":8080"})
		require.NoError(t, err)
		assert.Equal(t, "serve", cmd.Name())
		assert.Equal(t, []string{"--addr", ":8080"}, rest)
	})

	t.Run("nested subcommands", func(t *testing.T) {
		root := New("app")
		parent, err := root.AddCommand("remote", nil)
		require.NoError(t, err)
		_, err = parent.AddCommand("add", nil)
		require.NoError(t, err)
		cmd, _, err := root.Find([]string{"remote", "add"})
		require.NoError(t, err)
		assert.Equal(t, "add", cmd.Name())
	})

	t.Run("unknown command fails", func(t *testing.T) {
		root := buildRoot(t)
		_, _, err := root.Find([]string{"does-not-exist"})
		require.Error(t, err)
	})

	t.Run("leading flag stops descent", func(t *testing.T) {
		root := buildRoot(t)
		cmd, rest, err := root.Find([]string{"--help"})
		require.NoError(t, err)
		assert.Same(t, root.Command, cmd)
		assert.Equal(t, []string{"--help"}, rest)
	})

	t.Run("no subcommands returns self", func(t *testing.T) {
		root := New("app")
		cmd, rest, err := root.Find([]string{"positional"})
		require.NoError(t, err)
		assert.Same(t, root.Command, cmd)
		assert.Equal(t, []string{"positional"}, rest)
	})
}
