package xdigits

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRound64(t *testing.T) {
	t.Run("rounds to 2 decimals", func(t *testing.T) {
		result := Round64(12.3456, 2)
		assert.InDelta(t, 12.35, result, 0.001)
	})

	t.Run("rounds down", func(t *testing.T) {
		result := Round64(12.344, 2)
		assert.InDelta(t, 12.34, result, 0.001)
	})

	t.Run("zero precision", func(t *testing.T) {
		result := Round64(12.5, 0)
		assert.InDelta(t, 13.0, result, 0.001)
	})
}

func TestRound64Up(t *testing.T) {
	t.Run("rounds up to 2 decimals", func(t *testing.T) {
		result := Round64Up(12.3416, 2)
		assert.InDelta(t, 12.35, result, 0.001)
	})

	t.Run("already at boundary", func(t *testing.T) {
		result := Round64Up(12.34, 2)
		assert.InDelta(t, 12.34, result, 0.001)
	})
}

func TestRound64Down(t *testing.T) {
	t.Run("rounds down to 2 decimals", func(t *testing.T) {
		result := Round64Down(12.3496, 2)
		assert.InDelta(t, 12.34, result, 0.001)
	})

	t.Run("already at boundary", func(t *testing.T) {
		result := Round64Down(12.34, 2)
		assert.InDelta(t, 12.34, result, 0.001)
	})
}

func BenchmarkRound64(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Round64(12.3456, 2)
	}
}

func BenchmarkRound64Up(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Round64Up(12.3416, 2)
	}
}

func BenchmarkRound64Down(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Round64Down(12.3496, 2)
	}
}

func FuzzRound64(f *testing.F) {
	f.Add(12.3456, 2)
	f.Add(0.0, 0)
	f.Add(-12.345, 2)
	f.Add(100.5, 1)

	f.Fuzz(func(_ *testing.T, val float64, precision int) {
		if precision < 0 || precision > 10 {
			return
		}
		_ = Round64(val, precision)
	})
}
