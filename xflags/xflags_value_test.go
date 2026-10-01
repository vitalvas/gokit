package xflags

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegerBase(t *testing.T) {
	t.Run("hex via base tag", func(t *testing.T) {
		var o struct {
			Mask int `long:"mask" base:"16"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--mask=ff"}))
		assert.Equal(t, 255, o.Mask)
	})

	t.Run("base 0 detects prefix", func(t *testing.T) {
		var o struct {
			A int `long:"a" base:"0"`
			B int `long:"b" base:"0"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--a=0x10", "--b=0o17"}))
		assert.Equal(t, 16, o.A)
		assert.Equal(t, 15, o.B)
	})

	t.Run("invalid base tag errors", func(t *testing.T) {
		var o struct {
			A int `long:"a" base:"xx"`
		}
		p := New("app")
		require.Error(t, p.AddGroup("", &o))
	})

	t.Run("default base 10", func(t *testing.T) {
		var o struct {
			A int `long:"a"`
		}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NoError(t, p.Parse([]string{"--a=42"}))
		assert.Equal(t, 42, o.A)
	})
}

func TestNilCallbackNoPanic(t *testing.T) {
	t.Run("nil func field is a no-op", func(t *testing.T) {
		o := struct {
			Fn func(string) error `long:"fn"`
		}{}
		p := New("app")
		require.NoError(t, p.AddGroup("", &o))
		require.NotPanics(t, func() {
			require.NoError(t, p.Parse([]string{"--fn=x"}))
		})
	})
}

func TestCounterBounds(t *testing.T) {
	for _, tc := range []struct {
		name     string
		dst      any
		max      string
		overflow string
	}{
		{"int8", new(int8), "127", "128"},
		{"int64", new(int64), "9223372036854775807", "9223372036854775808"},
		{"uint8", new(uint8), "255", "256"},
		{"uint64", new(uint64), "18446744073709551615", "18446744073709551616"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := New("app")
			require.NoError(t, p.register(optionSpec{ptr: tc.dst, long: "n", short: "n", choice: choiceCount}))
			require.NoError(t, p.Parse([]string{"-nn"}))
			assert.EqualValues(t, 2, reflect.ValueOf(tc.dst).Elem().Interface())
			require.NoError(t, p.Parse([]string{fmt.Sprintf("--n=%s", tc.max)}))
			maximum := reflect.ValueOf(tc.dst).Elem().Interface()
			require.Error(t, p.Parse([]string{fmt.Sprintf("--n=%s", tc.max), "--n"}))
			assert.Equal(t, maximum, reflect.ValueOf(tc.dst).Elem().Interface())
			require.Error(t, p.Parse([]string{fmt.Sprintf("--n=%s", tc.overflow)}))
		})
	}
	var invalid struct {
		N string `long:"n" choice:"count"`
	}
	require.Error(t, New("app").AddGroup("", &invalid))
}

func TestNamedMapKey(t *testing.T) {
	type key string
	var o struct {
		Labels map[key]int `long:"label"`
	}
	p := New("app")
	require.NoError(t, p.AddGroup("", &o))
	require.NoError(t, p.Parse([]string{"--label=http=80", "--label=http=8080"}))
	assert.Equal(t, map[key]int{"http": 8080}, o.Labels)
}
