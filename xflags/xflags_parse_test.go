package xflags

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLongOptions(t *testing.T) {
	t.Run("equals and space forms", func(t *testing.T) {
		var opts struct {
			Name string `long:"name"`
			Host string `long:"host"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"--name=alice", "--host", "localhost"}))
		assert.Equal(t, "alice", opts.Name)
		assert.Equal(t, "localhost", opts.Host)
	})

	t.Run("missing value errors", func(t *testing.T) {
		var opts struct {
			Name string `long:"name"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.Error(t, p.Parse([]string{"--name"}))
	})

	t.Run("unknown option errors", func(t *testing.T) {
		p := New("app")
		require.Error(t, p.Parse([]string{"--nope"}))
	})
}

func TestParseBool(t *testing.T) {
	var opts struct {
		Debug bool `long:"debug"`
		Quiet bool `long:"quiet"`
	}
	p := New("app")
	require.NoError(t, p.AddGroup("", &opts))
	require.NoError(t, p.Parse([]string{"--debug", "--quiet=false"}))
	assert.True(t, opts.Debug)
	assert.False(t, opts.Quiet)
}

func TestParseShortClusters(t *testing.T) {
	t.Run("combined bool flags", func(t *testing.T) {
		var opts struct {
			A bool `short:"a"`
			B bool `short:"b"`
			C bool `short:"c"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"-abc"}))
		assert.True(t, opts.A)
		assert.True(t, opts.B)
		assert.True(t, opts.C)
	})

	t.Run("attached value", func(t *testing.T) {
		var opts struct {
			Out string `short:"o"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"-ofile.txt"}))
		assert.Equal(t, "file.txt", opts.Out)
	})

	t.Run("value from next arg", func(t *testing.T) {
		var opts struct {
			Out string `short:"o"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"-o", "file.txt"}))
		assert.Equal(t, "file.txt", opts.Out)
	})

	t.Run("unknown short errors", func(t *testing.T) {
		p := New("app")
		require.Error(t, p.Parse([]string{"-z"}))
	})
}

func TestPrimitiveTypes(t *testing.T) {
	var opts struct {
		S   string        `long:"s"`
		I   int           `long:"i"`
		I8  int8          `long:"i8"`
		U   uint          `long:"u"`
		F   float64       `long:"f"`
		B   bool          `long:"b"`
		Dur time.Duration `long:"dur"`
	}
	p := New("app")
	require.NoError(t, p.AddGroup("", &opts))
	require.NoError(t, p.Parse([]string{
		"--s=text", "--i=-5", "--i8=127", "--u=9", "--f=1.5", "--b", "--dur=30s",
	}))
	assert.Equal(t, "text", opts.S)
	assert.Equal(t, -5, opts.I)
	assert.Equal(t, int8(127), opts.I8)
	assert.Equal(t, uint(9), opts.U)
	assert.InDelta(t, 1.5, opts.F, 0)
	assert.True(t, opts.B)
	assert.Equal(t, 30*time.Second, opts.Dur)
}

func TestInvalidValues(t *testing.T) {
	cases := map[string]struct {
		build func() *Parser
		args  []string
	}{
		"int": {
			func() *Parser {
				var o struct {
					I int `long:"i"`
				}
				p := New("app")
				_ = p.AddGroup("", &o)
				return p
			},
			[]string{"--i=abc"},
		},
		"uint": {
			func() *Parser {
				var o struct {
					U uint `long:"u"`
				}
				p := New("app")
				_ = p.AddGroup("", &o)
				return p
			},
			[]string{"--u=-1"},
		},
		"float": {
			func() *Parser {
				var o struct {
					F float64 `long:"f"`
				}
				p := New("app")
				_ = p.AddGroup("", &o)
				return p
			},
			[]string{"--f=xx"},
		},
		"bool": {
			func() *Parser {
				var o struct {
					B bool `long:"b"`
				}
				p := New("app")
				_ = p.AddGroup("", &o)
				return p
			},
			[]string{"--b=maybe"},
		},
		"duration": {
			func() *Parser {
				var o struct {
					D time.Duration `long:"d"`
				}
				p := New("app")
				_ = p.AddGroup("", &o)
				return p
			},
			[]string{"--d=xx"},
		},
		"int8 overflow": {
			func() *Parser {
				var o struct {
					I int8 `long:"i"`
				}
				p := New("app")
				_ = p.AddGroup("", &o)
				return p
			},
			[]string{"--i=200"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, tc.build().Parse(tc.args))
		})
	}
}

func TestChoicePolicies(t *testing.T) {
	t.Run("append", func(t *testing.T) {
		var opts struct {
			Tags []string `long:"tag" choice:"append"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"--tag=a", "--tag=b", "--tag=c"}))
		assert.Equal(t, []string{"a", "b", "c"}, opts.Tags)
	})

	t.Run("count", func(t *testing.T) {
		var opts struct {
			V int `short:"v" choice:"count"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"-vvv"}))
		assert.Equal(t, 3, opts.V)
	})

	t.Run("last", func(t *testing.T) {
		var opts struct {
			Name string `long:"name" choice:"last"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"--name=a", "--name=b"}))
		assert.Equal(t, "b", opts.Name)
	})

	t.Run("default append for slices", func(t *testing.T) {
		var opts struct {
			Tags []string `long:"tag"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"--tag=a", "--tag=b"}))
		assert.Equal(t, []string{"a", "b"}, opts.Tags)
	})

	t.Run("unknown choice errors", func(t *testing.T) {
		var opts struct {
			X string `long:"x" choice:"bogus"`
		}
		p := New("app")
		require.Error(t, p.AddGroup("", &opts))
	})
}

func TestMaps(t *testing.T) {
	t.Run("collects pairs", func(t *testing.T) {
		var opts struct {
			Labels map[string]string `long:"label"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"--label=env=prod", "--label", "tier=web"}))
		assert.Equal(t, map[string]string{"env": "prod", "tier": "web"}, opts.Labels)
	})

	t.Run("int values", func(t *testing.T) {
		var opts struct {
			Ports map[string]int `long:"port"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"--port=http=80"}))
		assert.Equal(t, map[string]int{"http": 80}, opts.Ports)
	})

	t.Run("missing equals errors", func(t *testing.T) {
		var opts struct {
			Labels map[string]string `long:"label"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.Error(t, p.Parse([]string{"--label=novalue"}))
	})

	t.Run("empty key errors", func(t *testing.T) {
		var opts struct {
			Labels map[string]string `long:"label"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.Error(t, p.Parse([]string{"--label==v"}))
	})
}

func TestDefaults(t *testing.T) {
	var opts struct {
		Port int    `long:"port" default:"8080"`
		Name string `long:"name" default:"app"`
	}
	p := New("app")
	require.NoError(t, p.AddGroup("", &opts))
	require.NoError(t, p.Parse(nil))
	assert.Equal(t, 8080, opts.Port)
	assert.Equal(t, "app", opts.Name)
}

func TestRequired(t *testing.T) {
	t.Run("missing errors", func(t *testing.T) {
		var opts struct {
			Name string `long:"name" required:"true"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.Error(t, p.Parse(nil))
	})

	t.Run("satisfied by cli", func(t *testing.T) {
		var opts struct {
			Name string `long:"name" required:"true"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"--name=x"}))
	})

	t.Run("default does not satisfy required", func(t *testing.T) {
		var opts struct {
			Name string `long:"name" required:"true" default:"d"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.Error(t, p.Parse(nil))
	})

	t.Run("env satisfies required", func(t *testing.T) {
		t.Setenv("XF_REQ", "x")
		var opts struct {
			Name string `long:"name" required:"true" env:"XF_REQ"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse(nil))
	})
}

func TestValidationCallback(t *testing.T) {
	t.Run("builder validate rejects", func(t *testing.T) {
		var opts struct {
			Port int `long:"port"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Validate("port", func(v any) error {
			if v.(int) > 65535 {
				return errors.New("out of range")
			}
			return nil
		}))
		require.Error(t, p.Parse([]string{"--port=99999"}))
		require.NoError(t, p.Parse([]string{"--port=8080"}))
	})

	t.Run("struct func field callback", func(t *testing.T) {
		called := ""
		opts := struct {
			OnConfig func(string) error `long:"config"`
		}{
			OnConfig: func(s string) error { called = s; return nil },
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"--config=file.yaml"}))
		assert.Equal(t, "file.yaml", called)
	})

	t.Run("func field returning error fails parse", func(t *testing.T) {
		opts := struct {
			OnConfig func(string) error `long:"config"`
		}{
			OnConfig: func(string) error { return errors.New("bad") },
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.Error(t, p.Parse([]string{"--config=x"}))
	})

	t.Run("zero-arg func callback", func(t *testing.T) {
		n := 0
		opts := struct {
			OnDebug func() error `long:"debug"`
		}{
			OnDebug: func() error { n++; return nil },
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"--debug"}))
		assert.Equal(t, 1, n)
	})

	t.Run("store and validate paired fields", func(t *testing.T) {
		var seen int
		opts := struct {
			Port   int             `long:"port"`
			PortFn func(int) error `long:"port"`
		}{
			PortFn: func(v int) error { seen = v; return nil },
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &opts))
		require.NoError(t, p.Parse([]string{"--port=8080"}))
		assert.Equal(t, 8080, opts.Port)
		assert.Equal(t, 8080, seen)
	})
}

func TestDoubleDash(t *testing.T) {
	var opts struct {
		Name string `long:"name"`
	}
	p := New("app")
	require.NoError(t, p.AddGroup("", &opts))
	require.NoError(t, p.Parse([]string{"--name=x", "--"}))
	assert.Equal(t, "x", opts.Name)
}

func TestSubcommandsBuilder(t *testing.T) {
	t.Run("dispatches to subcommand", func(t *testing.T) {
		var serverOpts struct {
			Port int `long:"port" default:"8080"`
		}
		p := New("app")
		_, err := p.AddCommand("server", &serverOpts)
		require.NoError(t, err)

		require.NoError(t, p.Parse([]string{"server", "--port=9090"}))
		assert.Equal(t, 9090, serverOpts.Port)
	})

	t.Run("nested subcommands", func(t *testing.T) {
		var startOpts struct {
			Detach bool `long:"detach"`
		}
		p := New("app")
		server, err := p.AddCommand("server", nil)
		require.NoError(t, err)
		_, err = server.AddCommand("start", &startOpts)
		require.NoError(t, err)

		require.NoError(t, p.Parse([]string{"server", "start", "--detach"}))
		assert.True(t, startOpts.Detach)
	})

	t.Run("root options before subcommand", func(t *testing.T) {
		var rootOpts struct {
			Verbose bool `short:"v" long:"verbose"`
		}
		var serverOpts struct {
			Port int `long:"port"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &rootOpts))
		_, err := p.AddCommand("server", &serverOpts)
		require.NoError(t, err)

		require.NoError(t, p.Parse([]string{"-v", "server", "--port=80"}))
		assert.True(t, rootOpts.Verbose)
		assert.Equal(t, 80, serverOpts.Port)
	})

	t.Run("duplicate command errors", func(t *testing.T) {
		p := New("app")
		_, err := p.AddCommand("dup", nil)
		require.NoError(t, err)
		_, err = p.AddCommand("dup", nil)
		require.Error(t, err)
	})

	t.Run("unexpected positional errors", func(t *testing.T) {
		p := New("app")
		require.Error(t, p.Parse([]string{"bogus"}))
	})
}

func TestSubcommandsTag(t *testing.T) {
	type serverOptions struct {
		Port int `long:"port" default:"8080"`
	}
	type clientOptions struct {
		Addr string `long:"addr"`
	}

	t.Run("command tag declares subcommands", func(t *testing.T) {
		var root struct {
			Server serverOptions `command:"server" description:"run the server"`
			Client clientOptions `command:"client"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &root))

		require.NoError(t, p.Parse([]string{"server", "--port=9090"}))
		assert.Equal(t, 9090, root.Server.Port)
		assert.Equal(t, "run the server", p.byCommand["server"].description)
	})

	t.Run("client subcommand", func(t *testing.T) {
		var root struct {
			Server serverOptions `command:"server"`
			Client clientOptions `command:"client"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &root))
		require.NoError(t, p.Parse([]string{"client", "--addr=127.0.0.1"}))
		assert.Equal(t, "127.0.0.1", root.Client.Addr)
	})
}

func TestShortEquals(t *testing.T) {
	t.Run("valued short with equals", func(t *testing.T) {
		var o struct {
			Out string `short:"o"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"-o=value"}))
		assert.Equal(t, "value", o.Out)
	})

	t.Run("bool short with equals disables", func(t *testing.T) {
		var o struct {
			Debug bool `short:"d"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"-d=false"}))
		assert.False(t, o.Debug)
	})

	t.Run("clustered bool then valued with equals", func(t *testing.T) {
		var o struct {
			A   bool   `short:"a"`
			Out string `short:"o"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"-ao=file"}))
		assert.True(t, o.A)
		assert.Equal(t, "file", o.Out)
	})
}

func TestOptionalValue(t *testing.T) {
	t.Run("absent uses optional-value", func(t *testing.T) {
		var o struct {
			Log string `long:"log" optional-value:"info"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--log"}))
		assert.Equal(t, "info", o.Log)
	})

	t.Run("inline value wins", func(t *testing.T) {
		var o struct {
			Log string `long:"log" optional-value:"info"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--log=debug"}))
		assert.Equal(t, "debug", o.Log)
	})

	t.Run("next non-flag token is consumed", func(t *testing.T) {
		var o struct {
			Log string `long:"log" optional-value:"info"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--log", "trace"}))
		assert.Equal(t, "trace", o.Log)
	})

	t.Run("following flag is not consumed", func(t *testing.T) {
		var o struct {
			Log   string `long:"log" optional-value:"info"`
			Debug bool   `long:"debug"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--log", "--debug"}))
		assert.Equal(t, "info", o.Log)
		assert.True(t, o.Debug)
	})

	t.Run("optional tag without value", func(t *testing.T) {
		var o struct {
			Log string `long:"log" optional:"true"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--log"}))
		assert.Empty(t, o.Log)
	})

	t.Run("negative number is consumed as value", func(t *testing.T) {
		var o struct {
			Offset int `long:"offset" optional-value:"0"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--offset", "-5"}))
		assert.Equal(t, -5, o.Offset)
	})
}

func TestUserHelpVersionNotShadowed(t *testing.T) {
	t.Run("user long help option reachable", func(t *testing.T) {
		var o struct {
			Help string `long:"help"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--help=topic"}))
		assert.Equal(t, "topic", o.Help)
	})

	t.Run("user short h option reachable", func(t *testing.T) {
		var o struct {
			Host string `short:"h" long:"host"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"-h", "myhost"}))
		assert.Equal(t, "myhost", o.Host)
	})

	t.Run("user version option reachable despite SetVersion", func(t *testing.T) {
		var o struct {
			Version string `long:"version"`
		}
		p := New("app")
		p.SetVersion("9.9")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--version=1.2"}))
		assert.Equal(t, "1.2", o.Version)
	})

	t.Run("builtin help still works without user option", func(t *testing.T) {
		p := New("app")
		require.NoError(t, p.Parse([]string{"--help"}))
	})
}

func TestPositionalsAfterDoubleDash(t *testing.T) {
	t.Run("unknown positional after -- errors", func(t *testing.T) {
		p := New("app")
		_, err := p.AddCommand("server", nil)
		require.NoError(t, err)
		require.Error(t, p.Parse([]string{"--", "notacommand"}))
	})

	t.Run("subcommand after -- still dispatches", func(t *testing.T) {
		var serverOpts struct {
			Port int `long:"port" default:"8080"`
		}
		p := New("app")
		_, err := p.AddCommand("server", &serverOpts)
		require.NoError(t, err)
		require.NoError(t, p.Parse([]string{"--", "server", "--port=7070"}))
		assert.Equal(t, 7070, serverOpts.Port)
	})
}

func TestLooksLikeFlag(t *testing.T) {
	cases := map[string]bool{
		"--flag": true,
		"-f":     true,
		"--":     true,
		"-5":     false,
		"-1.2":   false,
		"-":      false,
		"value":  false,
		"":       false,
	}
	for token, want := range cases {
		t.Run(token, func(t *testing.T) {
			assert.Equal(t, want, looksLikeFlag(token))
		})
	}
}

func TestResolvedPairedCallbacks(t *testing.T) {
	calls := 0
	o := struct {
		N     int             `long:"n"`
		Check func(int) error `long:"n"`
	}{Check: func(n int) error {
		calls++
		if n > 10 {
			return errors.New("too large")
		}
		return nil
	}}
	p := New("app")
	require.NoError(t, p.AddGroup("", &o))
	require.NoError(t, p.Parse([]string{"--n=20", "--n=5"}))
	assert.Equal(t, 5, o.N)
	assert.Equal(t, 1, calls)
	require.NoError(t, p.Parse(nil))
	assert.Equal(t, 2, calls, "zero values must also be validated")
	require.Error(t, p.Parse([]string{"--n=20"}))

	var seenCount int
	var seenTags []string
	var seenMap map[string]int
	containers := struct {
		N      int                        `long:"n" choice:"count"`
		NFn    func(int) error            `long:"n"`
		Tags   []string                   `long:"tag"`
		TagsFn func([]string) error       `long:"tag"`
		Map    map[string]int             `long:"map"`
		MapFn  func(map[string]int) error `long:"map"`
	}{NFn: func(n int) error { seenCount = n; return nil }, TagsFn: func(v []string) error { seenTags = v; return nil }, MapFn: func(v map[string]int) error { seenMap = v; return nil }}
	p = New("app")
	require.NoError(t, p.AddGroup("", &containers))
	require.NoError(t, p.Parse([]string{"--n", "--n", "--tag=a", "--tag=b", "--map=x=1", "--map=y=2"}))
	assert.Equal(t, 2, seenCount)
	assert.Equal(t, []string{"a", "b"}, seenTags)
	assert.Equal(t, map[string]int{"x": 1, "y": 2}, seenMap)

	var invalid struct {
		N  int                `long:"n"`
		Fn func(string) error `long:"n"`
	}
	require.Error(t, New("app").AddGroup("", &invalid))
}

func TestStandaloneCallbackOccurrences(t *testing.T) {
	var seen []string
	p := New("app")
	require.NoError(t, p.Func("f", "", "", func(s string) error { seen = append(seen, s); return nil }))
	require.NoError(t, p.Parse([]string{"--f=a", "--f=b"}))
	assert.Equal(t, []string{"a", "b"}, seen)
}

func TestParseResetsSourcesAndDestinations(t *testing.T) {
	t.Setenv("XF_REUSE", "first")
	o := struct {
		N      int               `long:"n" required:"true"`
		Name   string            `long:"name" env:"XF_REUSE" default:"fallback"`
		Tags   []string          `long:"tag"`
		Labels map[string]string `long:"label"`
		Ptr    *int              `long:"ptr"`
	}{Tags: []string{"initial"}, Labels: map[string]string{"initial": "yes"}}
	p := New("app")
	require.NoError(t, p.AddGroup("", &o))
	require.Error(t, p.Parse([]string{"--n=bad"}))
	require.Error(t, p.Parse(nil))
	require.NoError(t, p.Parse([]string{"--n=1", "--tag=a", "--label=x=y", "--ptr=9"}))
	assert.Equal(t, "first", o.Name)
	t.Setenv("XF_REUSE", "second")
	require.NoError(t, p.Parse([]string{"--n=2"}))
	assert.Equal(t, "second", o.Name)
	assert.Equal(t, []string{"initial"}, o.Tags)
	assert.Equal(t, map[string]string{"initial": "yes"}, o.Labels)
	assert.Nil(t, o.Ptr)
	t.Setenv("XF_REUSE", "")
	require.NoError(t, p.Parse([]string{"--n=3"}))
	assert.Equal(t, "fallback", o.Name)
	require.Error(t, p.Parse(nil))
}

func TestOptionalZeroValues(t *testing.T) {
	var o struct {
		N     int             `long:"n" optional:"true"`
		F     float64         `long:"f" optional:"true"`
		D     time.Duration   `long:"d" optional:"true"`
		Slice []int           `long:"slice" optional:"true"`
		Fn    func(int) error `long:"fn" optional:"true"`
	}
	seen := -1
	o.N, o.F, o.D = 9, 9, time.Second
	o.Fn = func(n int) error { seen = n; return nil }
	p := New("app")
	require.NoError(t, p.AddGroup("", &o))
	require.NoError(t, p.Parse([]string{"--n", "--f", "--d", "--slice", "--fn"}))
	assert.Zero(t, o.N)
	assert.Zero(t, o.F)
	assert.Zero(t, o.D)
	assert.Equal(t, []int{0}, o.Slice)
	assert.Zero(t, seen)
	require.Error(t, p.Parse([]string{"--n="}), "explicit empty numeric values are invalid")
}
