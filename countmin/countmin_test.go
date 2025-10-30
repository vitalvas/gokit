package countmin

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("create with error bounds", func(t *testing.T) {
		cm := New(0.001, 0.01)
		assert.NotNil(t, cm)
		assert.Greater(t, cm.Width(), uint32(0))
		assert.Greater(t, cm.Depth(), uint32(0))
		assert.Equal(t, 0.001, cm.Epsilon())
		assert.Equal(t, 0.01, cm.Delta())
	})

	t.Run("create with default epsilon", func(t *testing.T) {
		cm := New(0, 0.01)
		assert.Equal(t, 0.001, cm.Epsilon())
	})

	t.Run("create with default delta", func(t *testing.T) {
		cm := New(0.001, 0)
		assert.Equal(t, 0.01, cm.Delta())
	})

	t.Run("dimensions match error bounds", func(t *testing.T) {
		cm := New(0.01, 0.01)
		// width ≈ e/ε = 2.718/0.01 ≈ 272
		// depth ≈ ln(1/δ) = ln(100) ≈ 5
		assert.InDelta(t, 272, cm.Width(), 10)
		assert.InDelta(t, 5, cm.Depth(), 1)
	})
}

func TestNewWithSize(t *testing.T) {
	t.Run("create with explicit dimensions", func(t *testing.T) {
		cm := NewWithSize(100, 5)
		assert.Equal(t, uint32(100), cm.Width())
		assert.Equal(t, uint32(5), cm.Depth())
	})

	t.Run("create with zero width uses default", func(t *testing.T) {
		cm := NewWithSize(0, 5)
		assert.Equal(t, uint32(272), cm.Width())
	})

	t.Run("create with zero depth uses default", func(t *testing.T) {
		cm := NewWithSize(100, 0)
		assert.Equal(t, uint32(5), cm.Depth())
	})
}

func TestAdd(t *testing.T) {
	t.Run("add single item", func(t *testing.T) {
		cm := New(0.01, 0.01)
		cm.Add([]byte("test"), 1)

		count := cm.Count([]byte("test"))
		assert.Equal(t, uint64(1), count)
		assert.Equal(t, uint64(1), cm.Total())
	})

	t.Run("add same item multiple times", func(t *testing.T) {
		cm := New(0.01, 0.01)
		cm.Add([]byte("test"), 5)
		cm.Add([]byte("test"), 3)

		count := cm.Count([]byte("test"))
		assert.Equal(t, uint64(8), count)
		assert.Equal(t, uint64(8), cm.Total())
	})

	t.Run("add different items", func(t *testing.T) {
		cm := New(0.01, 0.01)
		cm.Add([]byte("item1"), 10)
		cm.Add([]byte("item2"), 20)
		cm.Add([]byte("item3"), 30)

		assert.Equal(t, uint64(10), cm.Count([]byte("item1")))
		assert.Equal(t, uint64(20), cm.Count([]byte("item2")))
		assert.Equal(t, uint64(30), cm.Count([]byte("item3")))
		assert.Equal(t, uint64(60), cm.Total())
	})
}

func TestAddString(t *testing.T) {
	t.Run("add string item", func(t *testing.T) {
		cm := New(0.01, 0.01)
		cm.AddString("hello", 5)

		count := cm.CountString("hello")
		assert.Equal(t, uint64(5), count)
	})

	t.Run("string and bytes equivalence", func(t *testing.T) {
		cm := New(0.01, 0.01)
		cm.AddString("test", 10)

		countStr := cm.CountString("test")
		countBytes := cm.Count([]byte("test"))
		assert.Equal(t, countStr, countBytes)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("update is equivalent to Add(1)", func(t *testing.T) {
		cm1 := New(0.01, 0.01)
		cm2 := New(0.01, 0.01)

		cm1.Update([]byte("test"))
		cm2.Add([]byte("test"), 1)

		assert.Equal(t, cm2.Count([]byte("test")), cm1.Count([]byte("test")))
		assert.Equal(t, cm2.Total(), cm1.Total())
	})

	t.Run("update string", func(t *testing.T) {
		cm := New(0.01, 0.01)
		cm.UpdateString("test")

		count := cm.CountString("test")
		assert.Equal(t, uint64(1), count)
	})
}

func TestCount(t *testing.T) {
	t.Run("count never underestimates", func(t *testing.T) {
		cm := New(0.001, 0.01)

		// Add 100 different items
		for i := 0; i < 100; i++ {
			item := fmt.Sprintf("item-%d", i)
			cm.AddString(item, uint64(i+1))
		}

		// Verify all counts are >= true count
		for i := 0; i < 100; i++ {
			item := fmt.Sprintf("item-%d", i)
			count := cm.CountString(item)
			assert.GreaterOrEqual(t, count, uint64(i+1))
		}
	})

	t.Run("count of non-existent item", func(t *testing.T) {
		cm := New(0.01, 0.01)
		cm.AddString("exists", 10)

		count := cm.CountString("does-not-exist")
		// May be 0 or small value due to hash collisions
		assert.GreaterOrEqual(t, count, uint64(0))
	})
}

func TestTotal(t *testing.T) {
	t.Run("total tracks all additions", func(t *testing.T) {
		cm := New(0.01, 0.01)

		cm.Add([]byte("a"), 10)
		cm.Add([]byte("b"), 20)
		cm.Add([]byte("c"), 30)

		assert.Equal(t, uint64(60), cm.Total())
	})

	t.Run("initial total is zero", func(t *testing.T) {
		cm := New(0.01, 0.01)
		assert.Equal(t, uint64(0), cm.Total())
	})
}

func TestClear(t *testing.T) {
	t.Run("clear resets all counters", func(t *testing.T) {
		cm := New(0.01, 0.01)

		cm.AddString("item1", 100)
		cm.AddString("item2", 200)

		cm.Clear()

		assert.Equal(t, uint64(0), cm.CountString("item1"))
		assert.Equal(t, uint64(0), cm.CountString("item2"))
		assert.Equal(t, uint64(0), cm.Total())
	})
}

func TestMerge(t *testing.T) {
	t.Run("merge two sketches", func(t *testing.T) {
		cm1 := New(0.01, 0.01)
		cm2 := New(0.01, 0.01)

		cm1.AddString("item1", 10)
		cm1.AddString("item2", 20)

		cm2.AddString("item2", 5)
		cm2.AddString("item3", 30)

		err := cm1.Merge(cm2)
		assert.NoError(t, err)

		assert.Equal(t, uint64(10), cm1.CountString("item1"))
		assert.GreaterOrEqual(t, cm1.CountString("item2"), uint64(25))
		assert.Equal(t, uint64(30), cm1.CountString("item3"))
		assert.Equal(t, uint64(65), cm1.Total())
	})

	t.Run("merge with dimension mismatch", func(t *testing.T) {
		cm1 := NewWithSize(100, 5)
		cm2 := NewWithSize(200, 5)

		err := cm1.Merge(cm2)
		assert.Error(t, err)

		var dimErr *DimensionMismatchError
		assert.ErrorAs(t, err, &dimErr)
	})
}

func TestClone(t *testing.T) {
	t.Run("clone creates independent copy", func(t *testing.T) {
		cm1 := New(0.01, 0.01)
		cm1.AddString("item", 100)

		cm2 := cm1.Clone()

		// Modify original
		cm1.AddString("item", 50)

		// Clone should be unchanged
		assert.Equal(t, uint64(100), cm2.CountString("item"))
		assert.Equal(t, uint64(150), cm1.CountString("item"))
	})

	t.Run("clone preserves dimensions", func(t *testing.T) {
		cm1 := NewWithSize(123, 7)
		cm2 := cm1.Clone()

		assert.Equal(t, cm1.Width(), cm2.Width())
		assert.Equal(t, cm1.Depth(), cm2.Depth())
	})
}

func TestEstimatedError(t *testing.T) {
	t.Run("error bound scales with total", func(t *testing.T) {
		cm := New(0.01, 0.01)

		// Add items
		for i := 0; i < 1000; i++ {
			cm.AddString(fmt.Sprintf("item-%d", i), 1)
		}

		estimatedErr := cm.EstimatedError()
		// Error should be approximately ε × N = 0.01 × 1000 = 10
		assert.InDelta(t, 10, estimatedErr, 5)
	})

	t.Run("zero error for empty sketch", func(t *testing.T) {
		cm := New(0.01, 0.01)
		assert.Equal(t, uint64(0), cm.EstimatedError())
	})
}

func TestExportImport(t *testing.T) {
	t.Run("export and import preserves data", func(t *testing.T) {
		cm1 := New(0.01, 0.01)

		cm1.AddString("item1", 100)
		cm1.AddString("item2", 200)
		cm1.AddString("item3", 300)

		// Export
		data, err := cm1.Export()
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		// Import
		cm2, err := Import(data)
		assert.NoError(t, err)

		// Verify data
		assert.Equal(t, cm1.Width(), cm2.Width())
		assert.Equal(t, cm1.Depth(), cm2.Depth())
		assert.Equal(t, cm1.Total(), cm2.Total())
		assert.Equal(t, cm1.CountString("item1"), cm2.CountString("item1"))
		assert.Equal(t, cm1.CountString("item2"), cm2.CountString("item2"))
		assert.Equal(t, cm1.CountString("item3"), cm2.CountString("item3"))
	})

	t.Run("import invalid data", func(t *testing.T) {
		_, err := Import([]byte("invalid data"))
		assert.Error(t, err)
	})
}

func TestAccuracy(t *testing.T) {
	t.Run("accuracy with uniform distribution", func(t *testing.T) {
		cm := New(0.001, 0.01)

		// Add 100 items with frequency 10 each
		for i := 0; i < 100; i++ {
			item := fmt.Sprintf("item-%d", i)
			cm.AddString(item, 10)
		}

		// Check a sample of items
		errorCount := 0
		maxError := uint64(0)

		for i := 0; i < 20; i++ {
			item := fmt.Sprintf("item-%d", i)
			count := cm.CountString(item)
			trueCount := uint64(10)

			if count > trueCount {
				errorCount++
				if count-trueCount > maxError {
					maxError = count - trueCount
				}
			}
		}

		// Most items should have low error
		expectedError := cm.EstimatedError()
		t.Logf("Max error: %d, Expected error bound: %d", maxError, expectedError)
		assert.LessOrEqual(t, maxError, expectedError*2)
	})

	t.Run("accuracy with zipf distribution", func(t *testing.T) {
		cm := New(0.001, 0.01)

		// Zipf distribution: frequency(rank) ~ 1/rank
		for rank := 1; rank <= 100; rank++ {
			item := fmt.Sprintf("item-%d", rank)
			frequency := 1000 / uint64(rank)
			cm.AddString(item, frequency)
		}

		// Check top items (should be very accurate)
		for rank := 1; rank <= 10; rank++ {
			item := fmt.Sprintf("item-%d", rank)
			count := cm.CountString(item)
			trueCount := 1000 / uint64(rank)

			// Allow small overestimation
			assert.GreaterOrEqual(t, count, trueCount)
			assert.LessOrEqual(t, count, trueCount+cm.EstimatedError())
		}
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent adds", func(t *testing.T) {
		cm := New(0.01, 0.01)
		var wg sync.WaitGroup

		// Add from multiple goroutines
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					item := fmt.Sprintf("item-%d", id)
					cm.AddString(item, 1)
				}
			}(i)
		}

		wg.Wait()

		// Verify totals
		total := uint64(0)
		for i := 0; i < 10; i++ {
			item := fmt.Sprintf("item-%d", i)
			count := cm.CountString(item)
			assert.GreaterOrEqual(t, count, uint64(100))
			total += count
		}

		assert.GreaterOrEqual(t, cm.Total(), uint64(1000))
	})

	t.Run("concurrent reads and writes", func(_ *testing.T) {
		cm := New(0.01, 0.01)
		var wg sync.WaitGroup

		// Writers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					item := fmt.Sprintf("item-%d", id)
					cm.AddString(item, 1)
				}
			}(i)
		}

		// Readers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					item := fmt.Sprintf("item-%d", id)
					_ = cm.CountString(item)
					_ = cm.Total()
				}
			}(i)
		}

		wg.Wait()
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		cm := New(0.01, 0.01)
		cm.AddString("", 10)

		count := cm.CountString("")
		assert.Equal(t, uint64(10), count)
	})

	t.Run("empty bytes", func(t *testing.T) {
		cm := New(0.01, 0.01)
		cm.Add([]byte{}, 10)

		count := cm.Count([]byte{})
		assert.Equal(t, uint64(10), count)
	})

	t.Run("very long string", func(t *testing.T) {
		cm := New(0.01, 0.01)
		longStr := string(make([]byte, 10000))
		cm.AddString(longStr, 5)

		count := cm.CountString(longStr)
		assert.Equal(t, uint64(5), count)
	})

	t.Run("add zero count", func(t *testing.T) {
		cm := New(0.01, 0.01)
		cm.AddString("test", 0)

		assert.Equal(t, uint64(0), cm.CountString("test"))
		assert.Equal(t, uint64(0), cm.Total())
	})
}

func TestRealisticScenario(t *testing.T) {
	t.Run("web request tracking", func(t *testing.T) {
		cm := New(0.001, 0.01)

		// Simulate web requests
		endpoints := []string{
			"/api/users",
			"/api/posts",
			"/api/comments",
			"/health",
			"/metrics",
		}

		// Different frequencies for different endpoints
		frequencies := []uint64{100, 50, 30, 500, 10}

		for i, endpoint := range endpoints {
			for j := uint64(0); j < frequencies[i]; j++ {
				cm.UpdateString(endpoint)
			}
		}

		// Verify frequencies
		for i, endpoint := range endpoints {
			count := cm.CountString(endpoint)
			assert.GreaterOrEqual(t, count, frequencies[i])
			t.Logf("Endpoint %s: true=%d, estimate=%d", endpoint, frequencies[i], count)
		}

		assert.Equal(t, uint64(690), cm.Total())
	})
}

// Benchmarks

func BenchmarkAdd(b *testing.B) {
	cm := New(0.01, 0.01)
	data := []byte("benchmark-item")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cm.Add(data, 1)
	}
}

func BenchmarkAddString(b *testing.B) {
	cm := New(0.01, 0.01)
	str := "benchmark-item"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cm.AddString(str, 1)
	}
}

func BenchmarkCount(b *testing.B) {
	cm := New(0.01, 0.01)
	data := []byte("benchmark-item")
	cm.Add(data, 100)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cm.Count(data)
	}
}

func BenchmarkConcurrentAdd(b *testing.B) {
	cm := New(0.01, 0.01)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		data := []byte("benchmark-item")
		for pb.Next() {
			cm.Add(data, 1)
		}
	})
}

func BenchmarkConcurrentCount(b *testing.B) {
	cm := New(0.01, 0.01)
	data := []byte("benchmark-item")
	cm.Add(data, 1000)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cm.Count(data)
		}
	})
}

func BenchmarkClone(b *testing.B) {
	cm := New(0.01, 0.01)

	// Pre-populate
	for i := 0; i < 100; i++ {
		cm.AddString(fmt.Sprintf("item-%d", i), uint64(i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cm.Clone()
	}
}

func BenchmarkExport(b *testing.B) {
	cm := New(0.01, 0.01)

	// Pre-populate
	for i := 0; i < 100; i++ {
		cm.AddString(fmt.Sprintf("item-%d", i), uint64(i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = cm.Export()
	}
}

func BenchmarkImport(b *testing.B) {
	cm := New(0.01, 0.01)
	for i := 0; i < 100; i++ {
		cm.AddString(fmt.Sprintf("item-%d", i), uint64(i))
	}
	data, _ := cm.Export()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Import(data)
	}
}

func TestHashDistribution(t *testing.T) {
	t.Run("hash produces different values for different seeds", func(t *testing.T) {
		data := []byte("test")

		hash1 := hash64(data, 0)
		hash2 := hash64(data, 1)
		hash3 := hash64(data, 2)

		assert.NotEqual(t, hash1, hash2)
		assert.NotEqual(t, hash2, hash3)
		assert.NotEqual(t, hash1, hash3)
	})

	t.Run("hash is deterministic", func(t *testing.T) {
		data := []byte("test")
		seed := uint64(42)

		hash1 := hash64(data, seed)
		hash2 := hash64(data, seed)

		assert.Equal(t, hash1, hash2)
	})
}
