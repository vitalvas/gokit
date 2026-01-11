package markov

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpool_add(t *testing.T) {
	t.Run("Add single string", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		index := s.add("test")
		assert.Equal(t, 0, index)
		assert.Equal(t, 1, len(s.stringMap))
		assert.Equal(t, 1, len(s.intMap))
	})

	t.Run("Add same string twice", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		index1 := s.add("test")
		index2 := s.add("test")

		assert.Equal(t, index1, index2)
		assert.Equal(t, 1, len(s.stringMap))
		assert.Equal(t, 1, len(s.intMap))
	})

	t.Run("Add multiple different strings", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		index1 := s.add("a")
		index2 := s.add("b")
		index3 := s.add("c")

		assert.Equal(t, 0, index1)
		assert.Equal(t, 1, index2)
		assert.Equal(t, 2, index3)
		assert.Equal(t, 3, len(s.stringMap))
		assert.Equal(t, 3, len(s.intMap))
	})

	t.Run("Add empty string", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		index := s.add("")
		assert.Equal(t, 0, index)
		assert.Equal(t, 1, len(s.stringMap))
	})

	t.Run("Verify bidirectional mapping", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		strings := []string{"hello", "world", "test"}
		indices := make([]int, len(strings))

		for i, str := range strings {
			indices[i] = s.add(str)
		}

		for i, str := range strings {
			idx := indices[i]
			assert.Equal(t, str, s.intMap[idx])
			assert.Equal(t, idx, s.stringMap[str])
		}
	})
}

func TestSpool_get(t *testing.T) {
	t.Run("Get existing string", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		s.add("test")
		index, ok := s.get("test")

		assert.True(t, ok)
		assert.Equal(t, 0, index)
	})

	t.Run("Get non-existing string", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		index, ok := s.get("nonexistent")

		assert.False(t, ok)
		assert.Equal(t, 0, index)
	})

	t.Run("Get multiple strings", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		s.add("a")
		s.add("b")
		s.add("c")

		index1, ok1 := s.get("a")
		index2, ok2 := s.get("b")
		index3, ok3 := s.get("c")

		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.True(t, ok3)
		assert.Equal(t, 0, index1)
		assert.Equal(t, 1, index2)
		assert.Equal(t, 2, index3)
	})

	t.Run("Get after adding same string twice", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		idx1 := s.add("test")
		idx2 := s.add("test")
		index, ok := s.get("test")

		assert.True(t, ok)
		assert.Equal(t, idx1, index)
		assert.Equal(t, idx2, index)
	})
}

func TestSpool_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent add different strings", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		count := 100
		done := make(chan bool, count)

		for i := 0; i < count; i++ {
			go func(n int) {
				s.add(string(rune('a' + n%26)))
				done <- true
			}(i)
		}

		for i := 0; i < count; i++ {
			<-done
		}

		assert.LessOrEqual(t, len(s.stringMap), 26)
		assert.Equal(t, len(s.stringMap), len(s.intMap))
	})

	t.Run("Concurrent add same string", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		count := 100
		done := make(chan bool, count)
		indices := make(chan int, count)

		for i := 0; i < count; i++ {
			go func() {
				idx := s.add("same")
				indices <- idx
				done <- true
			}()
		}

		for i := 0; i < count; i++ {
			<-done
		}
		close(indices)

		firstIndex := <-indices
		for idx := range indices {
			assert.Equal(t, firstIndex, idx, "All concurrent adds of same string should return same index")
		}

		assert.Equal(t, 1, len(s.stringMap))
		assert.Equal(t, 1, len(s.intMap))
	})

	t.Run("Concurrent add and get", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		s.add("initial")

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				s.add("test")
			}
		}()

		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				s.get("test")
			}
		}()

		wg.Wait()

		assert.LessOrEqual(t, len(s.stringMap), 2)
	})
}

func TestSpool_Sequential(t *testing.T) {
	t.Run("Indices are sequential", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		strings := []string{"a", "b", "c", "d", "e"}

		for i, str := range strings {
			index := s.add(str)
			assert.Equal(t, i, index, "Indices should be sequential starting from 0")
		}
	})

	t.Run("Indices are not affected by duplicates", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		s.add("a")
		s.add("a")
		s.add("b")
		s.add("a")
		s.add("c")

		assert.Equal(t, 0, s.stringMap["a"])
		assert.Equal(t, 1, s.stringMap["b"])
		assert.Equal(t, 2, s.stringMap["c"])
		assert.Equal(t, 3, len(s.stringMap))
	})
}

func TestSpool_EdgeCases(t *testing.T) {
	t.Run("Special characters", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		specialChars := []string{"!", "@", "#", "$", "%", "^", "&", "*"}

		for i, char := range specialChars {
			index := s.add(char)
			assert.Equal(t, i, index)
		}

		assert.Equal(t, len(specialChars), len(s.stringMap))
	})

	t.Run("Unicode characters", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		unicodeStrings := []string{"你好", "世界", "テスト", "🎉"}

		for i, str := range unicodeStrings {
			index := s.add(str)
			assert.Equal(t, i, index)
		}

		for i, str := range unicodeStrings {
			idx, ok := s.get(str)
			assert.True(t, ok)
			assert.Equal(t, i, idx)
		}
	})

	t.Run("Very long string", func(t *testing.T) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		longString := ""
		for i := 0; i < 10000; i++ {
			longString += "a"
		}

		index := s.add(longString)
		assert.Equal(t, 0, index)

		idx, ok := s.get(longString)
		assert.True(t, ok)
		assert.Equal(t, 0, idx)
	})
}

func BenchmarkSpool_add(b *testing.B) {
	b.Run("UniqueStrings", func(b *testing.B) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		b.ReportAllocs()
		b.ResetTimer()

		i := 0
		for b.Loop() {
			s.add(fmt.Sprintf("str%d", i))
			i++
		}
	})

	b.Run("DuplicateStrings", func(b *testing.B) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		s.add("test")

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			s.add("test")
		}
	})

	b.Run("MixedStrings", func(b *testing.B) {
		s := &spool{
			stringMap: make(map[string]int),
			intMap:    make(map[int]string),
		}

		b.ReportAllocs()
		b.ResetTimer()

		i := 0
		for b.Loop() {
			s.add(fmt.Sprintf("str%d", i%100))
			i++
		}
	})
}

func BenchmarkSpool_get(b *testing.B) {
	s := &spool{
		stringMap: make(map[string]int),
		intMap:    make(map[int]string),
	}

	for i := 0; i < 1000; i++ {
		s.add(fmt.Sprintf("str%d", i))
	}

	b.Run("Existing", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			s.get(fmt.Sprintf("str%d", i%1000))
			i++
		}
	})

	b.Run("NonExisting", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			s.get(fmt.Sprintf("nonexist%d", i))
			i++
		}
	})
}

func BenchmarkSpool_ConcurrentAdd(b *testing.B) {
	s := &spool{
		stringMap: make(map[string]int),
		intMap:    make(map[int]string),
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s.add(fmt.Sprintf("str%d", i%100))
			i++
		}
	})
}

func BenchmarkSpool_ConcurrentGet(b *testing.B) {
	s := &spool{
		stringMap: make(map[string]int),
		intMap:    make(map[int]string),
	}

	for i := 0; i < 1000; i++ {
		s.add(fmt.Sprintf("str%d", i))
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s.get(fmt.Sprintf("str%d", i%1000))
			i++
		}
	})
}
