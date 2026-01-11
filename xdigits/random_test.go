package xdigits

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandInt(t *testing.T) {
	t.Run("returns value in range", func(t *testing.T) {
		for range 100 {
			result := RandInt(10, 20)
			assert.GreaterOrEqual(t, result, 10)
			assert.Less(t, result, 20)
		}
	})

	t.Run("min equals max minus 1", func(t *testing.T) {
		result := RandInt(5, 6)
		assert.Equal(t, 5, result)
	})
}

func BenchmarkRandInt(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = RandInt(1, 100)
	}
}

func FuzzRandInt(f *testing.F) {
	f.Add(0, 10)
	f.Add(1, 100)
	f.Add(-100, 100)

	f.Fuzz(func(t *testing.T, minVal, maxVal int) {
		if maxVal <= minVal {
			return
		}
		result := RandInt(minVal, maxVal)
		if result < minVal || result >= maxVal {
			t.Errorf("result %d out of range [%d, %d)", result, minVal, maxVal)
		}
	})
}
