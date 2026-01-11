package xentropy

import (
	"crypto/rand"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinEntropy(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		entropy := MinEntropy([]byte{})
		assert.Equal(t, 0.0, entropy)
	})

	t.Run("single byte", func(t *testing.T) {
		entropy := MinEntropy([]byte{'a'})
		assert.Equal(t, 0.0, entropy)
	})

	t.Run("all same bytes", func(t *testing.T) {
		data := []byte("aaaaaaaaaa")
		entropy := MinEntropy(data)
		assert.Equal(t, 0.0, entropy)
	})

	t.Run("two different bytes equal frequency", func(t *testing.T) {
		data := []byte("aabb")
		entropy := MinEntropy(data)
		// max probability = 2/4 = 0.5, -log2(0.5) = 1
		assert.InDelta(t, 1.0, entropy, 0.0001)
	})

	t.Run("perfectly balanced 4 symbols", func(t *testing.T) {
		data := []byte("aabbccdd")
		entropy := MinEntropy(data)
		// max probability = 2/8 = 0.25, -log2(0.25) = 2
		assert.InDelta(t, 2.0, entropy, 0.0001)
	})

	t.Run("unbalanced distribution", func(t *testing.T) {
		data := []byte("aaab")
		entropy := MinEntropy(data)
		// max probability = 3/4 = 0.75, -log2(0.75) ≈ 0.415
		expected := -math.Log2(3.0 / 4.0)
		assert.InDelta(t, expected, entropy, 0.0001)
	})

	t.Run("highly skewed distribution", func(t *testing.T) {
		data := []byte("aaaaaaaaab")
		entropy := MinEntropy(data)
		// max probability = 9/10 = 0.9, -log2(0.9) ≈ 0.152
		expected := -math.Log2(9.0 / 10.0)
		assert.InDelta(t, expected, entropy, 0.0001)
	})

	t.Run("all 256 bytes once", func(t *testing.T) {
		data := make([]byte, 256)
		for i := 0; i < 256; i++ {
			data[i] = byte(i)
		}
		entropy := MinEntropy(data)
		// max probability = 1/256, -log2(1/256) = 8
		assert.InDelta(t, 8.0, entropy, 0.0001)
	})

	t.Run("compare with Shannon entropy", func(t *testing.T) {
		// For skewed data, min-entropy should be lower than Shannon
		data := []byte("aaaaaaaaab")
		minEnt := MinEntropy(data)
		shannon := Shannon(data)
		assert.Less(t, minEnt, shannon)

		// For uniform data, they should be equal
		uniform := []byte("aabb")
		minEntUniform := MinEntropy(uniform)
		shannonUniform := Shannon(uniform)
		assert.InDelta(t, shannonUniform, minEntUniform, 0.0001)
	})
}

func TestMinNormalized(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		norm := MinNormalized([]byte{})
		assert.Equal(t, 0.0, norm)
	})

	t.Run("single unique byte", func(t *testing.T) {
		norm := MinNormalized([]byte("aaaaa"))
		assert.Equal(t, 0.0, norm)
	})

	t.Run("two symbols perfectly balanced", func(t *testing.T) {
		norm := MinNormalized([]byte("aabb"))
		assert.InDelta(t, 1.0, norm, 0.0001)
	})

	t.Run("four symbols perfectly balanced", func(t *testing.T) {
		norm := MinNormalized([]byte("aabbccdd"))
		assert.InDelta(t, 1.0, norm, 0.0001)
	})

	t.Run("unbalanced distribution", func(t *testing.T) {
		norm := MinNormalized([]byte("aaaab"))
		assert.Greater(t, norm, 0.0)
		assert.Less(t, norm, 1.0)
	})

	t.Run("256 unique bytes", func(t *testing.T) {
		data := make([]byte, 256)
		for i := 0; i < 256; i++ {
			data[i] = byte(i)
		}
		norm := MinNormalized(data)
		assert.InDelta(t, 1.0, norm, 0.0001)
	})
}

func TestMinMetric(t *testing.T) {
	t.Run("no entropy", func(t *testing.T) {
		metric := MinMetric([]byte("aaaaa"))
		assert.Equal(t, 0.0, metric)
	})

	t.Run("maximum entropy", func(t *testing.T) {
		metric := MinMetric([]byte("aabb"))
		assert.InDelta(t, 100.0, metric, 0.1)
	})

	t.Run("partial entropy", func(t *testing.T) {
		metric := MinMetric([]byte("aaaab"))
		assert.Greater(t, metric, 0.0)
		assert.Less(t, metric, 100.0)
	})
}

//nolint:dupl // Test code duplication with IsRandom is intentional for clarity
func TestIsSecure(t *testing.T) {
	t.Run("perfectly uniform data", func(t *testing.T) {
		data := make([]byte, 256)
		for i := 0; i < 256; i++ {
			data[i] = byte(i)
		}
		assert.True(t, IsSecure(data, 0.8))
	})

	t.Run("not secure - all same", func(t *testing.T) {
		data := []byte(strings.Repeat("a", 100))
		assert.False(t, IsSecure(data, 0.8))
	})

	t.Run("not secure - highly skewed", func(t *testing.T) {
		data := []byte(strings.Repeat("aaab", 25))
		assert.False(t, IsSecure(data, 0.8))
	})

	t.Run("crypto random", func(t *testing.T) {
		data := make([]byte, 256)
		_, err := rand.Read(data)
		require.NoError(t, err)
		// Crypto random should have good min-entropy
		assert.True(t, IsSecure(data, 0.7))
	})

	t.Run("default threshold", func(t *testing.T) {
		data := []byte(strings.Repeat("a", 100))
		assert.False(t, IsSecure(data, 0))
	})

	t.Run("custom threshold", func(t *testing.T) {
		data := []byte("aabb")
		assert.True(t, IsSecure(data, 0.5))
		assert.True(t, IsSecure(data, 0.9))

		// Test with unbalanced data (min normalized ≈ 0.32)
		unbalanced := []byte("aaaab")
		assert.True(t, IsSecure(unbalanced, 0.3))
		assert.False(t, IsSecure(unbalanced, 0.4))
	})
}

func TestMinEntropyRealWorld(t *testing.T) {
	t.Run("password comparison", func(t *testing.T) {
		// Weak password with repeated chars
		weak := []byte("password")
		weakMin := MinEntropy(weak)
		weakShannon := Shannon(weak)
		// Min-entropy should be lower for non-uniform distribution
		assert.LessOrEqual(t, weakMin, weakShannon)

		// Strong password with varied chars
		strong := []byte("aB3$xZ9@mK2#")
		strongMin := MinEntropy(strong)
		strongShannon := Shannon(strong)
		assert.LessOrEqual(t, strongMin, strongShannon)
	})

	t.Run("cryptographic key quality", func(t *testing.T) {
		// Good key - should have high min-entropy
		goodKey := make([]byte, 32)
		_, err := rand.Read(goodKey)
		require.NoError(t, err)
		minEnt := MinEntropy(goodKey)
		assert.Greater(t, minEnt, 3.0) // Should be reasonably high

		// Bad key - low min-entropy
		badKey := []byte(strings.Repeat("abcd", 8))
		minEntBad := MinEntropy(badKey)
		assert.Less(t, minEntBad, 3.0) // Should be low
	})

	t.Run("english text", func(t *testing.T) {
		text := []byte("The quick brown fox jumps over the lazy dog")
		minEnt := MinEntropy(text)
		shannon := Shannon(text)
		// For natural text, min-entropy should be noticeably lower
		assert.Less(t, minEnt, shannon)
	})
}

func TestMinEntropyEdgeCases(t *testing.T) {
	t.Run("very large identical data", func(t *testing.T) {
		data := []byte(strings.Repeat("a", 100000))
		entropy := MinEntropy(data)
		assert.Equal(t, 0.0, entropy)
	})

	t.Run("binary pattern", func(t *testing.T) {
		data := []byte{0, 1, 0, 1, 0, 1}
		entropy := MinEntropy(data)
		assert.InDelta(t, 1.0, entropy, 0.0001)
	})

	t.Run("single outlier", func(t *testing.T) {
		data := []byte(strings.Repeat("a", 99) + "b")
		entropy := MinEntropy(data)
		expected := -math.Log2(99.0 / 100.0)
		assert.InDelta(t, expected, entropy, 0.0001)
	})
}

// Benchmarks

//nolint:dupl // Benchmark structure similar to Shannon benchmarks is intentional
func BenchmarkMinEntropy(b *testing.B) {
	small := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 2))
	medium := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 12))
	large := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 120))

	b.Run("Small_100B", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			MinEntropy(small)
		}
	})

	b.Run("Medium_500B", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			MinEntropy(medium)
		}
	})

	b.Run("Large_5KB", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			MinEntropy(large)
		}
	})
}

func BenchmarkMinNormalized(b *testing.B) {
	data := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 10))

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		MinNormalized(data)
	}
}

func BenchmarkIsSecure(b *testing.B) {
	data := make([]byte, 256)
	for i := range 256 {
		data[i] = byte(i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		IsSecure(data, 0.8)
	}
}

func FuzzMinEntropy(f *testing.F) {
	f.Add([]byte("hello world"))
	f.Add([]byte("aaaaaaa"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, data []byte) {
		entropy := MinEntropy(data)
		if entropy < 0 {
			t.Errorf("entropy should be non-negative, got %f", entropy)
		}
		if entropy > 8 {
			t.Errorf("entropy should not exceed 8 bits, got %f", entropy)
		}
	})
}
