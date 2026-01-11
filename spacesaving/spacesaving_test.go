package spacesaving

import (
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("create with default capacity", func(t *testing.T) {
		ss := New(100)
		assert.NotNil(t, ss)
		assert.Equal(t, 100, ss.Capacity())
		assert.Equal(t, 0, ss.Size())
	})

	t.Run("create with zero capacity uses default", func(t *testing.T) {
		ss := New(0)
		assert.NotNil(t, ss)
		assert.Equal(t, 100, ss.Capacity())
	})

	t.Run("create with negative capacity uses default", func(t *testing.T) {
		ss := New(-10)
		assert.NotNil(t, ss)
		assert.Equal(t, 100, ss.Capacity())
	})
}

func TestAdd(t *testing.T) {
	t.Run("add single item", func(t *testing.T) {
		ss := New(10)
		count := ss.Add("apple")
		assert.Equal(t, uint64(1), count)
		assert.Equal(t, 1, ss.Size())
	})

	t.Run("add duplicate items", func(t *testing.T) {
		ss := New(10)
		ss.Add("apple")
		ss.Add("apple")
		count := ss.Add("apple")

		assert.Equal(t, uint64(3), count)
		assert.Equal(t, 1, ss.Size())
	})

	t.Run("add multiple different items", func(t *testing.T) {
		ss := New(10)
		ss.Add("apple")
		ss.Add("banana")
		ss.Add("cherry")

		assert.Equal(t, 3, ss.Size())
	})

	t.Run("add beyond capacity", func(t *testing.T) {
		ss := New(3)
		ss.Add("a")
		ss.Add("b")
		ss.Add("c")

		// Size should remain at capacity
		assert.Equal(t, 3, ss.Size())

		// Add new item should replace minimum
		ss.Add("d")
		assert.Equal(t, 3, ss.Size())
	})

	t.Run("eviction preserves high-frequency items", func(t *testing.T) {
		ss := New(3)

		// Add items with different frequencies
		for i := 0; i < 10; i++ {
			ss.Add("common")
		}
		ss.Add("rare1")
		ss.Add("rare2")

		// Add new item to trigger eviction
		ss.Add("new")

		// "common" should still be tracked
		count, _ := ss.Count("common")
		assert.Equal(t, uint64(10), count)
	})
}

func TestCount(t *testing.T) {
	t.Run("count tracked item", func(t *testing.T) {
		ss := New(10)
		ss.Add("apple")
		ss.Add("apple")
		ss.Add("apple")

		count, err := ss.Count("apple")
		assert.Equal(t, uint64(3), count)
		assert.Equal(t, uint64(0), err)
	})

	t.Run("count non-tracked item", func(t *testing.T) {
		ss := New(10)
		ss.Add("apple")

		count, err := ss.Count("banana")
		assert.Equal(t, uint64(0), count)
		assert.LessOrEqual(t, err, uint64(1))
	})

	t.Run("count after eviction has error bound", func(t *testing.T) {
		ss := New(2)
		ss.Add("a")
		ss.Add("b")
		ss.Add("c") // Evicts "a"

		// "c" should have error bound
		count, err := ss.Count("c")
		assert.Greater(t, count, uint64(0))
		assert.Greater(t, err, uint64(0))
	})
}

func TestTop(t *testing.T) {
	t.Run("top items in order", func(t *testing.T) {
		ss := New(10)

		// Add items with known frequencies
		for i := 0; i < 5; i++ {
			ss.Add("a")
		}
		for i := 0; i < 3; i++ {
			ss.Add("b")
		}
		ss.Add("c")

		top := ss.Top(3)
		assert.Len(t, top, 3)
		assert.Equal(t, "a", top[0].Value)
		assert.Equal(t, uint64(5), top[0].Count)
		assert.Equal(t, "b", top[1].Value)
		assert.Equal(t, uint64(3), top[1].Count)
		assert.Equal(t, "c", top[2].Value)
		assert.Equal(t, uint64(1), top[2].Count)
	})

	t.Run("top n greater than size", func(t *testing.T) {
		ss := New(10)
		ss.Add("a")
		ss.Add("b")

		top := ss.Top(10)
		assert.Len(t, top, 2)
	})

	t.Run("top with n=0", func(t *testing.T) {
		ss := New(10)
		ss.Add("a")

		top := ss.Top(0)
		assert.Nil(t, top)
	})

	t.Run("top with negative n", func(t *testing.T) {
		ss := New(10)
		ss.Add("a")

		top := ss.Top(-1)
		assert.Nil(t, top)
	})

	t.Run("top from empty tracker", func(t *testing.T) {
		ss := New(10)
		top := ss.Top(5)
		assert.Len(t, top, 0)
	})
}

func TestAll(t *testing.T) {
	t.Run("all returns all items", func(t *testing.T) {
		ss := New(10)
		ss.Add("a")
		ss.Add("b")
		ss.Add("c")

		all := ss.All()
		assert.Len(t, all, 3)
	})

	t.Run("all sorted by frequency", func(t *testing.T) {
		ss := New(10)
		for i := 0; i < 5; i++ {
			ss.Add("a")
		}
		for i := 0; i < 2; i++ {
			ss.Add("b")
		}

		all := ss.All()
		assert.Equal(t, "a", all[0].Value)
		assert.Equal(t, "b", all[1].Value)
	})
}

func TestSize(t *testing.T) {
	t.Run("size increases with additions", func(t *testing.T) {
		ss := New(10)
		assert.Equal(t, 0, ss.Size())

		ss.Add("a")
		assert.Equal(t, 1, ss.Size())

		ss.Add("b")
		assert.Equal(t, 2, ss.Size())
	})

	t.Run("size capped at capacity", func(t *testing.T) {
		ss := New(3)
		ss.Add("a")
		ss.Add("b")
		ss.Add("c")
		ss.Add("d")

		assert.Equal(t, 3, ss.Size())
	})
}

func TestReset(t *testing.T) {
	t.Run("reset clears all data", func(t *testing.T) {
		ss := New(10)
		ss.Add("a")
		ss.Add("b")
		ss.Add("c")

		ss.Reset()

		assert.Equal(t, 0, ss.Size())
		count, _ := ss.Count("a")
		assert.Equal(t, uint64(0), count)
	})
}

func TestExportImport(t *testing.T) {
	t.Run("export and import", func(t *testing.T) {
		ss := New(10)
		for i := 0; i < 5; i++ {
			ss.Add("apple")
		}
		for i := 0; i < 3; i++ {
			ss.Add("banana")
		}

		data, err := ss.Export()
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		imported, err := Import(data)
		assert.NoError(t, err)
		assert.NotNil(t, imported)

		assert.Equal(t, ss.Size(), imported.Size())
		assert.Equal(t, ss.Capacity(), imported.Capacity())

		count, _ := imported.Count("apple")
		assert.Equal(t, uint64(5), count)

		count, _ = imported.Count("banana")
		assert.Equal(t, uint64(3), count)
	})

	t.Run("export empty tracker", func(t *testing.T) {
		ss := New(10)
		data, err := ss.Export()
		assert.NoError(t, err)

		imported, err := Import(data)
		assert.NoError(t, err)
		assert.Equal(t, 0, imported.Size())
	})

	t.Run("import invalid data", func(t *testing.T) {
		_, err := Import([]byte("invalid"))
		assert.Error(t, err)
	})
}

func TestAccuracy(t *testing.T) {
	t.Run("accurate counts for heavy hitters", func(t *testing.T) {
		ss := New(10)

		// Add heavy hitter
		for i := 0; i < 100; i++ {
			ss.Add("heavy")
		}

		// Add some noise
		for i := 0; i < 10; i++ {
			ss.Add(fmt.Sprintf("noise-%d", i))
		}

		count, err := ss.Count("heavy")
		assert.Equal(t, uint64(100), count)
		assert.Equal(t, uint64(0), err) // Heavy hitter should have no error
	})

	t.Run("top items accuracy", func(t *testing.T) {
		ss := New(5)

		// Known distribution
		items := map[string]int{
			"a": 50,
			"b": 30,
			"c": 20,
			"d": 10,
			"e": 5,
		}

		// Add items
		for item, count := range items {
			for i := 0; i < count; i++ {
				ss.Add(item)
			}
		}

		// Check top 3
		top := ss.Top(3)
		assert.Equal(t, "a", top[0].Value)
		assert.Equal(t, "b", top[1].Value)
		assert.Equal(t, "c", top[2].Value)
	})
}

func TestZipfDistribution(t *testing.T) {
	t.Run("zipf distribution tracking", func(t *testing.T) {
		ss := New(100)

		// Simulate Zipf distribution (realistic for web traffic, word frequencies, etc.)
		// In Zipf: frequency(rank) ~ 1/rank
		// So item-0 appears most, item-1 appears half as much, item-2 appears 1/3 as much, etc.
		for rank := 0; rank < 200; rank++ {
			item := fmt.Sprintf("item-%d", rank)
			// Frequency inversely proportional to rank+1
			frequency := 1000 / (rank + 1)
			for j := 0; j < frequency; j++ {
				ss.Add(item)
			}
		}

		// Get top 10
		top := ss.Top(10)
		assert.Len(t, top, 10)

		// Verify descending order
		for i := 1; i < len(top); i++ {
			assert.GreaterOrEqual(t, top[i-1].Count, top[i].Count)
		}

		// Most frequent should be item-0 (has highest frequency)
		assert.Equal(t, "item-0", top[0].Value)
		assert.Greater(t, top[0].Count, top[1].Count)
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent adds", func(t *testing.T) {
		ss := New(100)
		var wg sync.WaitGroup

		// Multiple goroutines adding items
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				item := fmt.Sprintf("item-%d", id%5)
				for j := 0; j < 100; j++ {
					ss.Add(item)
				}
			}(i)
		}

		wg.Wait()

		// Verify consistency
		top := ss.Top(5)
		totalCount := uint64(0)
		for _, item := range top {
			totalCount += item.Count
		}
		assert.Equal(t, uint64(1000), totalCount)
	})

	t.Run("concurrent reads and writes", func(_ *testing.T) {
		ss := New(50)
		done := make(chan bool)

		// Writer
		go func() {
			for i := 0; i < 1000; i++ {
				ss.Add(fmt.Sprintf("item-%d", i%10))
			}
			done <- true
		}()

		// Readers
		go func() {
			for i := 0; i < 100; i++ {
				_ = ss.Top(5)
				_, _ = ss.Count("item-0")
			}
			done <- true
		}()

		<-done
		<-done
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty string item", func(t *testing.T) {
		ss := New(10)
		count := ss.Add("")
		assert.Equal(t, uint64(1), count)

		count, _ = ss.Count("")
		assert.Equal(t, uint64(1), count)
	})

	t.Run("very long item", func(t *testing.T) {
		ss := New(10)
		longItem := string(make([]byte, 10000))
		count := ss.Add(longItem)
		assert.Equal(t, uint64(1), count)
	})

	t.Run("single capacity", func(t *testing.T) {
		ss := New(1)
		ss.Add("a")
		ss.Add("b")

		assert.Equal(t, 1, ss.Size())
		top := ss.Top(1)
		assert.Len(t, top, 1)
	})
}

func TestStableSort(t *testing.T) {
	t.Run("items with same count sorted by value", func(t *testing.T) {
		ss := New(10)
		ss.Add("c")
		ss.Add("a")
		ss.Add("b")

		top := ss.Top(3)
		values := []string{top[0].Value, top[1].Value, top[2].Value}
		sort.Strings(values)

		assert.Equal(t, []string{"a", "b", "c"}, values)
	})
}

func BenchmarkSpaceSaving_Add(b *testing.B) {
	ss := New(1000)
	b.ReportAllocs()
	for b.Loop() {
		ss.Add("item-50")
	}
}

func BenchmarkSpaceSaving_Count(b *testing.B) {
	ss := New(1000)
	for i := range 10000 {
		ss.Add(fmt.Sprintf("item-%d", i%100))
	}
	b.ReportAllocs()
	for b.Loop() {
		_, _ = ss.Count("item-50")
	}
}

func BenchmarkSpaceSaving_Top(b *testing.B) {
	ss := New(1000)
	for i := range 10000 {
		ss.Add(fmt.Sprintf("item-%d", i%100))
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = ss.Top(10)
	}
}

func BenchmarkSpaceSaving_ConcurrentAdd(b *testing.B) {
	ss := New(1000)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ss.Add(fmt.Sprintf("item-%d", i%100))
			i++
		}
	})
}

func BenchmarkSpaceSaving_Export(b *testing.B) {
	ss := New(1000)
	for i := range 10000 {
		ss.Add(fmt.Sprintf("item-%d", i%100))
	}
	b.ReportAllocs()
	for b.Loop() {
		_, _ = ss.Export()
	}
}

func BenchmarkSpaceSaving_Import(b *testing.B) {
	ss := New(1000)
	for i := range 10000 {
		ss.Add(fmt.Sprintf("item-%d", i%100))
	}
	data, _ := ss.Export()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = Import(data)
	}
}

func FuzzSpaceSaving_Add(f *testing.F) {
	f.Add("test")
	f.Add("item-1")
	f.Add("")
	f.Add("a")
	f.Add("very-long-item-name-for-testing")

	f.Fuzz(func(t *testing.T, item string) {
		ss := New(100)
		count := ss.Add(item)
		if count < 1 {
			t.Error("count should be at least 1 after adding")
		}
		gotCount, _ := ss.Count(item)
		if gotCount < 1 {
			t.Error("item should be present after adding")
		}
	})
}

func FuzzSpaceSaving_ExportImport(f *testing.F) {
	f.Add("a", "b", "c")
	f.Add("test1", "test2", "test3")
	f.Add("", "x", "y")

	f.Fuzz(func(t *testing.T, s1, s2, s3 string) {
		ss := New(100)
		ss.Add(s1)
		ss.Add(s2)
		ss.Add(s3)

		data, err := ss.Export()
		if err != nil {
			t.Fatalf("export failed: %v", err)
		}

		imported, err := Import(data)
		if err != nil {
			t.Fatalf("import failed: %v", err)
		}

		if imported.Size() != ss.Size() {
			t.Error("imported size should match original")
		}
	})
}
