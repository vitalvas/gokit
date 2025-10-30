package tdigest

import (
	"math"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("create with default compression", func(t *testing.T) {
		td := New(100)
		assert.NotNil(t, td)
		assert.Equal(t, 100.0, td.Compression())
		assert.Equal(t, 0.0, td.Count())
	})

	t.Run("create with zero compression uses default", func(t *testing.T) {
		td := New(0)
		assert.Equal(t, 100.0, td.Compression())
	})

	t.Run("create with custom compression", func(t *testing.T) {
		td := New(200)
		assert.Equal(t, 200.0, td.Compression())
	})
}

func TestAdd(t *testing.T) {
	t.Run("add single value", func(t *testing.T) {
		td := New(100)
		td.Add(5.0)
		assert.Equal(t, 1.0, td.Count())
		assert.Equal(t, 5.0, td.Min())
		assert.Equal(t, 5.0, td.Max())
	})

	t.Run("add multiple values", func(t *testing.T) {
		td := New(100)
		for i := 1; i <= 100; i++ {
			td.Add(float64(i))
		}
		assert.Equal(t, 100.0, td.Count())
		assert.Equal(t, 1.0, td.Min())
		assert.Equal(t, 100.0, td.Max())
	})

	t.Run("add maintains min/max", func(t *testing.T) {
		td := New(100)
		td.Add(50)
		td.Add(10)
		td.Add(90)
		td.Add(5)
		td.Add(95)

		assert.Equal(t, 5.0, td.Min())
		assert.Equal(t, 95.0, td.Max())
	})
}

func TestAddWeighted(t *testing.T) {
	t.Run("add weighted values", func(t *testing.T) {
		td := New(100)
		td.AddWeighted(5.0, 3.0)
		assert.Equal(t, 3.0, td.Count())
	})

	t.Run("add zero weight is ignored", func(t *testing.T) {
		td := New(100)
		td.AddWeighted(5.0, 0)
		assert.Equal(t, 0.0, td.Count())
	})

	t.Run("add negative weight is ignored", func(t *testing.T) {
		td := New(100)
		td.AddWeighted(5.0, -1)
		assert.Equal(t, 0.0, td.Count())
	})
}

func TestQuantile(t *testing.T) {
	t.Run("quantile on empty digest", func(t *testing.T) {
		td := New(100)
		assert.True(t, math.IsNaN(td.Quantile(0.5)))
	})

	t.Run("quantile with single value", func(t *testing.T) {
		td := New(100)
		td.Add(42)
		assert.Equal(t, 42.0, td.Quantile(0.5))
		assert.Equal(t, 42.0, td.Quantile(0.0))
		assert.Equal(t, 42.0, td.Quantile(1.0))
	})

	t.Run("median of 1-100", func(t *testing.T) {
		td := New(100)
		for i := 1; i <= 100; i++ {
			td.Add(float64(i))
		}

		median := td.Quantile(0.5)
		assert.InDelta(t, 50.5, median, 2.0)
	})

	t.Run("quantiles of uniform distribution", func(t *testing.T) {
		td := New(100)
		for i := 0; i < 1000; i++ {
			td.Add(float64(i))
		}

		// Test various quantiles
		assert.InDelta(t, 0, td.Quantile(0.0), 1)
		assert.InDelta(t, 250, td.Quantile(0.25), 10)
		assert.InDelta(t, 500, td.Quantile(0.50), 10)
		assert.InDelta(t, 750, td.Quantile(0.75), 10)
		assert.InDelta(t, 950, td.Quantile(0.95), 10)
		assert.InDelta(t, 990, td.Quantile(0.99), 10)
		assert.InDelta(t, 999, td.Quantile(1.0), 1)
	})

	t.Run("quantiles clamp to 0-1", func(t *testing.T) {
		td := New(100)
		for i := 1; i <= 100; i++ {
			td.Add(float64(i))
		}

		// Negative quantile treated as 0
		assert.Equal(t, td.Quantile(0.0), td.Quantile(-0.5))

		// > 1 quantile treated as 1
		assert.Equal(t, td.Quantile(1.0), td.Quantile(1.5))
	})
}

func TestCDF(t *testing.T) {
	t.Run("cdf on empty digest", func(t *testing.T) {
		td := New(100)
		assert.True(t, math.IsNaN(td.CDF(50)))
	})

	t.Run("cdf with single value", func(t *testing.T) {
		td := New(100)
		td.Add(50)

		assert.Equal(t, 0.0, td.CDF(49))
		assert.Equal(t, 1.0, td.CDF(50))
		assert.Equal(t, 1.0, td.CDF(51))
	})

	t.Run("cdf of uniform distribution", func(t *testing.T) {
		td := New(100)
		for i := 0; i < 1000; i++ {
			td.Add(float64(i))
		}

		// Test CDF values
		assert.InDelta(t, 0.0, td.CDF(-10), 0.01)
		assert.InDelta(t, 0.25, td.CDF(250), 0.05)
		assert.InDelta(t, 0.50, td.CDF(500), 0.05)
		assert.InDelta(t, 0.75, td.CDF(750), 0.05)
		assert.InDelta(t, 0.95, td.CDF(950), 0.05)
		assert.InDelta(t, 1.0, td.CDF(1000), 0.01)
	})
}

func TestMean(t *testing.T) {
	t.Run("mean on empty digest", func(t *testing.T) {
		td := New(100)
		assert.True(t, math.IsNaN(td.Mean()))
	})

	t.Run("mean of 1-100", func(t *testing.T) {
		td := New(100)
		for i := 1; i <= 100; i++ {
			td.Add(float64(i))
		}
		assert.InDelta(t, 50.5, td.Mean(), 0.5)
	})

	t.Run("mean with weighted values", func(t *testing.T) {
		td := New(100)
		td.AddWeighted(10, 1)
		td.AddWeighted(20, 2)
		td.AddWeighted(30, 1)

		// (10*1 + 20*2 + 30*1) / 4 = 80/4 = 20
		assert.InDelta(t, 20.0, td.Mean(), 0.1)
	})
}

func TestMerge(t *testing.T) {
	t.Run("merge two digests", func(t *testing.T) {
		td1 := New(100)
		td2 := New(100)

		for i := 0; i < 50; i++ {
			td1.Add(float64(i))
		}

		for i := 50; i < 100; i++ {
			td2.Add(float64(i))
		}

		td1.Merge(td2)

		assert.Equal(t, 100.0, td1.Count())
		assert.Equal(t, 0.0, td1.Min())
		assert.Equal(t, 99.0, td1.Max())
		assert.InDelta(t, 49.5, td1.Mean(), 1.0)
	})

	t.Run("merge with empty digest", func(t *testing.T) {
		td1 := New(100)
		td2 := New(100)

		td1.Add(10)
		td1.Add(20)

		td1.Merge(td2)
		assert.Equal(t, 2.0, td1.Count())
	})

	t.Run("merge nil digest", func(t *testing.T) {
		td := New(100)
		td.Add(10)
		count := td.Count()

		td.Merge(nil)
		assert.Equal(t, count, td.Count())
	})

	t.Run("merge maintains quantile accuracy", func(t *testing.T) {
		td1 := New(100)
		td2 := New(100)

		for i := 0; i < 500; i++ {
			td1.Add(float64(i))
			td2.Add(float64(i + 500))
		}

		td1.Merge(td2)

		// Median should be around 500
		assert.InDelta(t, 500, td1.Quantile(0.5), 20)
	})
}

func TestReset(t *testing.T) {
	t.Run("reset clears digest", func(t *testing.T) {
		td := New(100)
		for i := 0; i < 100; i++ {
			td.Add(float64(i))
		}

		assert.Greater(t, td.Count(), 0.0)

		td.Reset()

		assert.Equal(t, 0.0, td.Count())
		assert.True(t, math.IsNaN(td.Mean()))
	})
}

func TestExportImport(t *testing.T) {
	t.Run("export and import", func(t *testing.T) {
		td1 := New(100)
		for i := 0; i < 1000; i++ {
			td1.Add(float64(i))
		}

		// Export
		data, err := td1.Export()
		require.NoError(t, err)
		assert.Greater(t, len(data), 0)

		// Import
		td2, err := Import(data)
		require.NoError(t, err)

		assert.Equal(t, td1.Count(), td2.Count())
		assert.Equal(t, td1.Min(), td2.Min())
		assert.Equal(t, td1.Max(), td2.Max())
		assert.InDelta(t, td1.Mean(), td2.Mean(), 0.1)
		assert.InDelta(t, td1.Quantile(0.5), td2.Quantile(0.5), 1.0)
		assert.InDelta(t, td1.Quantile(0.95), td2.Quantile(0.95), 5.0)
	})

	t.Run("export empty digest", func(t *testing.T) {
		td1 := New(100)

		data, err := td1.Export()
		require.NoError(t, err)

		td2, err := Import(data)
		require.NoError(t, err)
		assert.Equal(t, 0.0, td2.Count())
	})

	t.Run("import invalid data", func(t *testing.T) {
		invalidData := []byte{0, 1, 2, 3}
		_, err := Import(invalidData)
		assert.Error(t, err)
	})
}

func TestCompression(t *testing.T) {
	t.Run("higher compression maintains more centroids", func(t *testing.T) {
		td1 := New(50)
		td2 := New(200)

		for i := 0; i < 1000; i++ {
			td1.Add(float64(i))
			td2.Add(float64(i))
		}

		// Higher compression should have more centroids
		assert.Less(t, len(td1.centroids), len(td2.centroids))
	})
}

func TestAccuracy(t *testing.T) {
	t.Run("accuracy for uniform distribution", func(t *testing.T) {
		td := New(100)
		n := 1000

		for i := 0; i < n; i++ {
			td.Add(float64(i))
		}

		// Test common quantiles (skip extreme edges for smaller sample)
		quantiles := []float64{0.1, 0.25, 0.5, 0.75, 0.9, 0.95, 0.99}

		for _, q := range quantiles {
			expected := q * float64(n)
			actual := td.Quantile(q)
			relativeError := math.Abs(actual-expected) / expected

			// Allow 3% relative error for smaller sample size
			assert.Less(t, relativeError, 0.03,
				"Quantile %.2f: expected %.2f, got %.2f, error %.4f%%",
				q, expected, actual, relativeError*100)
		}
	})

	t.Run("accuracy for normal distribution", func(t *testing.T) {
		td := New(100)
		r := rand.New(rand.NewSource(42))

		values := make([]float64, 1000)
		for i := range values {
			values[i] = r.NormFloat64()*10 + 100
			td.Add(values[i])
		}

		// Sort for true quantiles
		sort.Float64s(values)

		quantiles := []float64{0.5, 0.9, 0.95, 0.99}

		for _, q := range quantiles {
			idx := int(q * float64(len(values)))
			if idx >= len(values) {
				idx = len(values) - 1
			}
			expected := values[idx]
			actual := td.Quantile(q)

			// Allow 5% relative error for normal distribution
			relativeError := math.Abs(actual-expected) / math.Abs(expected)
			assert.Less(t, relativeError, 0.05,
				"Quantile %.2f: expected %.2f, got %.2f, error %.4f%%",
				q, expected, actual, relativeError*100)
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("very large values", func(t *testing.T) {
		td := New(100)
		td.Add(1e308)
		td.Add(1e307)

		assert.False(t, math.IsInf(td.Quantile(0.5), 0))
	})

	t.Run("very small values", func(t *testing.T) {
		td := New(100)
		td.Add(1e-308)
		td.Add(1e-307)

		assert.Greater(t, td.Quantile(0.5), 0.0)
	})

	t.Run("negative values", func(t *testing.T) {
		td := New(100)
		for i := -50; i < 50; i++ {
			td.Add(float64(i))
		}

		assert.InDelta(t, -0.5, td.Mean(), 1.0)
		assert.InDelta(t, 0, td.Quantile(0.5), 2.0)
	})

	t.Run("duplicate values", func(t *testing.T) {
		td := New(100)
		for i := 0; i < 1000; i++ {
			td.Add(42)
		}

		assert.Equal(t, 42.0, td.Quantile(0.5))
		assert.Equal(t, 42.0, td.Mean())
	})
}

// Benchmarks

func BenchmarkAdd(b *testing.B) {
	td := New(100)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		td.Add(float64(i))
	}
}

func BenchmarkQuantile(b *testing.B) {
	td := New(100)
	for i := 0; i < 1000; i++ {
		td.Add(float64(i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = td.Quantile(0.95)
	}
}

func BenchmarkCDF(b *testing.B) {
	td := New(100)
	for i := 0; i < 1000; i++ {
		td.Add(float64(i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = td.CDF(500)
	}
}

func BenchmarkMerge(b *testing.B) {
	td1 := New(100)
	for i := 0; i < 500; i++ {
		td1.Add(float64(i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		td2 := New(100)
		for j := 500; j < 1000; j++ {
			td2.Add(float64(j))
		}
		td1.Merge(td2)
	}
}

func BenchmarkExport(b *testing.B) {
	td := New(100)
	for i := 0; i < 1000; i++ {
		td.Add(float64(i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = td.Export()
	}
}

func BenchmarkImport(b *testing.B) {
	td := New(100)
	for i := 0; i < 1000; i++ {
		td.Add(float64(i))
	}
	data, _ := td.Export()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Import(data)
	}
}
