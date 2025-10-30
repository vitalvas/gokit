package hyperloglog

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRate(t *testing.T) {
	t.Run("create with custom half-life", func(t *testing.T) {
		r := NewRate(30 * time.Second)
		assert.NotNil(t, r)
		assert.Equal(t, 30*time.Second, r.HalfLife())
	})

	t.Run("create with default half-life for zero", func(t *testing.T) {
		r := NewRate(0)
		assert.Equal(t, 60*time.Second, r.HalfLife())
	})

	t.Run("create with default half-life for negative", func(t *testing.T) {
		r := NewRate(-10 * time.Second)
		assert.Equal(t, 60*time.Second, r.HalfLife())
	})
}

func TestRateAdd(t *testing.T) {
	t.Run("add single unique item", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		r.AddAt(1.0, now)
		rate := r.RateAt(now)

		assert.Equal(t, 1.0, rate)
	})

	t.Run("add multiple unique items", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		r.AddAt(5.0, now)
		rate := r.RateAt(now)

		assert.Equal(t, 5.0, rate)
	})

	t.Run("add accumulates with same timestamp", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		r.AddAt(3.0, now)
		r.AddAt(2.0, now)
		rate := r.RateAt(now)

		assert.Equal(t, 5.0, rate)
	})

	t.Run("add with decay over time", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		r.AddAt(10.0, now)
		rate1 := r.RateAt(now)

		// Add after 60 seconds (one half-life)
		r.AddAt(2.0, now.Add(60*time.Second))
		rate2 := r.RateAt(now.Add(60 * time.Second))

		// rate2 should be: 10 * 0.5 + 2 = 7
		assert.Equal(t, 10.0, rate1)
		assert.InDelta(t, 7.0, rate2, 0.01)
	})
}

func TestRateDecay(t *testing.T) {
	t.Run("rate decays over time", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		r.AddAt(10.0, now)

		// Check rate immediately
		rate0 := r.RateAt(now)
		assert.Equal(t, 10.0, rate0)

		// Check after 60 seconds (one half-life)
		rate1 := r.RateAt(now.Add(60 * time.Second))
		assert.InDelta(t, 5.0, rate1, 0.01)

		// Check after 120 seconds (two half-lives)
		rate2 := r.RateAt(now.Add(120 * time.Second))
		assert.InDelta(t, 2.5, rate2, 0.01)

		// Check after 180 seconds (three half-lives)
		rate3 := r.RateAt(now.Add(180 * time.Second))
		assert.InDelta(t, 1.25, rate3, 0.01)
	})

	t.Run("short half-life decays faster", func(t *testing.T) {
		rShort := NewRate(10 * time.Second)
		rLong := NewRate(60 * time.Second)
		now := time.Now()

		rShort.AddAt(10.0, now)
		rLong.AddAt(10.0, now)

		// After 30 seconds
		t30 := now.Add(30 * time.Second)
		rateShort := rShort.RateAt(t30)
		rateLong := rLong.RateAt(t30)

		// Short half-life should have decayed more
		assert.Less(t, rateShort, rateLong)
	})

	t.Run("rate approaches zero over many half-lives", func(t *testing.T) {
		r := NewRate(10 * time.Second)
		now := time.Now()

		r.AddAt(100.0, now)

		// After 10 half-lives (100 seconds)
		rate := r.RateAt(now.Add(100 * time.Second))

		// Should be very close to zero (100 * 0.5^10 ≈ 0.0977)
		assert.Less(t, rate, 1.0)
	})
}

func TestRateQuery(t *testing.T) {
	t.Run("rate returns current rate with automatic decay", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		r.Add(10.0)

		// Small delay
		time.Sleep(10 * time.Millisecond)

		rate := r.Rate()
		// Should be slightly less than 10 due to decay
		assert.Less(t, rate, 10.0)
		assert.Greater(t, rate, 9.9)
	})

	t.Run("initial rate is zero", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		assert.Equal(t, 0.0, r.Rate())
	})

	t.Run("rate at earlier time returns last rate", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		r.AddAt(10.0, now)

		// Query at earlier time
		rate := r.RateAt(now.Add(-10 * time.Second))
		assert.Equal(t, 10.0, rate)
	})
}

func TestRateSet(t *testing.T) {
	t.Run("set initializes rate", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		r.SetAt(42.0, now)
		rate := r.RateAt(now)

		assert.Equal(t, 42.0, rate)
	})

	t.Run("set overwrites existing rate", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		r.AddAt(10.0, now)
		r.SetAt(50.0, now)
		rate := r.RateAt(now)

		assert.Equal(t, 50.0, rate)
	})

	t.Run("set with automatic decay", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		r.SetAt(10.0, now)

		// Check after one half-life
		rate := r.RateAt(now.Add(60 * time.Second))
		assert.InDelta(t, 5.0, rate, 0.01)
	})
}

func TestRateReset(t *testing.T) {
	t.Run("reset clears state", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		r.Add(10.0)

		r.Reset()

		assert.Equal(t, 0.0, r.Rate())
		snapshot := r.Snapshot()
		assert.True(t, snapshot.LastUpdate.IsZero())
	})
}

func TestRateSnapshot(t *testing.T) {
	t.Run("snapshot captures state", func(t *testing.T) {
		r := NewRate(30 * time.Second)
		now := time.Now()

		r.AddAt(42.0, now)

		snapshot := r.Snapshot()
		assert.Equal(t, 42.0, snapshot.Rate)
		assert.Equal(t, now, snapshot.LastUpdate)
		assert.Equal(t, 30*time.Second, snapshot.HalfLife)
	})

	t.Run("snapshot is consistent", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		r.Set(100.0)

		snapshot := r.Snapshot()

		// Modify after snapshot
		r.Set(200.0)

		// Snapshot should be unchanged
		assert.Equal(t, 100.0, snapshot.Rate)
	})
}

func TestRateConcurrency(t *testing.T) {
	t.Run("concurrent adds", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		var wg sync.WaitGroup

		// Add from multiple goroutines
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					r.Add(1.0)
				}
			}()
		}

		wg.Wait()

		// Rate should be positive
		assert.Greater(t, r.Rate(), 0.0)
	})

	t.Run("concurrent reads and writes", func(_ *testing.T) {
		r := NewRate(60 * time.Second)
		var wg sync.WaitGroup

		// Writers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					r.Add(1.0)
				}
			}()
		}

		// Readers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					_ = r.Rate()
					_ = r.Snapshot()
				}
			}()
		}

		wg.Wait()
	})
}

func TestRateEdgeCases(t *testing.T) {
	t.Run("add zero", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		r.AddAt(10.0, now)
		r.AddAt(0.0, now)

		assert.Equal(t, 10.0, r.RateAt(now))
	})

	t.Run("add negative", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		r.AddAt(10.0, now)
		r.AddAt(-3.0, now)

		assert.Equal(t, 7.0, r.RateAt(now))
	})

	t.Run("multiple adds at same time", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		r.AddAt(1.0, now)
		r.AddAt(2.0, now)
		r.AddAt(3.0, now)
		r.AddAt(4.0, now)

		assert.Equal(t, 10.0, r.RateAt(now))
	})

	t.Run("add at earlier time uses last rate", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		r.AddAt(10.0, now)
		r.AddAt(5.0, now.Add(-10*time.Second))

		// Should just add without decay
		assert.Equal(t, 15.0, r.RateAt(now))
	})
}

func TestRateRealisticScenario(t *testing.T) {
	t.Run("simulate unique visitor rate tracking", func(t *testing.T) {
		r := NewRate(60 * time.Second)
		now := time.Now()

		// Simulate unique visitors over time
		// 0s: 10 unique visitors
		r.AddAt(10.0, now)

		// 30s: 5 unique visitors (10 * sqrt(0.5) + 5 ≈ 12.07)
		r.AddAt(5.0, now.Add(30*time.Second))
		rate30 := r.RateAt(now.Add(30 * time.Second))

		// 60s: 8 unique visitors (12.07 * 0.5 + 8 ≈ 14.04)
		r.AddAt(8.0, now.Add(60*time.Second))
		rate60 := r.RateAt(now.Add(60 * time.Second))

		// Rates should be reasonable
		assert.Greater(t, rate30, 5.0)    // More than just new visitors
		assert.Less(t, rate30, 15.0)      // Less than sum due to decay
		assert.Greater(t, rate60, 8.0)    // More than just new visitors
		assert.Greater(t, rate60, rate30) // Rate increasing due to more events
	})
}

// Benchmarks

func BenchmarkRateAdd(b *testing.B) {
	r := NewRate(60 * time.Second)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.Add(1.0)
	}
}

func BenchmarkRateRate(b *testing.B) {
	r := NewRate(60 * time.Second)
	r.Set(100.0)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = r.Rate()
	}
}

func BenchmarkRateSnapshot(b *testing.B) {
	r := NewRate(60 * time.Second)
	r.Set(100.0)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = r.Snapshot()
	}
}

func BenchmarkRateConcurrentAdd(b *testing.B) {
	r := NewRate(60 * time.Second)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.Add(1.0)
		}
	})
}

func BenchmarkRateConcurrentRate(b *testing.B) {
	r := NewRate(60 * time.Second)
	r.Set(100.0)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = r.Rate()
		}
	})
}
