package ewma

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("create with custom alpha", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		assert.NotNil(t, e)
		assert.Equal(t, 0.5, e.alpha)
	})

	t.Run("create with custom interval", func(t *testing.T) {
		e := New(0.5, 10*time.Second)
		assert.Equal(t, 10*time.Second, e.interval)
	})

	t.Run("default interval for zero", func(t *testing.T) {
		e := New(0.5, 0)
		assert.Equal(t, Interval, e.interval)
	})
}

func TestStandardEWMAs(t *testing.T) {
	t.Run("1-minute EWMA", func(t *testing.T) {
		e := New1MinuteEWMA()
		assert.NotNil(t, e)
		assert.Greater(t, e.alpha, 0.0)
		assert.Less(t, e.alpha, 1.0)
	})

	t.Run("5-minute EWMA", func(t *testing.T) {
		e := New5MinuteEWMA()
		assert.NotNil(t, e)
		assert.Greater(t, e.alpha, 0.0)
		assert.Less(t, e.alpha, 1.0)
	})

	t.Run("15-minute EWMA", func(t *testing.T) {
		e := New15MinuteEWMA()
		assert.NotNil(t, e)
		assert.Greater(t, e.alpha, 0.0)
		assert.Less(t, e.alpha, 1.0)
	})

	t.Run("alpha values are ordered", func(t *testing.T) {
		e1 := New1MinuteEWMA()
		e5 := New5MinuteEWMA()
		e15 := New15MinuteEWMA()

		// Shorter time windows have higher alpha (react faster)
		assert.Greater(t, e1.alpha, e5.alpha)
		assert.Greater(t, e5.alpha, e15.alpha)
	})
}

func TestAdd(t *testing.T) {
	t.Run("add single event", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		e.Add(1)

		snapshot := e.Snapshot()
		assert.Equal(t, uint64(1), snapshot.Uncounted)
	})

	t.Run("add multiple events", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		e.Add(10)

		snapshot := e.Snapshot()
		assert.Equal(t, uint64(10), snapshot.Uncounted)
	})

	t.Run("add accumulates", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		e.Add(5)
		e.Add(3)

		snapshot := e.Snapshot()
		assert.Equal(t, uint64(8), snapshot.Uncounted)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("update is equivalent to Add(1)", func(t *testing.T) {
		e1 := NewWithAlpha(0.5)
		e2 := NewWithAlpha(0.5)

		e1.Update()
		e2.Add(1)

		assert.Equal(t, e2.Snapshot().Uncounted, e1.Snapshot().Uncounted)
	})
}

func TestUpdateWithValue(t *testing.T) {
	t.Run("update with initial value", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		e.UpdateWithValue(100)

		assert.Equal(t, 100.0, e.Rate())
	})

	t.Run("update smooths values", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		e.UpdateWithValue(100)
		e.UpdateWithValue(200)

		// With alpha=0.5: 0.5*200 + 0.5*100 = 150
		assert.InDelta(t, 150.0, e.Rate(), 0.1)
	})

	t.Run("multiple updates converge", func(t *testing.T) {
		e := NewWithAlpha(0.9)

		// Feed constant value
		for i := 0; i < 10; i++ {
			e.UpdateWithValue(100)
		}

		// Should converge close to 100
		assert.InDelta(t, 100.0, e.Rate(), 1.0)
	})
}

func TestTick(t *testing.T) {
	t.Run("tick initializes rate", func(t *testing.T) {
		e := New(0.5, 5*time.Second)
		e.Add(100)
		e.Tick()

		// Rate = 100 events / 5 seconds = 20 events/sec
		assert.InDelta(t, 20.0, e.Rate(), 0.1)
	})

	t.Run("tick clears uncounted", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		e.Add(10)
		e.Tick()

		snapshot := e.Snapshot()
		assert.Equal(t, uint64(0), snapshot.Uncounted)
	})

	t.Run("tick updates rate with smoothing", func(t *testing.T) {
		e := New(0.5, 5*time.Second)

		// First tick
		e.Add(100)
		e.Tick()
		rate1 := e.Rate()

		// Second tick with different count
		e.Add(200)
		e.Tick()
		rate2 := e.Rate()

		// Rate should change but be smoothed
		assert.NotEqual(t, rate1, rate2)
		assert.Greater(t, rate2, rate1)
	})

	t.Run("multiple ticks converge to stable rate", func(t *testing.T) {
		e := New(0.9, 1*time.Second)

		// Add same count each tick
		for i := 0; i < 10; i++ {
			e.Add(100)
			e.Tick()
		}

		// Should converge to 100 events/sec
		assert.InDelta(t, 100.0, e.Rate(), 5.0)
	})
}

func TestRate(t *testing.T) {
	t.Run("initial rate is zero", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		assert.Equal(t, 0.0, e.Rate())
	})

	t.Run("rate after tick", func(t *testing.T) {
		e := New(0.5, 1*time.Second)
		e.Add(50)
		e.Tick()

		assert.Equal(t, 50.0, e.Rate())
	})
}

func TestSnapshot(t *testing.T) {
	t.Run("snapshot captures state", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		e.Add(10)
		e.Set(42.0)

		snapshot := e.Snapshot()
		assert.Equal(t, 42.0, snapshot.Rate)
		assert.Equal(t, uint64(10), snapshot.Uncounted)
		assert.True(t, snapshot.Initialized)
	})

	t.Run("snapshot is consistent", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		e.Set(100.0)

		snapshot := e.Snapshot()

		// Modify after snapshot
		e.Set(200.0)

		// Snapshot should be unchanged
		assert.Equal(t, 100.0, snapshot.Rate)
	})
}

func TestReset(t *testing.T) {
	t.Run("reset clears state", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		e.Add(100)
		e.Set(42.0)

		e.Reset()

		snapshot := e.Snapshot()
		assert.Equal(t, 0.0, snapshot.Rate)
		assert.Equal(t, uint64(0), snapshot.Uncounted)
		assert.False(t, snapshot.Initialized)
	})
}

func TestSet(t *testing.T) {
	t.Run("set initializes rate", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		e.Set(123.45)

		assert.Equal(t, 123.45, e.Rate())
		assert.True(t, e.Snapshot().Initialized)
	})

	t.Run("set overwrites existing rate", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		e.Set(100)
		e.Set(200)

		assert.Equal(t, 200.0, e.Rate())
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent adds", func(t *testing.T) {
		e := NewWithAlpha(0.5)
		var wg sync.WaitGroup

		// Add from multiple goroutines
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 1000; j++ {
					e.Add(1)
				}
			}()
		}

		wg.Wait()
		snapshot := e.Snapshot()
		assert.Equal(t, uint64(10000), snapshot.Uncounted)
	})

	t.Run("concurrent reads and writes", func(_ *testing.T) {
		e := NewWithAlpha(0.5)
		var wg sync.WaitGroup

		// Writers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					e.Add(1)
					e.Tick()
				}
			}()
		}

		// Readers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					_ = e.Rate()
					_ = e.Snapshot()
				}
			}()
		}

		wg.Wait()
	})
}

func TestMovingAverage(t *testing.T) {
	t.Run("create moving average", func(t *testing.T) {
		ma := NewMovingAverage()
		assert.NotNil(t, ma)
		assert.NotNil(t, ma.m1)
		assert.NotNil(t, ma.m5)
		assert.NotNil(t, ma.m15)
	})

	t.Run("add updates all windows", func(t *testing.T) {
		ma := NewMovingAverage()
		ma.Add(100)

		s1, s5, s15 := ma.Snapshot()
		assert.Equal(t, uint64(100), s1.Uncounted)
		assert.Equal(t, uint64(100), s5.Uncounted)
		assert.Equal(t, uint64(100), s15.Uncounted)
	})

	t.Run("update is equivalent to Add(1)", func(t *testing.T) {
		ma := NewMovingAverage()
		ma.Update()

		s1, _, _ := ma.Snapshot()
		assert.Equal(t, uint64(1), s1.Uncounted)
	})

	t.Run("tick updates all windows", func(t *testing.T) {
		ma := NewMovingAverage()
		ma.Add(100)
		ma.Tick()

		// All rates should be non-zero
		assert.Greater(t, ma.Rate1(), 0.0)
		assert.Greater(t, ma.Rate5(), 0.0)
		assert.Greater(t, ma.Rate15(), 0.0)
	})

	t.Run("rates differ by time window", func(t *testing.T) {
		ma := NewMovingAverage()

		// Add constant rate
		for i := 0; i < 10; i++ {
			ma.Add(100)
			ma.Tick()
		}

		m1, m5, m15 := ma.Rates()

		// All should be positive
		assert.Greater(t, m1, 0.0)
		assert.Greater(t, m5, 0.0)
		assert.Greater(t, m15, 0.0)

		// 1-minute reacts fastest
		assert.GreaterOrEqual(t, m1, m5)
		assert.GreaterOrEqual(t, m5, m15)
	})

	t.Run("reset clears all windows", func(t *testing.T) {
		ma := NewMovingAverage()
		ma.Add(100)
		ma.Tick()

		ma.Reset()

		assert.Equal(t, 0.0, ma.Rate1())
		assert.Equal(t, 0.0, ma.Rate5())
		assert.Equal(t, 0.0, ma.Rate15())
	})
}

func TestDecayBehavior(t *testing.T) {
	t.Run("high alpha reacts quickly", func(t *testing.T) {
		eHigh := NewWithAlpha(0.9)
		eLow := NewWithAlpha(0.1)

		// Both start at same rate
		eHigh.UpdateWithValue(100)
		eLow.UpdateWithValue(100)

		// Update with new value
		eHigh.UpdateWithValue(200)
		eLow.UpdateWithValue(200)

		// High alpha should be closer to 200
		assert.Greater(t, eHigh.Rate(), eLow.Rate())
	})

	t.Run("low alpha provides smoothing", func(t *testing.T) {
		e := NewWithAlpha(0.1)

		// Add spike
		e.UpdateWithValue(100)
		e.UpdateWithValue(1000)

		// Low alpha should smooth the spike
		rate := e.Rate()
		assert.Less(t, rate, 1000.0)
		assert.Greater(t, rate, 100.0)
	})
}

// Benchmarks

func BenchmarkAdd(b *testing.B) {
	e := NewWithAlpha(0.5)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e.Add(1)
	}
}

func BenchmarkUpdate(b *testing.B) {
	e := NewWithAlpha(0.5)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e.Update()
	}
}

func BenchmarkTick(b *testing.B) {
	e := NewWithAlpha(0.5)
	e.Add(100)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e.Tick()
	}
}

func BenchmarkRate(b *testing.B) {
	e := NewWithAlpha(0.5)
	e.Set(100)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = e.Rate()
	}
}

func BenchmarkSnapshot(b *testing.B) {
	e := NewWithAlpha(0.5)
	e.Set(100)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = e.Snapshot()
	}
}

func BenchmarkMovingAverageAdd(b *testing.B) {
	ma := NewMovingAverage()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ma.Add(1)
	}
}

func BenchmarkMovingAverageTick(b *testing.B) {
	ma := NewMovingAverage()
	ma.Add(100)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ma.Tick()
	}
}

func BenchmarkConcurrentAdd(b *testing.B) {
	e := NewWithAlpha(0.5)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			e.Add(1)
		}
	})
}

func BenchmarkConcurrentRate(b *testing.B) {
	e := NewWithAlpha(0.5)
	e.Set(100)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = e.Rate()
		}
	})
}

// Helper to check for data races
func TestNoDataRaces(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race detection test in short mode")
	}

	e := NewWithAlpha(0.5)
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			e.Add(1)
			e.Tick()
			e.UpdateWithValue(100)
			e.Set(float64(i))
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			_ = e.Rate()
			_ = e.Snapshot()
		}
		done <- true
	}()

	// Wait for both
	<-done
	<-done
}
