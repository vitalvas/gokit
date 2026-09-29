package xflags

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	t.Run("string int bool", func(t *testing.T) {
		var (
			name  string
			count int
			debug bool
		)
		p := New("app")
		require.NoError(t, p.StringVar(&name, "name", "n", "def", "the name"))
		require.NoError(t, p.IntVar(&count, "count", "c", 0, "the count"))
		require.NoError(t, p.BoolVar(&debug, "debug", "d", false, "debug mode"))

		require.NoError(t, p.Parse([]string{"-n", "alice", "--count=3", "-d"}))
		assert.Equal(t, "alice", name)
		assert.Equal(t, 3, count)
		assert.True(t, debug)
	})

	t.Run("string var default", func(t *testing.T) {
		var name string
		p := New("app")
		require.NoError(t, p.StringVar(&name, "name", "", "fallback", ""))
		require.NoError(t, p.Parse(nil))
		assert.Equal(t, "fallback", name)
	})

	t.Run("count var", func(t *testing.T) {
		var v int
		p := New("app")
		require.NoError(t, p.CountVar(&v, "verbose", "v", "verbosity"))
		require.NoError(t, p.Parse([]string{"-vvvv"}))
		assert.Equal(t, 4, v)
	})

	t.Run("slice var", func(t *testing.T) {
		var tags []string
		p := New("app")
		require.NoError(t, p.StringSliceVar(&tags, "tag", "t", "tags"))
		require.NoError(t, p.Parse([]string{"-t", "a", "--tag=b"}))
		assert.Equal(t, []string{"a", "b"}, tags)
	})

	t.Run("map var", func(t *testing.T) {
		var labels map[string]string
		p := New("app")
		require.NoError(t, p.StringMapVar(&labels, "label", "l", "labels"))
		require.NoError(t, p.Parse([]string{"--label=a=1", "--label=b=2"}))
		assert.Equal(t, map[string]string{"a": "1", "b": "2"}, labels)
	})

	t.Run("func var", func(t *testing.T) {
		seen := ""
		p := New("app")
		require.NoError(t, p.Func("config", "", "load config", func(s string) error {
			seen = s
			return nil
		}))
		require.NoError(t, p.Parse([]string{"--config=x.yaml"}))
		assert.Equal(t, "x.yaml", seen)
	})

	t.Run("func var error", func(t *testing.T) {
		p := New("app")
		require.NoError(t, p.Func("config", "", "", func(string) error {
			return errors.New("bad")
		}))
		require.Error(t, p.Parse([]string{"--config=x"}))
	})

	t.Run("nil pointer errors", func(t *testing.T) {
		p := New("app")
		require.Error(t, p.StringVar(nil, "name", "", "", ""))
	})

	t.Run("bool var default true", func(t *testing.T) {
		var flag bool
		p := New("app")
		require.NoError(t, p.BoolVar(&flag, "on", "", true, ""))
		require.NoError(t, p.Parse(nil))
		assert.True(t, flag)
	})
}

func TestBuilderFeatureSetters(t *testing.T) {
	t.Run("set hidden", func(t *testing.T) {
		var s string
		p := New("app")
		require.NoError(t, p.StringVar(&s, "secret", "", "", ""))
		require.NoError(t, p.SetHidden("secret"))
		assert.NotContains(t, p.Help(), "--secret")
	})

	t.Run("set value name", func(t *testing.T) {
		var s string
		p := New("app")
		require.NoError(t, p.StringVar(&s, "file", "", "", ""))
		require.NoError(t, p.SetValueName("file", "PATH"))
		assert.Contains(t, p.Help(), "--file PATH")
	})

	t.Run("set base", func(t *testing.T) {
		var n int
		p := New("app")
		require.NoError(t, p.IntVar(&n, "mask", "", 0, ""))
		require.NoError(t, p.SetBase("mask", 16))
		require.NoError(t, p.Parse([]string{"--mask=ff"}))
		assert.Equal(t, 255, n)
	})

	t.Run("set optional value", func(t *testing.T) {
		var s string
		p := New("app")
		require.NoError(t, p.StringVar(&s, "log", "", "", ""))
		require.NoError(t, p.SetOptionalValue("log", "info"))
		require.NoError(t, p.Parse([]string{"--log"}))
		assert.Equal(t, "info", s)
	})

	t.Run("setters error on unknown option", func(t *testing.T) {
		p := New("app")
		require.Error(t, p.SetHidden("nope"))
		require.Error(t, p.SetValueName("nope", "X"))
		require.Error(t, p.SetBase("nope", 16))
		require.Error(t, p.SetOptionalValue("nope", "x"))
	})
}

func TestTypedBuilderDefaults(t *testing.T) {
	p := New("app")
	b, s, n := true, "old", 99
	require.NoError(t, p.BoolVar(&b, "b", "", false, ""))
	require.NoError(t, p.StringVar(&s, "s", "", "", ""))
	require.NoError(t, p.IntVar(&n, "n", "", 10, ""))
	require.NoError(t, p.SetBase("n", 16))
	require.NoError(t, p.Parse(nil))
	assert.False(t, b)
	assert.Empty(t, s)
	assert.Equal(t, 10, n)
	require.NoError(t, p.Parse([]string{"--n=ff"}))
	assert.Equal(t, 255, n)

	o := struct {
		S string `long:"s" default:""`
	}{S: "old"}
	p = New("app")
	require.NoError(t, p.AddGroup("", &o))
	require.NoError(t, p.Parse(nil))
	assert.Empty(t, o.S)
}
