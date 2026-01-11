package xentropy

import (
	"crypto/rand"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShannon(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		entropy := Shannon([]byte{})
		assert.Equal(t, 0.0, entropy)
	})

	t.Run("single byte", func(t *testing.T) {
		entropy := Shannon([]byte{'a'})
		assert.Equal(t, 0.0, entropy)
	})

	t.Run("all same bytes", func(t *testing.T) {
		data := []byte("aaaaaaaaaa")
		entropy := Shannon(data)
		assert.Equal(t, 0.0, entropy)
	})

	t.Run("two different bytes equal frequency", func(t *testing.T) {
		data := []byte("aabb")
		entropy := Shannon(data)
		assert.InDelta(t, 1.0, entropy, 0.0001) // log2(2) = 1
	})

	t.Run("perfectly balanced 4 symbols", func(t *testing.T) {
		data := []byte("aabbccdd")
		entropy := Shannon(data)
		assert.InDelta(t, 2.0, entropy, 0.0001) // log2(4) = 2
	})

	t.Run("binary data", func(t *testing.T) {
		data := []byte{0, 1, 0, 1, 0, 1, 0, 1}
		entropy := Shannon(data)
		assert.InDelta(t, 1.0, entropy, 0.0001)
	})

	t.Run("all 256 bytes once", func(t *testing.T) {
		data := make([]byte, 256)
		for i := 0; i < 256; i++ {
			data[i] = byte(i)
		}
		entropy := Shannon(data)
		assert.InDelta(t, 8.0, entropy, 0.0001) // log2(256) = 8
	})

	t.Run("unbalanced distribution", func(t *testing.T) {
		data := []byte("aaab")
		entropy := Shannon(data)
		// P(a) = 3/4, P(b) = 1/4
		// H = -3/4 * log2(3/4) - 1/4 * log2(1/4)
		expected := -(3.0/4.0)*math.Log2(3.0/4.0) - (1.0/4.0)*math.Log2(1.0/4.0)
		assert.InDelta(t, expected, entropy, 0.0001)
	})
}

func TestNormalized(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		norm := Normalized([]byte{})
		assert.Equal(t, 0.0, norm)
	})

	t.Run("single unique byte", func(t *testing.T) {
		norm := Normalized([]byte("aaaaa"))
		assert.Equal(t, 0.0, norm) // No entropy with one symbol
	})

	t.Run("two symbols perfectly balanced", func(t *testing.T) {
		norm := Normalized([]byte("aabb"))
		assert.InDelta(t, 1.0, norm, 0.0001) // Maximum entropy for 2 symbols
	})

	t.Run("four symbols perfectly balanced", func(t *testing.T) {
		norm := Normalized([]byte("aabbccdd"))
		assert.InDelta(t, 1.0, norm, 0.0001) // Maximum entropy for 4 symbols
	})

	t.Run("unbalanced distribution", func(t *testing.T) {
		norm := Normalized([]byte("aaaab"))
		assert.Greater(t, norm, 0.0)
		assert.Less(t, norm, 1.0)
	})

	t.Run("256 unique bytes", func(t *testing.T) {
		data := make([]byte, 256)
		for i := 0; i < 256; i++ {
			data[i] = byte(i)
		}
		norm := Normalized(data)
		assert.InDelta(t, 1.0, norm, 0.0001)
	})
}

func TestMetric(t *testing.T) {
	t.Run("no entropy", func(t *testing.T) {
		metric := Metric([]byte("aaaaa"))
		assert.Equal(t, 0.0, metric)
	})

	t.Run("maximum entropy", func(t *testing.T) {
		metric := Metric([]byte("aabb"))
		assert.InDelta(t, 100.0, metric, 0.1)
	})

	t.Run("partial entropy", func(t *testing.T) {
		metric := Metric([]byte("aaaab"))
		assert.Greater(t, metric, 0.0)
		assert.Less(t, metric, 100.0)
	})
}

//nolint:dupl // Test code duplication with IsSecure is intentional for clarity
func TestIsRandom(t *testing.T) {
	t.Run("perfectly random data", func(t *testing.T) {
		data := make([]byte, 256)
		for i := 0; i < 256; i++ {
			data[i] = byte(i)
		}
		assert.True(t, IsRandom(data, 0.9))
	})

	t.Run("not random - all same", func(t *testing.T) {
		data := []byte(strings.Repeat("a", 100))
		assert.False(t, IsRandom(data, 0.9))
	})

	t.Run("not random - imbalanced pattern", func(t *testing.T) {
		data := []byte(strings.Repeat("aaab", 25))
		// Imbalanced distribution should have lower normalized entropy
		assert.False(t, IsRandom(data, 0.9))
	})

	t.Run("crypto random", func(t *testing.T) {
		data := make([]byte, 256)
		_, err := rand.Read(data)
		require.NoError(t, err)
		// Crypto random should have high entropy
		assert.True(t, IsRandom(data, 0.85))
	})

	t.Run("default threshold", func(t *testing.T) {
		data := []byte(strings.Repeat("a", 100))
		assert.False(t, IsRandom(data, 0))
	})

	t.Run("custom threshold", func(t *testing.T) {
		data := []byte("aabb")
		assert.True(t, IsRandom(data, 0.5))
		assert.True(t, IsRandom(data, 0.99)) // Perfect entropy for 2 symbols

		// Test with unbalanced data
		unbalanced := []byte("aaaab")
		assert.True(t, IsRandom(unbalanced, 0.5))
		assert.False(t, IsRandom(unbalanced, 0.99))
	})
}

func TestRealWorldExamples(t *testing.T) {
	t.Run("english text", func(t *testing.T) {
		text := []byte("The quick brown fox jumps over the lazy dog")
		entropy := Shannon(text)
		// English text typically has entropy between 1-5 bits
		assert.Greater(t, entropy, 1.0)
		assert.Less(t, entropy, 6.0)
	})

	t.Run("base64 encoded", func(t *testing.T) {
		base64 := []byte("SGVsbG8gV29ybGQh")
		entropy := Shannon(base64)
		// Base64 data typically has moderate entropy
		assert.Greater(t, entropy, 3.0)
		assert.Less(t, entropy, 5.0)
	})

	t.Run("hex string", func(t *testing.T) {
		hex := []byte("deadbeef1234567890abcdef")
		entropy := Shannon(hex)
		// Hex has 16 possible characters
		assert.Greater(t, entropy, 3.0)
	})

	t.Run("json data", func(t *testing.T) {
		json := []byte(`{"name":"test","value":123,"enabled":true}`)
		entropy := Shannon(json)
		assert.Greater(t, entropy, 2.0)
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("very large identical data", func(t *testing.T) {
		data := []byte(strings.Repeat("a", 100000))
		entropy := Shannon(data)
		assert.Equal(t, 0.0, entropy)
	})

	t.Run("alternating pattern", func(t *testing.T) {
		data := []byte(strings.Repeat("01", 50))
		entropy := Shannon(data)
		assert.InDelta(t, 1.0, entropy, 0.0001)
	})

	t.Run("binary zeros and ones", func(t *testing.T) {
		data := []byte{0, 1, 0, 1, 0, 1}
		entropy := Shannon(data)
		assert.InDelta(t, 1.0, entropy, 0.0001)
	})
}

func BenchmarkShannon(b *testing.B) {
	data := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 10))
	b.ReportAllocs()
	for b.Loop() {
		Shannon(data)
	}
}

func BenchmarkNormalized(b *testing.B) {
	data := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 10))
	b.ReportAllocs()
	for b.Loop() {
		Normalized(data)
	}
}

func BenchmarkIsRandom(b *testing.B) {
	data := make([]byte, 256)
	for i := range 256 {
		data[i] = byte(i)
	}
	b.ReportAllocs()
	for b.Loop() {
		IsRandom(data, 0.9)
	}
}

func FuzzShannon(f *testing.F) {
	f.Add([]byte("test"))
	f.Add([]byte("hello world"))
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x01, 0x02})

	f.Fuzz(func(t *testing.T, data []byte) {
		entropy := Shannon(data)
		if len(data) > 0 && (entropy < 0 || entropy > 8) {
			t.Errorf("entropy should be between 0 and 8, got %f", entropy)
		}
	})
}

func FuzzNormalized(f *testing.F) {
	f.Add([]byte("test"))
	f.Add([]byte("hello world"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		entropy := Normalized(data)
		if len(data) > 0 && (entropy < 0 || entropy > 1) {
			t.Errorf("normalized entropy should be between 0 and 1, got %f", entropy)
		}
	})
}
