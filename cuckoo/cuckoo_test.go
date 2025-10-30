package cuckoo

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("create filter with capacity", func(t *testing.T) {
		f := New(1000)
		assert.NotNil(t, f)
		assert.Equal(t, uint(0), f.Count())
		assert.Greater(t, f.numBuckets, uint(0))
	})

	t.Run("minimum capacity", func(t *testing.T) {
		f := New(1)
		assert.NotNil(t, f)
		assert.GreaterOrEqual(t, f.numBuckets, uint(2))
	})

	t.Run("large capacity", func(t *testing.T) {
		f := New(100000)
		assert.NotNil(t, f)
		assert.Greater(t, f.numBuckets, uint(0))
	})
}

func TestInsert(t *testing.T) {
	t.Run("insert single element", func(t *testing.T) {
		f := New(100)
		ok := f.Insert([]byte("test"))
		assert.True(t, ok)
		assert.Equal(t, uint(1), f.Count())
	})

	t.Run("insert multiple elements", func(t *testing.T) {
		f := New(100)
		for i := 0; i < 50; i++ {
			ok := f.Insert([]byte(fmt.Sprintf("element-%d", i)))
			assert.True(t, ok)
		}
		assert.Equal(t, uint(50), f.Count())
	})

	t.Run("insert duplicate elements", func(t *testing.T) {
		f := New(100)
		f.Insert([]byte("test"))
		f.Insert([]byte("test"))
		// Both inserts succeed, count increases
		assert.Equal(t, uint(2), f.Count())
	})
}

func TestInsertUnique(t *testing.T) {
	t.Run("insert unique elements only", func(t *testing.T) {
		f := New(100)

		ok1 := f.InsertUnique([]byte("test"))
		assert.True(t, ok1)
		assert.Equal(t, uint(1), f.Count())

		ok2 := f.InsertUnique([]byte("test"))
		assert.False(t, ok2)
		assert.Equal(t, uint(1), f.Count())
	})

	t.Run("insert different unique elements", func(t *testing.T) {
		f := New(100)

		successful := 0
		for i := 0; i < 10; i++ {
			if f.InsertUnique([]byte(fmt.Sprintf("element-%d", i))) {
				successful++
			}
		}
		// Most should succeed (allow for rare fingerprint collisions)
		assert.GreaterOrEqual(t, successful, 8)
		assert.Equal(t, uint(successful), f.Count())
	})
}

func TestContains(t *testing.T) {
	t.Run("contains inserted element", func(t *testing.T) {
		f := New(100)
		data := []byte("test")
		f.Insert(data)
		assert.True(t, f.Contains(data))
	})

	t.Run("does not contain non-inserted element", func(t *testing.T) {
		f := New(100)
		f.Insert([]byte("test1"))
		assert.False(t, f.Contains([]byte("test2")))
	})

	t.Run("contains all inserted elements", func(t *testing.T) {
		f := New(1000)
		elements := make([][]byte, 100)

		for i := 0; i < 100; i++ {
			elements[i] = []byte(fmt.Sprintf("element-%d", i))
			f.Insert(elements[i])
		}

		for i := 0; i < 100; i++ {
			assert.True(t, f.Contains(elements[i]), "element-%d should be found", i)
		}
	})

	t.Run("empty filter", func(t *testing.T) {
		f := New(100)
		assert.False(t, f.Contains([]byte("anything")))
	})
}

func TestDelete(t *testing.T) {
	t.Run("delete existing element", func(t *testing.T) {
		f := New(100)
		data := []byte("test")

		f.Insert(data)
		assert.True(t, f.Contains(data))
		assert.Equal(t, uint(1), f.Count())

		ok := f.Delete(data)
		assert.True(t, ok)
		assert.Equal(t, uint(0), f.Count())
		assert.False(t, f.Contains(data))
	})

	t.Run("delete non-existing element", func(t *testing.T) {
		f := New(100)
		ok := f.Delete([]byte("test"))
		assert.False(t, ok)
		assert.Equal(t, uint(0), f.Count())
	})

	t.Run("delete one of duplicates", func(t *testing.T) {
		f := New(100)
		data := []byte("test")

		f.Insert(data)
		f.Insert(data)
		assert.Equal(t, uint(2), f.Count())

		ok := f.Delete(data)
		assert.True(t, ok)
		assert.Equal(t, uint(1), f.Count())
		assert.True(t, f.Contains(data)) // Still contains one copy
	})

	t.Run("delete multiple elements", func(t *testing.T) {
		f := New(100)

		for i := 0; i < 10; i++ {
			f.Insert([]byte(fmt.Sprintf("element-%d", i)))
		}
		assert.Equal(t, uint(10), f.Count())

		for i := 0; i < 5; i++ {
			ok := f.Delete([]byte(fmt.Sprintf("element-%d", i)))
			assert.True(t, ok)
		}
		assert.Equal(t, uint(5), f.Count())
	})
}

func TestReset(t *testing.T) {
	t.Run("reset clears filter", func(t *testing.T) {
		f := New(100)

		for i := 0; i < 50; i++ {
			f.Insert([]byte(fmt.Sprintf("element-%d", i)))
		}
		assert.Equal(t, uint(50), f.Count())

		f.Reset()
		assert.Equal(t, uint(0), f.Count())
		assert.Equal(t, 0.0, f.LoadFactor())
	})
}

func TestLoadFactor(t *testing.T) {
	t.Run("load factor increases with insertions", func(t *testing.T) {
		f := New(100)

		assert.Equal(t, 0.0, f.LoadFactor())

		f.Insert([]byte("test1"))
		lf1 := f.LoadFactor()
		assert.Greater(t, lf1, 0.0)

		for i := 0; i < 50; i++ {
			f.Insert([]byte(fmt.Sprintf("element-%d", i)))
		}
		lf2 := f.LoadFactor()
		assert.Greater(t, lf2, lf1)
	})

	t.Run("load factor after reset", func(t *testing.T) {
		f := New(100)

		for i := 0; i < 50; i++ {
			f.Insert([]byte(fmt.Sprintf("element-%d", i)))
		}
		assert.Greater(t, f.LoadFactor(), 0.0)

		f.Reset()
		assert.Equal(t, 0.0, f.LoadFactor())
	})
}

func TestExportImport(t *testing.T) {
	t.Run("export and import", func(t *testing.T) {
		f1 := New(1000)

		elements := make([][]byte, 100)
		for i := 0; i < 100; i++ {
			elements[i] = []byte(fmt.Sprintf("element-%d", i))
			f1.Insert(elements[i])
		}

		// Export
		data, err := f1.Export()
		require.NoError(t, err)
		assert.Greater(t, len(data), 0)

		// Import
		f2, err := Import(data)
		require.NoError(t, err)
		assert.Equal(t, f1.Count(), f2.Count())
		assert.Equal(t, f1.numBuckets, f2.numBuckets)

		// Verify all elements are present
		for i := 0; i < 100; i++ {
			assert.True(t, f2.Contains(elements[i]), "element-%d should be found", i)
		}
	})

	t.Run("export empty filter", func(t *testing.T) {
		f1 := New(100)

		data, err := f1.Export()
		require.NoError(t, err)

		f2, err := Import(data)
		require.NoError(t, err)
		assert.Equal(t, uint(0), f2.Count())
	})

	t.Run("import invalid data", func(t *testing.T) {
		invalidData := []byte{0, 1, 2, 3}
		_, err := Import(invalidData)
		assert.Error(t, err)
	})
}

func TestFalsePositiveRate(t *testing.T) {
	t.Run("measure false positive rate", func(t *testing.T) {
		f := New(10000)

		// Insert 5000 elements
		inserted := make(map[string]bool)
		for i := 0; i < 5000; i++ {
			data := []byte(fmt.Sprintf("element-%d", i))
			f.Insert(data)
			inserted[string(data)] = true
		}

		// Test 5000 non-inserted elements
		falsePositives := 0
		for i := 5000; i < 10000; i++ {
			data := []byte(fmt.Sprintf("element-%d", i))
			if f.Contains(data) {
				falsePositives++
			}
		}

		fpRate := float64(falsePositives) / 5000.0
		t.Logf("False positive rate: %.4f%% (%d/5000)", fpRate*100, falsePositives)

		// Should be less than 1% for this configuration
		assert.Less(t, fpRate, 0.01)
	})
}

func TestCapacity(t *testing.T) {
	t.Run("insert up to capacity", func(t *testing.T) {
		capacity := uint(1000)
		f := New(capacity)

		successful := 0
		for i := 0; i < int(capacity); i++ {
			if f.Insert([]byte(fmt.Sprintf("element-%d", i))) {
				successful++
			}
		}

		// Should be able to insert most elements
		assert.Greater(t, successful, int(capacity)*80/100)
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		f := New(100)
		ok := f.Insert([]byte{})
		assert.True(t, ok)
		assert.True(t, f.Contains([]byte{}))
	})

	t.Run("large data", func(t *testing.T) {
		f := New(100)
		data := make([]byte, 10000)
		for i := range data {
			data[i] = byte(i % 256)
		}
		ok := f.Insert(data)
		assert.True(t, ok)
		assert.True(t, f.Contains(data))
	})

	t.Run("single byte values", func(t *testing.T) {
		f := New(256)
		for i := 0; i < 256; i++ {
			f.Insert([]byte{byte(i)})
		}

		for i := 0; i < 256; i++ {
			assert.True(t, f.Contains([]byte{byte(i)}))
		}
	})
}

func TestConcurrentOperations(t *testing.T) {
	t.Run("sequential operations", func(t *testing.T) {
		f := New(1000)

		// Insert
		for i := 0; i < 100; i++ {
			f.Insert([]byte(fmt.Sprintf("element-%d", i)))
		}

		// Contains
		for i := 0; i < 100; i++ {
			assert.True(t, f.Contains([]byte(fmt.Sprintf("element-%d", i))))
		}

		// Delete
		for i := 0; i < 50; i++ {
			f.Delete([]byte(fmt.Sprintf("element-%d", i)))
		}

		assert.Equal(t, uint(50), f.Count())
	})
}

// Benchmarks

func BenchmarkInsert(b *testing.B) {
	f := New(uint(b.N))
	data := []byte("benchmark data")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		f.Insert(data)
	}
}

func BenchmarkInsertUnique(b *testing.B) {
	f := New(uint(b.N))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		f.InsertUnique([]byte(fmt.Sprintf("element-%d", i)))
	}
}

func BenchmarkContains(b *testing.B) {
	f := New(100000)
	for i := 0; i < 50000; i++ {
		f.Insert([]byte(fmt.Sprintf("element-%d", i)))
	}
	data := []byte("element-25000")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = f.Contains(data)
	}
}

func BenchmarkDelete(b *testing.B) {
	elements := make([][]byte, b.N)
	f := New(uint(b.N))

	for i := 0; i < b.N; i++ {
		elements[i] = []byte(fmt.Sprintf("element-%d", i))
		f.Insert(elements[i])
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		f.Delete(elements[i])
	}
}

func BenchmarkExport(b *testing.B) {
	f := New(10000)
	for i := 0; i < 5000; i++ {
		f.Insert([]byte(fmt.Sprintf("element-%d", i)))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = f.Export()
	}
}

func BenchmarkImport(b *testing.B) {
	f := New(10000)
	for i := 0; i < 5000; i++ {
		f.Insert([]byte(fmt.Sprintf("element-%d", i)))
	}
	data, _ := f.Export()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Import(data)
	}
}

func BenchmarkRandom(b *testing.B) {
	f := New(100000)
	r := rand.New(rand.NewSource(42))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data := make([]byte, 16)
		r.Read(data)
		f.Insert(data)
	}
}
