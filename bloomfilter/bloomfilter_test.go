package bloomfilter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBloomFilter(t *testing.T) {
	tests := []struct {
		n         uint
		p         float64
		expectMinM uint64 // Minimum expected size due to power-of-2 rounding
		expectK   uint32
	}{
		{500, 0.01, 8192, 7},     // Will be rounded up to next power of 2
		{500, 0.001, 8192, 10},
		{1000, 0.01, 16384, 7},
		{1000, 0.001, 16384, 10},
		{1_000_000, 0.01, 16777216, 7}, // 2^24
		{10_000_000, 0.01, 134217728, 7}, // 2^27
	}

	for _, test := range tests {
		bf := NewBloomFilter(test.n, test.p)
		assert.GreaterOrEqual(t, bf.Size(), test.expectMinM, "NewBloomFilter(%d, %f) size should be at least %d", test.n, test.p, test.expectMinM)
		assert.Equal(t, test.expectK, bf.K(), "NewBloomFilter(%d, %f) K", test.n, test.p)
		
		// Verify size is power of 2
		size := bf.Size()
		assert.Equal(t, uint64(0), size&(size-1), "Size should be power of 2")
		
		// Verify size is divisible by 64
		assert.Equal(t, uint64(0), size%64, "Size should be divisible by 64")
	}
}

func TestBloomFilter_Add(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)

	bf.Add("test")
	assert.True(t, bf.Contains("test"), "BloomFilter should contain 'test' after adding it")

	elements := []string{"foo", "bar", "baz", "hello", "world"}
	for _, element := range elements {
		bf.Add(element)
	}

	for _, element := range elements {
		assert.True(t, bf.Contains(element), "BloomFilter should contain '%s' after adding it", element)
	}

	assert.False(t, bf.Contains("not_added"), "BloomFilter should not contain 'not_added' as it was never added")
}

func TestBloomFilter_AddBytes(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)

	testData := []byte("test data")
	bf.AddBytes(testData)
	assert.True(t, bf.ContainsBytes(testData), "BloomFilter should contain test data after adding it")

	elements := [][]byte{
		[]byte("foo"),
		[]byte("bar"),
		[]byte("baz"),
		[]byte{0x00, 0x01, 0x02}, // Binary data
		[]byte(""),               // Empty data
	}

	for _, element := range elements {
		bf.AddBytes(element)
	}

	for _, element := range elements {
		assert.True(t, bf.ContainsBytes(element), "BloomFilter should contain bytes after adding them")
	}

	assert.False(t, bf.ContainsBytes([]byte("not_added")), "BloomFilter should not contain bytes that were never added")
}

func TestBloomFilter_StringVsBytes(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)
	
	testStr := "hello world"
	testBytes := []byte(testStr)
	
	// Add as string
	bf.Add(testStr)
	
	// Should be found as both string and bytes
	assert.True(t, bf.Contains(testStr), "Should contain string")
	assert.True(t, bf.ContainsBytes(testBytes), "Should contain equivalent bytes")
	
	bf.Clear()
	
	// Add as bytes
	bf.AddBytes(testBytes)
	
	// Should be found as both string and bytes
	assert.True(t, bf.Contains(testStr), "Should contain equivalent string")
	assert.True(t, bf.ContainsBytes(testBytes), "Should contain bytes")
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
		expected uint32
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

func TestBloomFilter_ExportImport(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)

	elements := []string{"foo", "bar", "baz", "hello", "world"}
	for _, element := range elements {
		bf.Add(element)
	}

	data, err := bf.Export()
	require.NoError(t, err, "Export should not return an error")

	importedBf, err := ImportBloomFilter(data)
	require.NoError(t, err, "ImportBloomFilter should not return an error")

	for _, element := range elements {
		assert.True(t, importedBf.Contains(element), "Imported BloomFilter should contain '%s'", element)
	}

	assert.False(t, importedBf.Contains("not_added"), "Imported BloomFilter should not contain 'not_added'")

	assert.Equal(t, bf.Size(), importedBf.Size(), "Imported BloomFilter should have the same size")
	assert.Equal(t, bf.K(), importedBf.K(), "Imported BloomFilter should have the same K")
}

func TestBloomFilter_EstimatedCount(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)
	
	// Empty filter
	assert.Equal(t, uint64(0), bf.EstimatedCount(), "Empty filter should have count 0")
	
	// Add some elements
	elements := []string{"a", "b", "c", "d", "e"}
	for _, elem := range elements {
		bf.Add(elem)
	}
	
	count := bf.EstimatedCount()
	// Should be roughly close to the number of elements added
	assert.Greater(t, count, uint64(0), "Should have non-zero count")
	assert.Less(t, count, uint64(100), "Should not be too large")
	
	// Clear and verify
	bf.Clear()
	assert.Equal(t, uint64(0), bf.EstimatedCount(), "Cleared filter should have count 0")
}

func TestBloomFilter_Clear(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)
	
	// Add elements
	bf.Add("test1")
	bf.Add("test2")
	assert.True(t, bf.Contains("test1"))
	assert.True(t, bf.Contains("test2"))
	
	// Clear
	bf.Clear()
	assert.False(t, bf.Contains("test1"))
	assert.False(t, bf.Contains("test2"))
	assert.Equal(t, uint64(0), bf.EstimatedCount())
}

func TestNextPowerOf2(t *testing.T) {
	tests := []struct {
		input    uint
		expected uint64
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{7, 8},
		{8, 8},
		{9, 16},
		{1000, 1024},
		{1024, 1024},
		{1025, 2048},
	}
	
	for _, test := range tests {
		result := nextPowerOf2(test.input)
		assert.Equal(t, test.expected, result, "nextPowerOf2(%d)", test.input)
	}
}

func TestBloomFilter_FalsePositiveRate(t *testing.T) {
	// Test with a small filter to get measurable false positive rate
	bf := NewBloomFilter(100, 0.1) // 10% false positive rate
	
	// Add 100 elements
	added := make(map[string]bool)
	for i := 0; i < 100; i++ {
		elem := fmt.Sprintf("element-%d", i)
		bf.Add(elem)
		added[elem] = true
	}
	
	// Test 1000 random elements not in the set
	falsePositives := 0
	total := 1000
	
	for i := 100; i < 100+total; i++ {
		elem := fmt.Sprintf("element-%d", i)
		if bf.Contains(elem) {
			falsePositives++
		}
	}
	
	falsePositiveRate := float64(falsePositives) / float64(total)
	
	// Should be roughly around 10% but allow some variance
	assert.Less(t, falsePositiveRate, 0.2, "False positive rate should be less than 20%")
	t.Logf("False positive rate: %.2f%% (expected ~10%%)", falsePositiveRate*100)
}

// Benchmark tests for performance comparison
func BenchmarkBloomFilter_Add(b *testing.B) {
	benchmarks := []struct {
		name string
		n    uint
	}{
		{"1M", 1_000_000},
		{"10M", 10_000_000},
		{"50M", 50_000_000},
	}
	
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			bf := NewBloomFilter(bm.n, 0.01)
			b.ReportAllocs()
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				bf.Add(fmt.Sprintf("test-%d", i))
			}
		})
	}
}

func BenchmarkBloomFilter_AddBytes(b *testing.B) {
	bf := NewBloomFilter(1_000_000, 0.01)
	data := []byte("benchmark-test-data")
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		bf.AddBytes(data)
	}
}

func BenchmarkBloomFilter_Contains(b *testing.B) {
	benchmarks := []struct {
		name string
		n    uint
	}{
		{"1M", 1_000_000},
		{"10M", 10_000_000},
	}
	
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			bf := NewBloomFilter(bm.n, 0.01)
			
			// Pre-populate
			for i := 0; i < int(bm.n); i++ {
				bf.Add(fmt.Sprintf("test-%d", i))
			}
			
			b.ReportAllocs()
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				bf.Contains(fmt.Sprintf("test-%d", i%int(bm.n)))
			}
		})
	}
}

func BenchmarkBloomFilter_ContainsBytes(b *testing.B) {
	bf := NewBloomFilter(1_000_000, 0.01)
	data := []byte("benchmark-test-data")
	bf.AddBytes(data)
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		bf.ContainsBytes(data)
	}
}

func BenchmarkHash(b *testing.B) {
	bf := NewBloomFilter(1000, 0.01)
	testStr := "benchmark test string for hashing performance"
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		bf.hash(testStr)
	}
}

func BenchmarkHashBytes(b *testing.B) {
	bf := NewBloomFilter(1000, 0.01)
	testData := []byte("benchmark test string for hashing performance")
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		bf.hashBytes(testData)
	}
}