package xflags

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serverConfig and runServerCapture model an isolated command: a typed config
// and a free-function handler with no closure over test state.
type serverConfig struct {
	Port int    `long:"port" default:"8080"`
	Host string `long:"host"`
}

var captured *serverConfig
var capturedArgs []string
var capturedName string
var capturedCtx context.Context

func runServerCapture(ctx context.Context, cmd *Command, cfg *serverConfig, args []string) error {
	capturedCtx = ctx
	capturedName = cmd.Name()
	captured = cfg
	capturedArgs = args
	return nil
}

func TestBindCommand(t *testing.T) {
	t.Run("typed handler receives populated config", func(t *testing.T) {
		captured, capturedArgs, capturedName, capturedCtx = nil, nil, "", nil
		p := New("app")
		cfg, cmd, err := BindCommand("server", "", runServerCapture)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "server", cmd.Name())
		require.NoError(t, p.AddCommand(cmd))

		p.SetArgs([]string{"server", "--port=9090", "--host=db", "extra"})
		require.NoError(t, p.Execute(context.Background()))

		assert.NotNil(t, capturedCtx)
		assert.Equal(t, "server", capturedName)
		require.Same(t, cfg, captured)
		assert.Equal(t, 9090, cfg.Port)
		assert.Equal(t, "db", cfg.Host)
		assert.Equal(t, []string{"extra"}, capturedArgs)
	})

	t.Run("default applies when flag absent", func(t *testing.T) {
		p := New("app")
		cfg, cmd, err := BindCommand("server", "", func(context.Context, *Command, *serverConfig, []string) error { return nil })
		require.NoError(t, err)
		require.NoError(t, p.AddCommand(cmd))

		p.SetArgs([]string{"server"})
		require.NoError(t, p.Execute(context.Background()))
		assert.Equal(t, 8080, cfg.Port)
	})

	t.Run("nested typed commands", func(t *testing.T) {
		type startConfig struct {
			Detach bool `long:"detach"`
		}
		p := New("app")
		_, server, err := BindCommand("server", "", func(context.Context, *Command, *serverConfig, []string) error { return nil })
		require.NoError(t, err)
		require.NoError(t, p.AddCommand(server))

		var started bool
		startCfg, start, err := BindCommand("start", "", func(_ context.Context, _ *Command, cfg *startConfig, _ []string) error {
			started = cfg.Detach
			return nil
		})
		require.NoError(t, err)
		require.NotNil(t, startCfg)
		require.NoError(t, server.AddCommand(start))

		p.SetArgs([]string{"server", "start", "--detach"})
		require.NoError(t, p.Execute(context.Background()))
		assert.True(t, started)
	})

	t.Run("propagates handler error", func(t *testing.T) {
		sentinel := errors.New("boom")
		p := New("app")
		_, cmd, err := BindCommand("fail", "", func(context.Context, *Command, *serverConfig, []string) error { return sentinel })
		require.NoError(t, err)
		require.NoError(t, p.AddCommand(cmd))

		p.SetArgs([]string{"fail"})
		require.ErrorIs(t, p.Execute(context.Background()), sentinel)
	})

	t.Run("nil handler leaves command without run", func(t *testing.T) {
		_, cmd, err := BindCommand("group", "", (func(context.Context, *Command, *serverConfig, []string) error)(nil))
		require.NoError(t, err)
		assert.Nil(t, cmd.run)
	})

	t.Run("non-struct type errors", func(t *testing.T) {
		_, _, err := BindCommand("cmd", "", func(context.Context, *Command, *int, []string) error { return nil })
		require.Error(t, err)
	})
}

func TestBindRoot(t *testing.T) {
	t.Run("typed root handler", func(t *testing.T) {
		type rootConfig struct {
			Verbose bool `short:"v" long:"verbose"`
		}
		p := New("app")
		var ranVerbose bool
		var ranArgs []string
		cfg, err := BindRoot(p, func(_ context.Context, _ *Command, c *rootConfig, args []string) error {
			ranVerbose = c.Verbose
			ranArgs = args
			return nil
		})
		require.NoError(t, err)
		require.NotNil(t, cfg)

		p.SetArgs([]string{"-v", "file1", "file2"})
		require.NoError(t, p.Execute(context.Background()))
		assert.True(t, ranVerbose)
		assert.Equal(t, []string{"file1", "file2"}, ranArgs)
	})

	t.Run("root falls back to subcommand dispatch", func(t *testing.T) {
		type rootConfig struct {
			Verbose bool `short:"v" long:"verbose"`
		}
		type subConfig struct {
			Flag bool `long:"flag"`
		}
		p := New("app")
		rootCfg, err := BindRoot(p, func(context.Context, *Command, *rootConfig, []string) error {
			t.Fatal("root handler must not run when a subcommand is selected")
			return nil
		})
		require.NoError(t, err)

		var subRan bool
		_, sub, err := BindCommand("sub", "", func(context.Context, *Command, *subConfig, []string) error {
			subRan = true
			return nil
		})
		require.NoError(t, err)
		require.NoError(t, p.AddCommand(sub))

		p.SetArgs([]string{"-v", "sub"})
		require.NoError(t, p.Execute(context.Background()))
		assert.True(t, subRan)
		assert.True(t, rootCfg.Verbose)
	})

	t.Run("non-struct type errors", func(t *testing.T) {
		p := New("app")
		_, err := BindRoot(p, func(context.Context, *Command, *string, []string) error { return nil })
		require.Error(t, err)
	})

	t.Run("nil handler leaves root without run", func(t *testing.T) {
		type rootConfig struct {
			Name string `long:"name"`
		}
		p := New("app")
		var buf bytes.Buffer
		p.SetOutput(&buf)
		_, err := BindRoot(p, (func(context.Context, *Command, *rootConfig, []string) error)(nil))
		require.NoError(t, err)

		p.SetArgs([]string{})
		require.NoError(t, p.Execute(context.Background()))
		assert.Contains(t, buf.String(), "app")
	})
}
