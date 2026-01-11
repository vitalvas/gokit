package xstrings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortMap(t *testing.T) {
	t.Run("sorts keys", func(t *testing.T) {
		data := map[string]string{
			"zz": "yes",
			"aa": "there",
			"bb": "no",
		}
		sorted := SortMap(data)
		assert.Equal(t, map[string]string{
			"aa": "there",
			"bb": "no",
			"zz": "yes",
		}, sorted)
	})

	t.Run("empty map", func(t *testing.T) {
		data := map[string]string{}
		sorted := SortMap(data)
		assert.Empty(t, sorted)
	})
}

func TestReplaceMap(t *testing.T) {
	t.Run("replaces all occurrences", func(t *testing.T) {
		data := map[string]string{
			"aa": "there",
			"bb": "no",
			"zz": "yes",
		}
		payload := "hello, aa, bb, zz"
		result := ReplaceMap(payload, data)
		assert.Equal(t, "hello, there, no, yes", result)
	})

	t.Run("no matches", func(t *testing.T) {
		data := map[string]string{"x": "y"}
		payload := "hello, world"
		result := ReplaceMap(payload, data)
		assert.Equal(t, "hello, world", result)
	})
}

func BenchmarkSortMap(b *testing.B) {
	data := map[string]string{
		"zz": "yes",
		"aa": "there",
		"bb": "no",
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = SortMap(data)
	}
}

func BenchmarkReplaceMap(b *testing.B) {
	data := map[string]string{
		"aa": "there",
		"bb": "no",
		"zz": "yes",
	}
	payload := "hello, aa, bb, zz"
	b.ReportAllocs()
	for b.Loop() {
		_ = ReplaceMap(payload, data)
	}
}

func FuzzReplaceMap(f *testing.F) {
	f.Add("hello, aa", "aa", "world")
	f.Add("test", "x", "y")
	f.Add("", "key", "value")

	f.Fuzz(func(t *testing.T, payload, key, value string) {
		data := map[string]string{key: value}
		_ = ReplaceMap(payload, data)
	})
}
