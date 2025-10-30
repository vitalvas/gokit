package spacesaving

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRateTracker(t *testing.T) {
	t.Run("create with default parameters", func(t *testing.T) {
		rt := NewRateTracker(100, 60*time.Second)
		assert.NotNil(t, rt)
		assert.Equal(t, 100, rt.Capacity())
		assert.Equal(t, 60*time.Second, rt.HalfLife())
		assert.Equal(t, 0, rt.Size())
	})

	t.Run("create with zero capacity uses default", func(t *testing.T) {
		rt := NewRateTracker(0, 60*time.Second)
		assert.Equal(t, 100, rt.Capacity())
	})

	t.Run("create with zero half-life uses default", func(t *testing.T) {
		rt := NewRateTracker(100, 0)
		assert.Equal(t, 60*time.Second, rt.HalfLife())
	})
}

func TestRateTrackerTouch(t *testing.T) {
	t.Run("touch single item", func(t *testing.T) {
		rt := NewRateTracker(10, 60*time.Second)
		rate := rt.Touch("apple")
		assert.Equal(t, 1.0, rate)
		assert.Equal(t, 1, rt.Size())
	})

	t.Run("touch duplicate items increases rate", func(t *testing.T) {
		rt := NewRateTracker(10, 60*time.Second)
		now := time.Now()

		rt.TouchAt("apple", now)
		rate := rt.TouchAt("apple", now.Add(1*time.Second))

		// Rate should be decayed original + 1
		assert.Greater(t, rate, 1.0)
		assert.Less(t, rate, 2.0) // Less than 2 due to decay
	})

	t.Run("touch beyond capacity evicts minimum", func(t *testing.T) {
		rt := NewRateTracker(3, 60*time.Second)
		now := time.Now()

		rt.TouchAt("a", now)
		rt.TouchAt("b", now)
		rt.TouchAt("c", now)

		assert.Equal(t, 3, rt.Size())

		// Add new item should evict one
		rt.TouchAt("d", now)
		assert.Equal(t, 3, rt.Size())
	})
}

func TestRateTrackerDecay(t *testing.T) {
	t.Run("rate decays over time", func(t *testing.T) {
		rt := NewRateTracker(10, 60*time.Second)
		now := time.Now()

		// Touch item
		rt.TouchAt("apple", now)

		// Check rate immediately
		rate1, _ := rt.RateAt("apple", now)
		assert.Equal(t, 1.0, rate1)

		// Check rate after 60 seconds (one half-life)
		rate2, _ := rt.RateAt("apple", now.Add(60*time.Second))
		assert.InDelta(t, 0.5, rate2, 0.01) // Should be ~0.5
	})

	t.Run("rate decays exponentially", func(t *testing.T) {
		rt := NewRateTracker(10, 10*time.Second)
		now := time.Now()

		rt.TouchAt("apple", now)

		// After 1 half-life: ~0.5
		rate1, _ := rt.RateAt("apple", now.Add(10*time.Second))
		assert.InDelta(t, 0.5, rate1, 0.01)

		// After 2 half-lives: ~0.25
		rate2, _ := rt.RateAt("apple", now.Add(20*time.Second))
		assert.InDelta(t, 0.25, rate2, 0.01)

		// After 3 half-lives: ~0.125
		rate3, _ := rt.RateAt("apple", now.Add(30*time.Second))
		assert.InDelta(t, 0.125, rate3, 0.01)
	})
}

func TestRateTrackerRate(t *testing.T) {
	t.Run("rate for tracked item", func(t *testing.T) {
		rt := NewRateTracker(10, 60*time.Second)
		now := time.Now()

		rt.TouchAt("apple", now)
		rt.TouchAt("apple", now.Add(1*time.Second))
		rt.TouchAt("apple", now.Add(2*time.Second))

		rate, errorRate := rt.RateAt("apple", now.Add(2*time.Second))
		assert.Greater(t, rate, 0.0)
		assert.GreaterOrEqual(t, errorRate, 0.0)
	})

	t.Run("rate for non-tracked item", func(t *testing.T) {
		rt := NewRateTracker(10, 60*time.Second)
		rt.Touch("apple")

		rate, errorRate := rt.Rate("banana")
		assert.Equal(t, 0.0, rate)
		assert.Greater(t, errorRate, 0.0) // Should have error bound
	})
}

func TestRateTrackerTop(t *testing.T) {
	t.Run("top items by rate", func(t *testing.T) {
		rt := NewRateTracker(10, 60*time.Second)
		now := time.Now()

		// Add items with different frequencies
		for i := 0; i < 5; i++ {
			rt.TouchAt("a", now.Add(time.Duration(i)*time.Second))
		}
		for i := 0; i < 3; i++ {
			rt.TouchAt("b", now.Add(time.Duration(i)*time.Second))
		}
		rt.TouchAt("c", now)

		top := rt.TopAt(3, now.Add(5*time.Second))
		assert.Len(t, top, 3)
		assert.Equal(t, "a", top[0].Value)
		assert.Greater(t, top[0].Rate, top[1].Rate)
	})

	t.Run("top with n=0", func(t *testing.T) {
		rt := NewRateTracker(10, 60*time.Second)
		rt.Touch("a")

		top := rt.Top(0)
		assert.Nil(t, top)
	})

	t.Run("top from empty tracker", func(t *testing.T) {
		rt := NewRateTracker(10, 60*time.Second)
		top := rt.Top(5)
		assert.Len(t, top, 0)
	})
}

func TestRateTrackerAll(t *testing.T) {
	t.Run("all returns all items", func(t *testing.T) {
		rt := NewRateTracker(10, 60*time.Second)
		rt.Touch("a")
		rt.Touch("b")
		rt.Touch("c")

		all := rt.All()
		assert.Len(t, all, 3)
	})

	t.Run("all sorted by rate", func(t *testing.T) {
		rt := NewRateTracker(10, 60*time.Second)
		now := time.Now()

		for i := 0; i < 5; i++ {
			rt.TouchAt("a", now.Add(time.Duration(i)*time.Second))
		}
		for i := 0; i < 2; i++ {
			rt.TouchAt("b", now.Add(time.Duration(i)*time.Second))
		}

		all := rt.AllAt(now.Add(5 * time.Second))
		assert.Equal(t, "a", all[0].Value)
		assert.Greater(t, all[0].Rate, all[1].Rate)
	})
}

func TestRateTrackerReset(t *testing.T) {
	t.Run("reset clears all data", func(t *testing.T) {
		rt := NewRateTracker(10, 60*time.Second)
		rt.Touch("a")
		rt.Touch("b")

		rt.Reset()

		assert.Equal(t, 0, rt.Size())
		rate, _ := rt.Rate("a")
		assert.Equal(t, 0.0, rate)
	})
}

func TestRateTrackerExportImport(t *testing.T) {
	t.Run("export and import", func(t *testing.T) {
		rt := NewRateTracker(10, 60*time.Second)
		now := time.Now()

		for i := 0; i < 5; i++ {
			rt.TouchAt("apple", now.Add(time.Duration(i)*time.Second))
		}
		for i := 0; i < 3; i++ {
			rt.TouchAt("banana", now.Add(time.Duration(i)*time.Second))
		}

		data, err := rt.Export()
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		imported, err := ImportRateTracker(data)
		assert.NoError(t, err)
		assert.NotNil(t, imported)

		assert.Equal(t, rt.Size(), imported.Size())
		assert.Equal(t, rt.Capacity(), imported.Capacity())
		assert.Equal(t, rt.HalfLife(), imported.HalfLife())
	})

	t.Run("export empty tracker", func(t *testing.T) {
		rt := NewRateTracker(10, 60*time.Second)
		data, err := rt.Export()
		assert.NoError(t, err)

		imported, err := ImportRateTracker(data)
		assert.NoError(t, err)
		assert.Equal(t, 0, imported.Size())
	})

	t.Run("import invalid data", func(t *testing.T) {
		_, err := ImportRateTracker([]byte("invalid"))
		assert.Error(t, err)
	})
}

func TestRateTrackerRecentVsOld(t *testing.T) {
	t.Run("recent events have more weight", func(t *testing.T) {
		rt := NewRateTracker(10, 10*time.Second)
		now := time.Now()

		// Item A: old events
		for i := 0; i < 10; i++ {
			rt.TouchAt("old", now.Add(time.Duration(i)*time.Second))
		}

		// Wait and add recent events for item B
		laterTime := now.Add(50 * time.Second)
		for i := 0; i < 5; i++ {
			rt.TouchAt("recent", laterTime.Add(time.Duration(i)*time.Second))
		}

		// Check rates at the later time
		top := rt.TopAt(2, laterTime.Add(5*time.Second))

		// "recent" should have higher rate due to recency
		assert.Equal(t, "recent", top[0].Value)
	})
}

func TestRateTrackerConcurrency(t *testing.T) {
	t.Run("concurrent touches", func(t *testing.T) {
		rt := NewRateTracker(100, 60*time.Second)
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				item := fmt.Sprintf("item-%d", id%5)
				for j := 0; j < 100; j++ {
					rt.Touch(item)
				}
			}(i)
		}

		wg.Wait()

		// Verify all items were tracked
		top := rt.Top(5)
		assert.Len(t, top, 5)
	})

	t.Run("concurrent reads and writes", func(_ *testing.T) {
		rt := NewRateTracker(50, 60*time.Second)
		done := make(chan bool)

		// Writer
		go func() {
			for i := 0; i < 1000; i++ {
				rt.Touch(fmt.Sprintf("item-%d", i%10))
			}
			done <- true
		}()

		// Readers
		go func() {
			for i := 0; i < 100; i++ {
				_ = rt.Top(5)
				_, _ = rt.Rate("item-0")
			}
			done <- true
		}()

		<-done
		<-done
	})
}

func TestRateTrackerEdgeCases(t *testing.T) {
	t.Run("zero rate after long time", func(t *testing.T) {
		rt := NewRateTracker(10, 1*time.Second)
		now := time.Now()

		rt.TouchAt("apple", now)

		// After many half-lives, rate should be near zero
		rate, _ := rt.RateAt("apple", now.Add(100*time.Second))
		assert.Less(t, rate, 0.001)
	})

	t.Run("single capacity", func(t *testing.T) {
		rt := NewRateTracker(1, 60*time.Second)
		rt.Touch("a")
		rt.Touch("b")

		assert.Equal(t, 1, rt.Size())
	})
}

// Benchmarks for RateTracker

func BenchmarkRateTrackerTouch(b *testing.B) {
	rt := NewRateTracker(1000, 60*time.Second)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rt.Touch(fmt.Sprintf("item-%d", i%100))
	}
}

func BenchmarkRateTrackerRate(b *testing.B) {
	rt := NewRateTracker(1000, 60*time.Second)
	for i := 0; i < 10000; i++ {
		rt.Touch(fmt.Sprintf("item-%d", i%100))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = rt.Rate("item-50")
	}
}

func BenchmarkRateTrackerTop(b *testing.B) {
	rt := NewRateTracker(1000, 60*time.Second)
	for i := 0; i < 10000; i++ {
		rt.Touch(fmt.Sprintf("item-%d", i%100))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = rt.Top(10)
	}
}

func BenchmarkRateTrackerConcurrentTouch(b *testing.B) {
	rt := NewRateTracker(1000, 60*time.Second)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			rt.Touch(fmt.Sprintf("item-%d", i%100))
			i++
		}
	})
}

func BenchmarkRateTrackerExport(b *testing.B) {
	rt := NewRateTracker(1000, 60*time.Second)
	for i := 0; i < 10000; i++ {
		rt.Touch(fmt.Sprintf("item-%d", i%100))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = rt.Export()
	}
}

func BenchmarkRateTrackerImport(b *testing.B) {
	rt := NewRateTracker(1000, 60*time.Second)
	for i := 0; i < 10000; i++ {
		rt.Touch(fmt.Sprintf("item-%d", i%100))
	}
	data, _ := rt.Export()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ImportRateTracker(data)
	}
}
