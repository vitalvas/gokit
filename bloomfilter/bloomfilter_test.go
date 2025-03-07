package bloomfilter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBloomFilter(t *testing.T) {
	tests := []struct {
		n         uint
		p         float64
		expectedM uint
		expectedK uint
	}{
		{500, 0.01, 4793, 7},
		{500, 0.001, 7189, 10},
		{1000, 0.01, 9586, 7},
		{1000, 0.001, 14378, 10},
		{1_000_000, 0.01, 9585059, 7},
		{10_000_000, 0.01, 95850584, 7},
	}

	for _, test := range tests {
		bf := NewBloomFilter(test.n, test.p)
		assert.Equal(t, test.expectedM, bf.Size, "NewBloomFilter(%d, %f) size", test.n, test.p)
		assert.Equal(t, test.expectedK, uint(len(bf.hashFuncs)), "NewBloomFilter(%d, %f) hashFuncs", test.n, test.p)
	}
}

func TestBloomFilter_Add(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)

	bf.Add("test")
	assert.True(t, bf.Contains("test"), "BloomFilter should contain 'test' after adding it")

	elements := []string{"foo", "bar", "baz"}
	for _, element := range elements {
		bf.Add(element)
	}

	for _, element := range elements {
		assert.True(t, bf.Contains(element), "BloomFilter should contain '%s' after adding it", element)
	}

	assert.False(t, bf.Contains("not_added"), "BloomFilter should not contain 'not_added' as it was never added")
}

func TestOptimalM(t *testing.T) {
	tests := []struct {
		n        uint
		p        float64
		expected uint
	}{
		{1000, 0.01, 9586},
		{1000, 0.001, 14378},
		{500, 0.01, 4793},
		{500, 0.001, 7189},
	}

	for _, test := range tests {
		result := optimalM(test.n, test.p)
		assert.Equal(t, test.expected, result, "optimalM(%d, %f)", test.n, test.p)
	}
}

func TestOptimalK(t *testing.T) {
	tests := []struct {
		n        uint
		m        uint
		expected uint
	}{
		{1000, 9586, 7},
		{1000, 14378, 10},
		{500, 4793, 7},
		{500, 7189, 10},
	}

	for _, test := range tests {
		result := optimalK(test.n, test.m)
		assert.Equal(t, test.expected, result, "optimalK(%d, %d)", test.n, test.m)
	}
}
func BenchmarkBloomFilter_Add(b *testing.B) {
	b.Run("1m", func(b *testing.B) {
		b.ReportAllocs()

		bf := NewBloomFilter(1_000_000, 0.01)

		for i := 0; i < b.N; i++ {
			bf.Add(fmt.Sprintf("test-%d", i))
		}
	})

	b.Run("10m", func(b *testing.B) {
		b.ReportAllocs()

		bf := NewBloomFilter(10_000_000, 0.01)

		for i := 0; i < b.N; i++ {
			bf.Add(fmt.Sprintf("test-%d", i))
		}
	})

	b.Run("50m", func(b *testing.B) {
		b.ReportAllocs()

		bf := NewBloomFilter(50_000_000, 0.01)

		for i := 0; i < b.N; i++ {
			bf.Add(fmt.Sprintf("test-%d", i))
		}
	})
}

func BenchmarkBloomFilter_Contains(b *testing.B) {
	b.Run("1m", func(b *testing.B) {
		b.ReportAllocs()

		bf := NewBloomFilter(1_000_000, 0.01)

		for i := 0; i < 1_000_000; i++ {
			bf.Add(fmt.Sprintf("test-%d", i))
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			bf.Contains(fmt.Sprintf("test-%d", i%1_000_000))
		}
	})

	b.Run("10m", func(b *testing.B) {
		b.ReportAllocs()

		bf := NewBloomFilter(10_000_000, 0.01)

		for i := 0; i < 10_000_000; i++ {
			bf.Add(fmt.Sprintf("test-%d", i))
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			bf.Contains(fmt.Sprintf("test-%d", i%1_000_000))
		}
	})
}

func TestBloomFilter_ExportImport(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)

	elements := []string{"foo", "bar", "baz"}
	for _, element := range elements {
		bf.Add(element)
	}

	data, err := bf.Export()
	assert.NoError(t, err, "Export should not return an error")

	importedBf, err := ImportBloomFilter(data)
	assert.NoError(t, err, "ImportBloomFilter should not return an error")

	for _, element := range elements {
		assert.True(t, importedBf.Contains(element), "Imported BloomFilter should contain '%s'", element)
	}

	assert.False(t, importedBf.Contains("not_added"), "Imported BloomFilter should not contain 'not_added' as it was never added")

	assert.Equal(t, bf.Size, importedBf.Size, "Imported BloomFilter should have the same size")
	assert.Equal(t, bf.K, importedBf.K, "Imported BloomFilter should have the same K")

}
