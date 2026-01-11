package xstrings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunks(t *testing.T) {
	t.Run("chunks of 2", func(t *testing.T) {
		result := Chunks([]string{"a", "b", "c", "d", "e"}, 2)
		assert.Equal(t, [][]string{{"a", "b"}, {"c", "d"}, {"e"}}, result)
	})

	t.Run("chunks of 3", func(t *testing.T) {
		result := Chunks([]string{"apple", "banana", "cherry", "date", "elderberry"}, 3)
		assert.Equal(t, [][]string{{"apple", "banana", "cherry"}, {"date", "elderberry"}}, result)
	})

	t.Run("chunks of 1", func(t *testing.T) {
		result := Chunks([]string{"one", "two", "three"}, 1)
		assert.Equal(t, [][]string{{"one"}, {"two"}, {"three"}}, result)
	})

	t.Run("chunk size larger than list", func(t *testing.T) {
		result := Chunks([]string{"x", "y", "z"}, 5)
		assert.Equal(t, [][]string{{"x", "y", "z"}}, result)
	})

	t.Run("empty list", func(t *testing.T) {
		result := Chunks([]string{}, 2)
		assert.Empty(t, result)
	})
}

func BenchmarkChunks(b *testing.B) {
	list := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	b.ReportAllocs()
	for b.Loop() {
		_ = Chunks(list, 3)
	}
}

func FuzzChunks(f *testing.F) {
	f.Add("a,b,c,d,e", 2)
	f.Add("x,y,z", 5)
	f.Add("", 1)

	f.Fuzz(func(t *testing.T, input string, chunkSize int) {
		if chunkSize <= 0 {
			return
		}
		var list []string
		if input != "" {
			for _, s := range input {
				list = append(list, string(s))
			}
		}
		result := Chunks(list, chunkSize)
		total := 0
		for _, chunk := range result {
			total += len(chunk)
		}
		if total != len(list) {
			t.Error("total elements in chunks should equal original list length")
		}
	})
}
