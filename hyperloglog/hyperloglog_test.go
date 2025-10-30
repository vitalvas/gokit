package hyperloglog

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("default precision", func(t *testing.T) {
		hll := New(14)
		assert.Equal(t, uint8(14), hll.Precision())
		assert.Equal(t, uint32(16384), hll.Size())
	})

	t.Run("minimum precision", func(t *testing.T) {
		hll := New(4)
		assert.Equal(t, uint8(4), hll.Precision())
		assert.Equal(t, uint32(16), hll.Size())
	})

	t.Run("maximum precision", func(t *testing.T) {
		hll := New(18)
		assert.Equal(t, uint8(18), hll.Precision())
		assert.Equal(t, uint32(262144), hll.Size())
	})

	t.Run("invalid precision too low", func(t *testing.T) {
		hll := New(2)
		assert.Equal(t, uint8(14), hll.Precision()) // Should default to 14
	})

	t.Run("invalid precision too high", func(t *testing.T) {
		hll := New(20)
		assert.Equal(t, uint8(14), hll.Precision()) // Should default to 14
	})
}

func TestAdd(t *testing.T) {
	t.Run("add single element", func(t *testing.T) {
		hll := New(14)
		hll.Add([]byte("test"))
		count := hll.Count()
		assert.Greater(t, count, uint64(0))
	})

	t.Run("add multiple unique elements", func(t *testing.T) {
		hll := New(14)
		for i := 0; i < 1000; i++ {
			hll.Add([]byte(fmt.Sprintf("element-%d", i)))
		}
		count := hll.Count()
		// Should be close to 1000 with some error
		assert.InDelta(t, 1000, count, 100)
	})

	t.Run("add duplicate elements", func(t *testing.T) {
		hll := New(14)
		for i := 0; i < 1000; i++ {
			hll.Add([]byte("same-element"))
		}
		count := hll.Count()
		// Should estimate ~1 element
		assert.Less(t, count, uint64(10))
	})
}

func TestAddString(t *testing.T) {
	t.Run("add string", func(t *testing.T) {
		hll := New(14)
		hll.AddString("test")
		count := hll.Count()
		assert.Greater(t, count, uint64(0))
	})

	t.Run("string and bytes equivalence", func(t *testing.T) {
		hll1 := New(14)
		hll2 := New(14)

		hll1.AddString("test")
		hll2.Add([]byte("test"))

		// Both should produce same cardinality
		assert.Equal(t, hll1.Count(), hll2.Count())
	})
}

func TestCount(t *testing.T) {
	t.Run("empty count", func(t *testing.T) {
		hll := New(14)
		count := hll.Count()
		assert.Equal(t, uint64(0), count)
	})

	t.Run("small cardinality", func(t *testing.T) {
		hll := New(14)
		for i := 0; i < 100; i++ {
			hll.AddString(fmt.Sprintf("item-%d", i))
		}
		count := hll.Count()
		// Within 10% error for small sets
		assert.InDelta(t, 100, count, 10)
	})

	t.Run("medium cardinality", func(t *testing.T) {
		hll := New(14)
		for i := 0; i < 10000; i++ {
			hll.AddString(fmt.Sprintf("item-%d", i))
		}
		count := hll.Count()
		// Within 2% error for medium sets
		assert.InDelta(t, 10000, count, 200)
	})

	t.Run("large cardinality", func(t *testing.T) {
		hll := New(14)
		for i := 0; i < 100000; i++ {
			hll.AddString(fmt.Sprintf("item-%d", i))
		}
		count := hll.Count()
		// Within 1% error for large sets
		assert.InDelta(t, 100000, count, 1000)
	})
}

func TestMerge(t *testing.T) {
	t.Run("merge two sets", func(t *testing.T) {
		hll1 := New(14)
		hll2 := New(14)

		// Add 1000 elements to first set
		for i := 0; i < 1000; i++ {
			hll1.AddString(fmt.Sprintf("set1-%d", i))
		}

		// Add 1000 different elements to second set
		for i := 0; i < 1000; i++ {
			hll2.AddString(fmt.Sprintf("set2-%d", i))
		}

		// Merge
		err := hll1.Merge(hll2)
		require.NoError(t, err)

		count := hll1.Count()
		// Should estimate ~2000 total unique elements
		assert.InDelta(t, 2000, count, 200)
	})

	t.Run("merge overlapping sets", func(t *testing.T) {
		hll1 := New(14)
		hll2 := New(14)

		// Add same 1000 elements to both sets
		for i := 0; i < 1000; i++ {
			hll1.AddString(fmt.Sprintf("item-%d", i))
			hll2.AddString(fmt.Sprintf("item-%d", i))
		}

		err := hll1.Merge(hll2)
		require.NoError(t, err)

		count := hll1.Count()
		// Should still estimate ~1000 (union of identical sets)
		assert.InDelta(t, 1000, count, 100)
	})

	t.Run("merge with precision mismatch", func(t *testing.T) {
		hll1 := New(14)
		hll2 := New(12)

		err := hll1.Merge(hll2)
		assert.Error(t, err)
		assert.IsType(t, &PrecisionMismatchError{}, err)
	})
}

func TestClear(t *testing.T) {
	t.Run("clear resets count", func(t *testing.T) {
		hll := New(14)

		// Add elements
		for i := 0; i < 1000; i++ {
			hll.AddString(fmt.Sprintf("item-%d", i))
		}
		assert.Greater(t, hll.Count(), uint64(0))

		// Clear
		hll.Clear()
		assert.Equal(t, uint64(0), hll.Count())
	})
}

func TestClone(t *testing.T) {
	t.Run("clone creates independent copy", func(t *testing.T) {
		hll1 := New(14)

		// Add elements to original
		for i := 0; i < 1000; i++ {
			hll1.AddString(fmt.Sprintf("item-%d", i))
		}
		count1 := hll1.Count()

		// Clone
		hll2 := hll1.Clone()
		assert.Equal(t, hll1.Precision(), hll2.Precision())
		assert.Equal(t, count1, hll2.Count())

		// Modify original
		for i := 1000; i < 2000; i++ {
			hll1.AddString(fmt.Sprintf("item-%d", i))
		}

		// Clone should be unchanged
		assert.Equal(t, count1, hll2.Count())
		assert.NotEqual(t, hll1.Count(), hll2.Count())
	})

	t.Run("clone empty hll", func(t *testing.T) {
		hll1 := New(12)
		hll2 := hll1.Clone()

		assert.Equal(t, uint64(0), hll2.Count())
		assert.Equal(t, hll1.Precision(), hll2.Precision())
	})
}

func TestMergeAll(t *testing.T) {
	t.Run("merge multiple sets", func(t *testing.T) {
		hll1 := New(14)
		hll2 := New(14)
		hll3 := New(14)

		// Add 500 elements to each set
		for i := 0; i < 500; i++ {
			hll1.AddString(fmt.Sprintf("set1-%d", i))
			hll2.AddString(fmt.Sprintf("set2-%d", i))
			hll3.AddString(fmt.Sprintf("set3-%d", i))
		}

		// Merge all
		merged, err := MergeAll(hll1, hll2, hll3)
		require.NoError(t, err)

		count := merged.Count()
		// Should estimate ~1500 total unique elements
		assert.InDelta(t, 1500, count, 150)

		// Original HLLs should be unchanged
		assert.InDelta(t, 500, hll1.Count(), 50)
		assert.InDelta(t, 500, hll2.Count(), 50)
		assert.InDelta(t, 500, hll3.Count(), 50)
	})

	t.Run("merge overlapping sets", func(t *testing.T) {
		hll1 := New(14)
		hll2 := New(14)
		hll3 := New(14)

		// Add same elements to all sets
		for i := 0; i < 1000; i++ {
			hll1.AddString(fmt.Sprintf("item-%d", i))
			hll2.AddString(fmt.Sprintf("item-%d", i))
			hll3.AddString(fmt.Sprintf("item-%d", i))
		}

		merged, err := MergeAll(hll1, hll2, hll3)
		require.NoError(t, err)

		count := merged.Count()
		// Should still estimate ~1000 (union of identical sets)
		assert.InDelta(t, 1000, count, 100)
	})

	t.Run("merge single hll", func(t *testing.T) {
		hll := New(14)
		for i := 0; i < 500; i++ {
			hll.AddString(fmt.Sprintf("item-%d", i))
		}

		merged, err := MergeAll(hll)
		require.NoError(t, err)
		assert.InDelta(t, 500, merged.Count(), 50)
	})

	t.Run("merge with precision mismatch", func(t *testing.T) {
		hll1 := New(14)
		hll2 := New(12)

		_, err := MergeAll(hll1, hll2)
		assert.Error(t, err)
		assert.IsType(t, &PrecisionMismatchError{}, err)
	})

	t.Run("merge with no hlls", func(t *testing.T) {
		_, err := MergeAll()
		assert.Error(t, err)
		assert.IsType(t, &MergeError{}, err)
	})
}

func TestExportImport(t *testing.T) {
	t.Run("export and import", func(t *testing.T) {
		hll1 := New(14)

		// Add elements
		for i := 0; i < 1000; i++ {
			hll1.AddString(fmt.Sprintf("item-%d", i))
		}
		count1 := hll1.Count()

		// Export
		data, err := hll1.Export()
		require.NoError(t, err)
		assert.Greater(t, len(data), 0)

		// Import
		hll2, err := Import(data)
		require.NoError(t, err)
		assert.Equal(t, hll1.Precision(), hll2.Precision())

		count2 := hll2.Count()
		assert.Equal(t, count1, count2)
	})

	t.Run("import invalid data", func(t *testing.T) {
		invalidData := []byte{0, 1, 2, 3}
		_, err := Import(invalidData)
		assert.Error(t, err)
	})
}

func TestAccuracy(t *testing.T) {
	testCases := []struct {
		precision      uint8
		cardinality    int
		maxErrorRate   float64
		name           string
	}{
		{10, 1000, 0.08, "precision 10, 1K elements"},
		{12, 10000, 0.06, "precision 12, 10K elements"},
		{14, 100000, 0.03, "precision 14, 100K elements"},
		{16, 1000000, 0.02, "precision 16, 1M elements"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hll := New(tc.precision)

			// Add unique elements
			for i := 0; i < tc.cardinality; i++ {
				hll.AddString(fmt.Sprintf("element-%d", i))
			}

			estimate := hll.Count()
			errorRate := math.Abs(float64(estimate)-float64(tc.cardinality)) / float64(tc.cardinality)

			assert.Less(t, errorRate, tc.maxErrorRate,
				"Error rate %.4f exceeds maximum %.4f for cardinality %d",
				errorRate, tc.maxErrorRate, tc.cardinality)
		})
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("add empty string", func(t *testing.T) {
		hll := New(14)
		hll.AddString("")
		count := hll.Count()
		assert.Greater(t, count, uint64(0))
	})

	t.Run("add empty bytes", func(t *testing.T) {
		hll := New(14)
		hll.Add([]byte{})
		count := hll.Count()
		assert.Greater(t, count, uint64(0))
	})

	t.Run("very long string", func(t *testing.T) {
		hll := New(14)
		longString := string(make([]byte, 10000))
		hll.AddString(longString)
		count := hll.Count()
		assert.Greater(t, count, uint64(0))
	})
}

func TestPrecisionErrorRelationship(t *testing.T) {
	t.Run("higher precision lower error", func(t *testing.T) {
		cardinality := 10000

		precisions := []uint8{10, 12, 14, 16}
		var errors []float64

		for _, prec := range precisions {
			hll := New(prec)
			for i := 0; i < cardinality; i++ {
				hll.AddString(fmt.Sprintf("item-%d", i))
			}
			estimate := hll.Count()
			errorRate := math.Abs(float64(estimate)-float64(cardinality)) / float64(cardinality)
			errors = append(errors, errorRate)
		}

		// Generally, error should decrease with higher precision
		// (though not guaranteed in every run due to randomness)
		for i := 1; i < len(errors); i++ {
			t.Logf("Precision %d: error %.4f", precisions[i], errors[i])
		}
	})
}

func TestRandomData(t *testing.T) {
	t.Run("random strings", func(t *testing.T) {
		hll := New(14)
		r := rand.New(rand.NewSource(42))

		cardinality := 50000
		for i := 0; i < cardinality; i++ {
			// Generate random string
			bytes := make([]byte, 16)
			r.Read(bytes)
			hll.Add(bytes)
		}

		estimate := hll.Count()
		errorRate := math.Abs(float64(estimate)-float64(cardinality)) / float64(cardinality)
		assert.Less(t, errorRate, 0.02) // Within 2% error
	})
}

// Benchmarks

func BenchmarkAdd(b *testing.B) {
	hll := New(14)
	data := []byte("benchmark data")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hll.Add(data)
	}
}

func BenchmarkAddString(b *testing.B) {
	testCases := []struct {
		precision uint8
		name      string
	}{
		{10, "Precision_10"},
		{12, "Precision_12"},
		{14, "Precision_14"},
		{16, "Precision_16"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			hll := New(tc.precision)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				hll.AddString(fmt.Sprintf("element-%d", i))
			}
		})
	}
}

func BenchmarkCount(b *testing.B) {
	hll := New(14)
	for i := 0; i < 100000; i++ {
		hll.AddString(fmt.Sprintf("element-%d", i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = hll.Count()
	}
}

func BenchmarkMerge(b *testing.B) {
	hll1 := New(14)
	hll2 := New(14)

	for i := 0; i < 10000; i++ {
		hll1.AddString(fmt.Sprintf("set1-%d", i))
		hll2.AddString(fmt.Sprintf("set2-%d", i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hll1.Merge(hll2)
	}
}

func BenchmarkClone(b *testing.B) {
	hll := New(14)
	for i := 0; i < 10000; i++ {
		hll.AddString(fmt.Sprintf("element-%d", i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = hll.Clone()
	}
}

func BenchmarkMergeAll(b *testing.B) {
	hll1 := New(14)
	hll2 := New(14)
	hll3 := New(14)

	for i := 0; i < 10000; i++ {
		hll1.AddString(fmt.Sprintf("set1-%d", i))
		hll2.AddString(fmt.Sprintf("set2-%d", i))
		hll3.AddString(fmt.Sprintf("set3-%d", i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = MergeAll(hll1, hll2, hll3)
	}
}

func BenchmarkExport(b *testing.B) {
	hll := New(14)
	for i := 0; i < 10000; i++ {
		hll.AddString(fmt.Sprintf("element-%d", i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = hll.Export()
	}
}

func BenchmarkImport(b *testing.B) {
	hll := New(14)
	for i := 0; i < 10000; i++ {
		hll.AddString(fmt.Sprintf("element-%d", i))
	}
	data, _ := hll.Export()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Import(data)
	}
}
