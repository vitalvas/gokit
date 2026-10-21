package xflags

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootAndSubcommandDispatch(t *testing.T) {
	// ./app runs the root handler; ./app random runs the subcommand handler.
	newApp := func() (*Parser, *string) {
		ran := new(string)
		p := New("app")
		p.SetRun(func(context.Context, *Command, []string) error { *ran = "root"; return nil })
		random := NewCommand("random", "")
		random.SetRun(func(context.Context, *Command, []string) error { *ran = "random"; return nil })
		require.NoError(t, p.AddCommand(random))
		return p, ran
	}

	t.Run("no args runs root", func(t *testing.T) {
		p, ran := newApp()
		p.SetArgs([]string{})
		require.NoError(t, p.Execute(context.Background()))
		assert.Equal(t, "root", *ran)
	})

	t.Run("subcommand name runs subcommand", func(t *testing.T) {
		p, ran := newApp()
		p.SetArgs([]string{"random"})
		require.NoError(t, p.Execute(context.Background()))
		assert.Equal(t, "random", *ran)
	})

	t.Run("root flags then no subcommand runs root", func(t *testing.T) {
		ran := new(string)
		var verbose bool
		p := New("app")
		require.NoError(t, p.BoolVar(&verbose, "verbose", "v", false, ""))
		p.SetRun(func(context.Context, *Command, []string) error { *ran = "root"; return nil })
		require.NoError(t, p.AddCommand(NewCommand("random", "")))

		p.SetArgs([]string{"-v"})
		require.NoError(t, p.Execute(context.Background()))
		assert.Equal(t, "root", *ran)
		assert.True(t, verbose)
	})

	t.Run("unknown subcommand is a root positional", func(t *testing.T) {
		var got []string
		p := New("app")
		p.SetRun(func(_ context.Context, _ *Command, args []string) error { got = args; return nil })
		require.NoError(t, p.AddCommand(NewCommand("random", "")))

		p.SetArgs([]string{"other"})
		require.NoError(t, p.Execute(context.Background()))
		assert.Equal(t, []string{"other"}, got)
	})
}

func TestNewCommandAndAttach(t *testing.T) {
	t.Run("standalone command has no parent until attached", func(t *testing.T) {
		cmd := NewCommand("server", "run the server")
		assert.Equal(t, "server", cmd.Name())
		assert.Nil(t, cmd.parent)

		p := New("app")
		require.NoError(t, p.AddCommand(cmd))
		assert.Same(t, p.Command, cmd.parent)
	})

	t.Run("attaches multiple children in one call", func(t *testing.T) {
		p := New("app")
		a := NewCommand("a", "")
		b := NewCommand("b", "")
		require.NoError(t, p.AddCommand(a, b))

		cmd, _, err := p.Find([]string{"a"})
		require.NoError(t, err)
		assert.Equal(t, "a", cmd.Name())
		cmd, _, err = p.Find([]string{"b"})
		require.NoError(t, err)
		assert.Equal(t, "b", cmd.Name())
	})

	t.Run("builds tree across construction boundaries", func(t *testing.T) {
		// Simulates a command defined in its own package and wired into the
		// root elsewhere: the child is fully built, then attached.
		user := NewCommand("user", "User management commands")
		user.SetDescription("Commands for managing user accounts.")
		var ran bool
		create := NewCommand("create", "")
		create.SetRun(func(context.Context, *Command, []string) error { ran = true; return nil })
		require.NoError(t, user.AddCommand(create))

		p := New("app")
		require.NoError(t, p.AddCommand(user))

		p.SetArgs([]string{"user", "create"})
		require.NoError(t, p.Execute(context.Background()))
		assert.True(t, ran)
	})

	t.Run("nil child errors", func(t *testing.T) {
		p := New("app")
		require.Error(t, p.AddCommand(nil))
	})

	t.Run("already-attached child errors", func(t *testing.T) {
		p := New("app")
		child := NewCommand("c", "")
		require.NoError(t, p.AddCommand(child))

		other := New("other")
		require.Error(t, other.AddCommand(child))
	})

	t.Run("duplicate name errors and attaches nothing", func(t *testing.T) {
		p := New("app")
		require.NoError(t, p.AddCommand(NewCommand("dup", "")))

		fresh := NewCommand("fresh", "")
		require.Error(t, p.AddCommand(fresh, NewCommand("dup", "")))
		assert.Nil(t, fresh.parent, "no child attaches when one in the batch is invalid")
		_, _, err := p.Find([]string{"fresh"})
		require.Error(t, err)
	})

	t.Run("invalid name errors at attach", func(t *testing.T) {
		p := New("app")
		require.Error(t, p.AddCommand(NewCommand("--bad", "")))
	})
}

func TestRunDispatch(t *testing.T) {
	t.Run("executes selected command handler", func(t *testing.T) {
		var serverOpts struct {
			Port int `long:"port" default:"8080"`
		}
		p := New("app")
		server := NewCommand("server", "")
		require.NoError(t, server.AddGroup("", &serverOpts))
		require.NoError(t, p.AddCommand(server))

		var ran bool
		server.SetRun(func(ctx context.Context, cmd *Command, args []string) error {
			ran = true
			assert.NotNil(t, ctx)
			assert.Equal(t, "server", cmd.Name())
			assert.Empty(t, args)
			return nil
		})

		p.SetArgs([]string{"server", "--port=9090"})
		require.NoError(t, p.Execute(context.Background()))
		assert.True(t, ran)
		assert.Equal(t, 9090, serverOpts.Port)
	})

	t.Run("passes positional arguments", func(t *testing.T) {
		p := New("app")
		var got []string
		p.SetRun(func(_ context.Context, _ *Command, args []string) error {
			got = args
			return nil
		})

		p.SetArgs([]string{"a", "b", "c"})
		require.NoError(t, p.Execute(context.Background()))
		assert.Equal(t, []string{"a", "b", "c"}, got)
		assert.Equal(t, []string{"a", "b", "c"}, p.Args())
	})

	t.Run("positional arguments after terminator", func(t *testing.T) {
		var opts struct {
			Name string `long:"name"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		var got []string
		p.SetRun(func(_ context.Context, _ *Command, args []string) error {
			got = args
			return nil
		})

		p.SetArgs([]string{"--name=x", "--", "--not-a-flag", "value"})
		require.NoError(t, p.Execute(context.Background()))
		assert.Equal(t, "x", opts.Name)
		assert.Equal(t, []string{"--not-a-flag", "value"}, got)
	})

	t.Run("dispatches to deepest command", func(t *testing.T) {
		p := New("app")
		server := NewCommand("server", "")
		require.NoError(t, p.AddCommand(server))
		start := NewCommand("start", "")
		require.NoError(t, server.AddCommand(start))

		var ran string
		server.SetRun(func(context.Context, *Command, []string) error { ran = "server"; return nil })
		start.SetRun(func(context.Context, *Command, []string) error { ran = "start"; return nil })

		p.SetArgs([]string{"server", "start"})
		require.NoError(t, p.Execute(context.Background()))
		assert.Equal(t, "start", ran)
	})

	t.Run("propagates handler error", func(t *testing.T) {
		p := New("app")
		sentinel := errors.New("boom")
		p.SetRun(func(context.Context, *Command, []string) error { return sentinel })

		p.SetArgs([]string{})
		require.ErrorIs(t, p.Execute(context.Background()), sentinel)
	})

	t.Run("prints help when no handler", func(t *testing.T) {
		p := New("app")
		server := NewCommand("server", "")
		require.NoError(t, p.AddCommand(server))
		require.NoError(t, server.AddCommand(NewCommand("start", "")))

		var buf bytes.Buffer
		p.SetOutput(&buf)
		p.SetArgs([]string{"server"})
		require.NoError(t, p.Execute(context.Background()))
		assert.Contains(t, buf.String(), "app server")
	})

	t.Run("built-in help skips dispatch", func(t *testing.T) {
		p := New("app")
		var ran bool
		p.SetRun(func(context.Context, *Command, []string) error { ran = true; return nil })

		var buf bytes.Buffer
		p.SetOutput(&buf)
		p.SetArgs([]string{"--help"})
		require.NoError(t, p.Execute(context.Background()))
		assert.False(t, ran)
	})

	t.Run("positional without handler errors", func(t *testing.T) {
		p := New("app")
		require.Error(t, p.Parse("bogus"))
	})

	t.Run("flags after first positional are positional", func(t *testing.T) {
		var opts struct {
			Name string `long:"name"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		var got []string
		p.SetRun(func(_ context.Context, _ *Command, args []string) error {
			got = args
			return nil
		})

		p.SetArgs([]string{"--name=x", "file", "--name=y"})
		require.NoError(t, p.Execute(context.Background()))
		assert.Equal(t, "x", opts.Name)
		assert.Equal(t, []string{"file", "--name=y"}, got)
	})

	t.Run("positionals reset between parses", func(t *testing.T) {
		p := New("app")
		p.SetRun(func(context.Context, *Command, []string) error { return nil })

		require.NoError(t, p.Parse("a", "b"))
		assert.Equal(t, []string{"a", "b"}, p.Args())
		require.NoError(t, p.Parse([]string{}...))
		assert.Empty(t, p.Args())
	})
}
