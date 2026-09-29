package xflags

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelp(t *testing.T) {
	t.Run("renders groups and options", func(t *testing.T) {
		var opts struct {
			Verbose int    `short:"v" long:"verbose" description:"increase verbosity"`
			Name    string `long:"name" default:"app" description:"service name"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("Application Options", &opts))

		out := p.Help()
		assert.Contains(t, out, "Usage:")
		assert.Contains(t, out, "Application Options:")
		assert.Contains(t, out, "-v, --verbose")
		assert.Contains(t, out, "increase verbosity")
		assert.Contains(t, out, "(default: app)")
		assert.Contains(t, out, "-h, --help")
	})

	t.Run("version line only when version set", func(t *testing.T) {
		p := New("app")
		assert.NotContains(t, p.Help(), "--version")

		p.SetVersion("1.0.0")
		assert.Contains(t, p.Help(), "--version")
	})

	t.Run("version short only when v free", func(t *testing.T) {
		var opts struct {
			Verbose int `short:"v" long:"verbose"`
		}
		p := New("app")
		p.SetVersion("1.0.0")
		require.NoError(t, p.AddGroup("", &opts))
		out := p.Help()
		assert.Contains(t, out, "    --version")
		assert.NotContains(t, out, "-v, --version")
	})

	t.Run("lists commands", func(t *testing.T) {
		p := New("app")
		sub, err := p.AddCommand("server", nil)
		require.NoError(t, err)
		sub.SetDescription("run the server")
		assert.Contains(t, p.Help(), "server")
		assert.Contains(t, p.Help(), "run the server")
	})
}

func TestHelpFlag(t *testing.T) {
	t.Run("long help stops parsing", func(t *testing.T) {
		var opts struct {
			Name string `long:"name" required:"true"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		// --help should short-circuit before the required check fails.
		require.NoError(t, p.Parse([]string{"--help"}))
	})

	t.Run("short help stops parsing", func(t *testing.T) {
		var opts struct {
			Name string `long:"name" required:"true"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"-h"}))
	})
}

func TestVersionFlag(t *testing.T) {
	t.Run("version stops parsing", func(t *testing.T) {
		p := New("app")
		p.SetVersion("1.2.3")
		require.NoError(t, p.Parse([]string{"--version"}))
	})

	t.Run("no version flag when unset", func(t *testing.T) {
		p := New("app")
		require.Error(t, p.Parse([]string{"--version"}))
	})

	t.Run("short v version when free", func(t *testing.T) {
		p := New("app")
		p.SetVersion("1.2.3")
		require.NoError(t, p.Parse([]string{"-v"}))
	})
}

func TestHiddenAndValueName(t *testing.T) {
	t.Run("hidden option omitted from help", func(t *testing.T) {
		var o struct {
			Secret string `long:"secret" hidden:"true"`
			Public string `long:"public"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		out := p.Help()
		assert.NotContains(t, out, "--secret")
		assert.Contains(t, out, "--public")
	})

	t.Run("hidden option still parses", func(t *testing.T) {
		var o struct {
			Secret string `long:"secret" hidden:"true"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--secret=x"}))
		assert.Equal(t, "x", o.Secret)
	})

	t.Run("value-name shown in help", func(t *testing.T) {
		var o struct {
			File string `long:"file" value-name:"PATH"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		assert.Contains(t, p.Help(), "--file PATH")
	})

	t.Run("empty group after hiding is skipped", func(t *testing.T) {
		var o struct {
			Secret string `long:"secret" hidden:"true"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("Hidden Group", &o))
		assert.NotContains(t, p.Help(), "Hidden Group")
	})
}

func captureParseOutput(t *testing.T, p *Parser, args ...string) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old; _ = r.Close(); _ = w.Close() })
	parseErr := p.Parse(args)
	os.Stdout = old
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, parseErr)
	return string(out)
}

func TestHelpCommandAndParseOutcome(t *testing.T) {
	p := New("app")
	p.SetVersion("1.0")
	var o struct {
		Config string `long:"config"`
		Token  string `long:"token" required:"true"`
	}
	require.NoError(t, p.AddGroup("", &o))
	validations := 0
	require.NoError(t, p.Validate("config", func(any) error { validations++; return nil }))
	sub, err := p.AddCommand("server", nil)
	require.NoError(t, err)
	sub.SetDescription("run the server")
	assert.Nil(t, p.SelectedCommand())
	for _, args := range [][]string{
		{"--config", "file", "server", "--help"},
		{"--config=server", "server", "-h"},
		{"--", "server", "--help"},
	} {
		out := captureParseOutput(t, p, args...)
		assert.Contains(t, out, "Usage:\n  app server\n")
		assert.Contains(t, out, "run the server")
		assert.Same(t, sub, p.SelectedCommand())
		assert.True(t, p.Handled())
	}
	out := captureParseOutput(t, p, "--help", "server")
	assert.Contains(t, out, "app [options] <command>")
	assert.Same(t, p.Command, p.SelectedCommand())
	assert.Contains(t, captureParseOutput(t, p, "server", "--version"), "app version 1.0")
	assert.Zero(t, validations)
	require.NoError(t, p.Parse([]string{"--token=x", "server"}))
	assert.False(t, p.Handled())
	assert.Same(t, sub, p.SelectedCommand())
	assert.Equal(t, "server", sub.Name())
	assert.Equal(t, 1, validations)
	require.Error(t, p.Parse([]string{"--unknown"}))
	assert.False(t, p.Handled())
}

func TestHelpBuiltinOverrides(t *testing.T) {
	p := New("app")
	p.SetVersion("1")
	var host, version string
	require.NoError(t, p.StringVar(&host, "host", "h", "", "hostname"))
	require.NoError(t, p.StringVar(&version, "version", "", "", "custom version"))
	out := p.Help()
	assert.NotContains(t, out, "-h, --help")
	assert.NotContains(t, out, "-v, --version")
	assert.Contains(t, out, "--help  show this help message")
	assert.Contains(t, out, "-v  show version information")
}

func TestSetOutput(t *testing.T) {
	t.Run("captures help", func(t *testing.T) {
		var buf bytes.Buffer
		p := New("app")
		p.SetOutput(&buf)
		require.NoError(t, p.Parse([]string{"--help"}))
		assert.True(t, p.Handled())
		assert.Contains(t, buf.String(), "Usage:")
		assert.Contains(t, buf.String(), "app")
	})

	t.Run("captures version", func(t *testing.T) {
		var buf bytes.Buffer
		p := New("app")
		p.SetVersion("1.2.3")
		p.SetOutput(&buf)
		require.NoError(t, p.Parse([]string{"--version"}))
		assert.Equal(t, "app version 1.2.3\n", buf.String())
	})
}
