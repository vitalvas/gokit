package fixedwindow

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		c := New(time.Second, 100, 0)
		defer c.Stop()

		assert.NotNil(t, c)
		assert.Equal(t, 0, c.Len())
	})

	t.Run("zero window uses default", func(t *testing.T) {
		c := New(0, 100, 0)
		defer c.Stop()

		assert.NotNil(t, c)
		assert.Equal(t, time.Minute, c.window)
	})

	t.Run("negative window uses default", func(t *testing.T) {
		c := New(-time.Second, 100, 0)
		defer c.Stop()

		assert.Equal(t, time.Minute, c.window)
	})

	t.Run("zero maxKeys uses default", func(t *testing.T) {
		c := New(time.Second, 0, 0)
		defer c.Stop()

		assert.Equal(t, 1000, c.maxKeys)
	})

	t.Run("negative maxKeys uses default", func(t *testing.T) {
		c := New(time.Second, -5, 0)
		defer c.Stop()

		assert.Equal(t, 1000, c.maxKeys)
	})

	t.Run("with cleanup interval", func(t *testing.T) {
		c := New(time.Second, 100, 50*time.Millisecond)
		defer c.Stop()

		assert.NotNil(t, c.cleanup)
	})

	t.Run("without cleanup interval", func(t *testing.T) {
		c := New(time.Second, 100, 0)
		defer c.Stop()

		assert.Nil(t, c.cleanup)
	})
}

func TestAllow(t *testing.T) {
	t.Run("within limit", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		assert.True(t, c.Allow("key1", 5))
		assert.True(t, c.Allow("key1", 5))
		assert.True(t, c.Allow("key1", 5))
	})

	t.Run("at exact limit", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		for range 3 {
			assert.True(t, c.Allow("key1", 3))
		}
		// Next call exceeds limit
		assert.False(t, c.Allow("key1", 3))
	})

	t.Run("exceeds limit locks out", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		// Use up limit
		for range 5 {
			c.Allow("key1", 5)
		}

		// Should be locked out
		assert.False(t, c.Allow("key1", 5))
		assert.False(t, c.Allow("key1", 5))
	})

	t.Run("lockout persists within window", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		for range 6 {
			c.Allow("key1", 5)
		}

		// Still locked out
		for range 10 {
			assert.False(t, c.Allow("key1", 5))
		}
	})

	t.Run("independent keys", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		// Exhaust key1
		for range 3 {
			c.Allow("key1", 2)
		}

		// key2 should still be allowed
		assert.True(t, c.Allow("key2", 2))
	})

	t.Run("window expiry resets lockout", func(t *testing.T) {
		c := New(50*time.Millisecond, 100, 0)
		defer c.Stop()

		// Lock out key
		for range 3 {
			c.Allow("key1", 1)
		}
		assert.False(t, c.Allow("key1", 1))

		// Wait for window to expire
		time.Sleep(80 * time.Millisecond)

		// Should be allowed again
		assert.True(t, c.Allow("key1", 1))
	})

	t.Run("limit of zero locks immediately", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		assert.False(t, c.Allow("key1", 0))
	})

	t.Run("consumes on allow", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		c.Allow("key1", 5)
		assert.Equal(t, 1, c.Count("key1"))

		c.Allow("key1", 5)
		assert.Equal(t, 2, c.Count("key1"))
	})
}

func TestAdd(t *testing.T) {
	t.Run("basic increment", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		count := c.Add("key1", 1)
		assert.Equal(t, 1, count)

		count = c.Add("key1", 1)
		assert.Equal(t, 2, count)
	})

	t.Run("increment by delta", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		count := c.Add("key1", 5)
		assert.Equal(t, 5, count)

		count = c.Add("key1", 3)
		assert.Equal(t, 8, count)
	})

	t.Run("zero delta returns current count", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		c.Add("key1", 5)
		count := c.Add("key1", 0)
		assert.Equal(t, 5, count)
	})

	t.Run("negative delta returns current count", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		c.Add("key1", 5)
		count := c.Add("key1", -1)
		assert.Equal(t, 5, count)
	})

	t.Run("increments even when locked out", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		// Lock out via Allow
		for range 3 {
			c.Allow("key1", 2)
		}
		assert.True(t, c.IsLockedOut("key1"))

		// Add should still increment
		count := c.Add("key1", 10)
		assert.Equal(t, 13, count) // 3 from Allow + 10 from Add
	})

	t.Run("window expiry resets counter", func(t *testing.T) {
		c := New(50*time.Millisecond, 100, 0)
		defer c.Stop()

		c.Add("key1", 10)
		time.Sleep(80 * time.Millisecond)

		count := c.Add("key1", 1)
		assert.Equal(t, 1, count)
	})
}

func TestCount(t *testing.T) {
	t.Run("non-existent key", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		assert.Equal(t, 0, c.Count("unknown"))
	})

	t.Run("tracked key", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		c.Add("key1", 5)
		assert.Equal(t, 5, c.Count("key1"))
	})

	t.Run("expired key returns zero", func(t *testing.T) {
		c := New(50*time.Millisecond, 100, 0)
		defer c.Stop()

		c.Add("key1", 5)
		time.Sleep(80 * time.Millisecond)

		assert.Equal(t, 0, c.Count("key1"))
	})

	t.Run("expired key is removed", func(t *testing.T) {
		c := New(50*time.Millisecond, 100, 0)
		defer c.Stop()

		c.Add("key1", 5)
		assert.Equal(t, 1, c.Len())

		time.Sleep(80 * time.Millisecond)
		c.Count("key1") // Triggers lazy cleanup

		assert.Equal(t, 0, c.Len())
	})
}

func TestIsLockedOut(t *testing.T) {
	t.Run("non-existent key", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		assert.False(t, c.IsLockedOut("unknown"))
	})

	t.Run("key within limit", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		c.Allow("key1", 5)
		assert.False(t, c.IsLockedOut("key1"))
	})

	t.Run("locked out key", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		for range 4 {
			c.Allow("key1", 3)
		}
		assert.True(t, c.IsLockedOut("key1"))
	})

	t.Run("lockout resets after window expiry", func(t *testing.T) {
		c := New(50*time.Millisecond, 100, 0)
		defer c.Stop()

		for range 3 {
			c.Allow("key1", 1)
		}
		assert.True(t, c.IsLockedOut("key1"))

		time.Sleep(80 * time.Millisecond)
		assert.False(t, c.IsLockedOut("key1"))
	})

	t.Run("add does not set lockout", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		c.Add("key1", 100)
		assert.False(t, c.IsLockedOut("key1"))
	})
}

func TestWindowExpiry(t *testing.T) {
	t.Run("non-existent key", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		assert.True(t, c.WindowExpiry("unknown").IsZero())
	})

	t.Run("active key returns future time", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		before := time.Now()
		c.Add("key1", 1)
		expiry := c.WindowExpiry("key1")

		assert.False(t, expiry.IsZero())
		assert.True(t, expiry.After(before))
		assert.True(t, expiry.Before(before.Add(2*time.Minute)))
	})

	t.Run("expired key returns zero time", func(t *testing.T) {
		c := New(50*time.Millisecond, 100, 0)
		defer c.Stop()

		c.Add("key1", 1)
		time.Sleep(80 * time.Millisecond)

		assert.True(t, c.WindowExpiry("key1").IsZero())
	})

	t.Run("expired key is removed", func(t *testing.T) {
		c := New(50*time.Millisecond, 100, 0)
		defer c.Stop()

		c.Add("key1", 1)
		assert.Equal(t, 1, c.Len())

		time.Sleep(80 * time.Millisecond)
		c.WindowExpiry("key1")

		assert.Equal(t, 0, c.Len())
	})

	t.Run("consistent with window duration", func(t *testing.T) {
		window := 200 * time.Millisecond
		c := New(window, 100, 0)
		defer c.Stop()

		before := time.Now()
		c.Add("key1", 1)
		expiry := c.WindowExpiry("key1")

		expected := before.Add(window)
		assert.InDelta(t, expected.UnixMilli(), expiry.UnixMilli(), 50)
	})
}

func TestReset(t *testing.T) {
	t.Run("clears all entries", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		c.Add("key1", 5)
		c.Add("key2", 10)
		c.Add("key3", 15)

		c.Reset()

		assert.Equal(t, 0, c.Len())
		assert.Equal(t, 0, c.Count("key1"))
		assert.Equal(t, 0, c.Count("key2"))
		assert.Equal(t, 0, c.Count("key3"))
	})

	t.Run("clears lockout state", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		for range 3 {
			c.Allow("key1", 1)
		}
		assert.True(t, c.IsLockedOut("key1"))

		c.Reset()
		assert.False(t, c.IsLockedOut("key1"))
	})
}

func TestLen(t *testing.T) {
	t.Run("empty counter", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		assert.Equal(t, 0, c.Len())
	})

	t.Run("tracks additions", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		c.Add("key1", 1)
		assert.Equal(t, 1, c.Len())

		c.Add("key2", 1)
		assert.Equal(t, 2, c.Len())

		// Same key doesn't increase len
		c.Add("key1", 1)
		assert.Equal(t, 2, c.Len())
	})
}

func TestEviction(t *testing.T) {
	t.Run("evicts oldest when at capacity", func(t *testing.T) {
		c := New(time.Minute, 3, 0)
		defer c.Stop()

		c.Add("key1", 1)
		time.Sleep(time.Millisecond)
		c.Add("key2", 1)
		time.Sleep(time.Millisecond)
		c.Add("key3", 1)

		assert.Equal(t, 3, c.Len())

		// Adding a 4th key should evict key1 (oldest window)
		c.Add("key4", 1)
		assert.Equal(t, 3, c.Len())
		assert.Equal(t, 0, c.Count("key1"))
		assert.Equal(t, 1, c.Count("key4"))
	})

	t.Run("evicts expired before active", func(t *testing.T) {
		c := New(50*time.Millisecond, 3, 0)
		defer c.Stop()

		c.Add("key1", 1)
		time.Sleep(80 * time.Millisecond)

		// key1 is now expired
		c.Add("key2", 1)
		c.Add("key3", 1)

		// Should evict expired key1 rather than active key2
		c.Add("key4", 1)
		assert.Equal(t, 3, c.Len())
		assert.Equal(t, 1, c.Count("key2"))
		assert.Equal(t, 1, c.Count("key3"))
		assert.Equal(t, 1, c.Count("key4"))
	})

	t.Run("single capacity", func(t *testing.T) {
		c := New(time.Minute, 1, 0)
		defer c.Stop()

		c.Add("key1", 5)
		c.Add("key2", 3)

		assert.Equal(t, 1, c.Len())
		assert.Equal(t, 3, c.Count("key2"))
		assert.Equal(t, 0, c.Count("key1"))
	})
}

func TestBackgroundCleanup(t *testing.T) {
	t.Run("removes expired entries", func(t *testing.T) {
		c := New(50*time.Millisecond, 100, 30*time.Millisecond)
		defer c.Stop()

		c.Add("key1", 1)
		c.Add("key2", 1)
		assert.Equal(t, 2, c.Len())

		// Wait for entries to expire and cleanup to run
		time.Sleep(120 * time.Millisecond)

		assert.Equal(t, 0, c.Len())
	})
}

func TestStop(t *testing.T) {
	t.Run("stop is idempotent", func(_ *testing.T) {
		c := New(time.Second, 100, 50*time.Millisecond)

		c.Stop()
		c.Stop() // Should not panic
	})

	t.Run("stop without cleanup goroutine", func(_ *testing.T) {
		c := New(time.Second, 100, 0)
		c.Stop() // Should not panic
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent allow", func(t *testing.T) {
		c := New(time.Minute, 1000, 0)
		defer c.Stop()

		var wg sync.WaitGroup

		for i := range 10 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				key := fmt.Sprintf("key-%d", id%5)
				for range 100 {
					c.Allow(key, 50)
				}
			}(i)
		}

		wg.Wait()

		// Each of 5 keys got 200 requests (10 goroutines / 5 keys * 100)
		total := 0
		for i := range 5 {
			total += c.Count(fmt.Sprintf("key-%d", i))
		}
		assert.Equal(t, 1000, total)
	})

	t.Run("concurrent reads and writes", func(_ *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		done := make(chan struct{})

		// Writer
		go func() {
			for i := range 1000 {
				c.Allow(fmt.Sprintf("key-%d", i%10), 100)
			}
			close(done)
		}()

		// Reader
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					c.Count("key-0")
					c.IsLockedOut("key-0")
					c.Len()
				}
			}
		}()

		<-done
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty string key", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		assert.True(t, c.Allow("", 5))
		assert.Equal(t, 1, c.Count(""))
	})

	t.Run("very long key", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		longKey := string(make([]byte, 10000))
		assert.True(t, c.Allow(longKey, 5))
		assert.Equal(t, 1, c.Count(longKey))
	})

	t.Run("allow then add combination", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		c.Allow("key1", 10) // count=1
		c.Add("key1", 4)    // count=5
		c.Allow("key1", 10) // count=6

		assert.Equal(t, 6, c.Count("key1"))
		assert.False(t, c.IsLockedOut("key1"))
	})

	t.Run("add does not trigger lockout", func(t *testing.T) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		c.Add("key1", 1000)
		assert.False(t, c.IsLockedOut("key1"))

		// But Allow would see the high count and lock out
		assert.False(t, c.Allow("key1", 5))
		assert.True(t, c.IsLockedOut("key1"))
	})
}

func BenchmarkCounter_Allow(b *testing.B) {
	c := New(time.Minute, 10000, 0)
	defer c.Stop()

	b.ReportAllocs()
	for b.Loop() {
		c.Allow("bench-key", 1000000)
	}
}

func BenchmarkCounter_Add(b *testing.B) {
	c := New(time.Minute, 10000, 0)
	defer c.Stop()

	b.ReportAllocs()
	for b.Loop() {
		c.Add("bench-key", 1)
	}
}

func BenchmarkCounter_Count(b *testing.B) {
	c := New(time.Minute, 10000, 0)
	defer c.Stop()

	c.Add("bench-key", 100)

	b.ReportAllocs()
	for b.Loop() {
		c.Count("bench-key")
	}
}

func BenchmarkCounter_IsLockedOut(b *testing.B) {
	c := New(time.Minute, 10000, 0)
	defer c.Stop()

	for range 3 {
		c.Allow("bench-key", 2)
	}

	b.ReportAllocs()
	for b.Loop() {
		c.IsLockedOut("bench-key")
	}
}

func BenchmarkCounter_ConcurrentAllow(b *testing.B) {
	c := New(time.Minute, 10000, 0)
	defer c.Stop()

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Allow(fmt.Sprintf("key-%d", i%100), 1000000)
			i++
		}
	})
}

func FuzzCounter_Allow(f *testing.F) {
	f.Add("test", 10)
	f.Add("", 1)
	f.Add("key-1", 0)
	f.Add("a", 100)

	f.Fuzz(func(t *testing.T, key string, limit int) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		result := c.Allow(key, limit)
		count := c.Count(key)

		if count < 1 {
			t.Error("count should be at least 1 after Allow")
		}

		if limit >= 1 && count <= limit && !result {
			t.Error("Allow should return true when count is within limit")
		}
	})
}

func FuzzCounter_Add(f *testing.F) {
	f.Add("test", 1)
	f.Add("", 5)
	f.Add("key-1", 100)

	f.Fuzz(func(t *testing.T, key string, delta int) {
		c := New(time.Minute, 100, 0)
		defer c.Stop()

		count := c.Add(key, delta)

		if delta > 0 && count != delta {
			t.Errorf("expected count %d, got %d", delta, count)
		}

		if delta <= 0 && count != 0 {
			t.Errorf("expected count 0 for non-positive delta, got %d", count)
		}
	})
}
