package gcra

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("computes interval and tau", func(t *testing.T) {
		l := New(10, time.Second, 5, 100, 0)
		defer l.Stop()

		assert.Equal(t, int64(100*time.Millisecond), l.interval)
		assert.Equal(t, int64(400*time.Millisecond), l.tau)
	})

	t.Run("defaults on invalid arguments", func(t *testing.T) {
		l := New(0, 0, 0, 0, 0)
		defer l.Stop()

		assert.Equal(t, int64(time.Second), l.interval)
		assert.Equal(t, int64(0), l.tau)
		assert.Equal(t, 1000, l.maxKeys)
	})

	t.Run("sub-nanosecond interval clamped", func(t *testing.T) {
		l := New(1000, 100*time.Nanosecond, 1, 100, 0)
		defer l.Stop()

		assert.Equal(t, int64(1), l.interval)
	})

	t.Run("extreme config clamped without overflow", func(t *testing.T) {
		l := New(1, math.MaxInt64, math.MaxInt, 100, 0)
		defer l.Stop()

		assert.Equal(t, maxInterval, l.interval)
		assert.Equal(t, maxTau, l.tau)
	})

	t.Run("with and without cleanup ticker", func(t *testing.T) {
		withCleanup := New(1, time.Second, 1, 100, time.Minute)
		defer withCleanup.Stop()
		assert.NotNil(t, withCleanup.cleanup)

		noCleanup := New(1, time.Second, 1, 100, 0)
		defer noCleanup.Stop()
		assert.Nil(t, noCleanup.cleanup)
	})
}

func TestAllow(t *testing.T) {
	t.Run("burst then deny", func(t *testing.T) {
		l := New(10, time.Second, 3, 100, 0)
		defer l.Stop()

		for i := range 3 {
			assert.True(t, l.allowN("key1", 1, 0), "burst request %d", i)
		}
		assert.False(t, l.allowN("key1", 1, 0))
	})

	t.Run("refills at emission interval", func(t *testing.T) {
		l := New(10, time.Second, 1, 100, 0)
		defer l.Stop()

		assert.True(t, l.allowN("key1", 1, 0))
		assert.False(t, l.allowN("key1", 1, 0))
		assert.False(t, l.allowN("key1", 1, int64(99*time.Millisecond)))
		assert.True(t, l.allowN("key1", 1, int64(100*time.Millisecond)))
	})

	t.Run("full burst restored after drain", func(t *testing.T) {
		l := New(10, time.Second, 3, 100, 0)
		defer l.Stop()

		for range 3 {
			assert.True(t, l.allowN("key1", 1, 0))
		}

		later := int64(time.Second)
		for i := range 3 {
			assert.True(t, l.allowN("key1", 1, later), "restored burst request %d", i)
		}
		assert.False(t, l.allowN("key1", 1, later))
	})

	t.Run("keys are independent", func(t *testing.T) {
		l := New(10, time.Second, 1, 100, 0)
		defer l.Stop()

		assert.True(t, l.allowN("key1", 1, 0))
		assert.True(t, l.allowN("key2", 1, 0))
		assert.False(t, l.allowN("key1", 1, 0))
	})

	t.Run("public API", func(t *testing.T) {
		l := New(10, time.Second, 2, 100, 0)
		defer l.Stop()

		assert.True(t, l.Allow("key1"))
		assert.True(t, l.Allow("key1"))
		assert.False(t, l.Allow("key1"))
	})
}

func TestAllowN(t *testing.T) {
	t.Run("batch within burst", func(t *testing.T) {
		l := New(10, time.Second, 5, 100, 0)
		defer l.Stop()

		assert.True(t, l.allowN("key1", 5, 0))
		assert.False(t, l.allowN("key1", 1, 0))
	})

	t.Run("batch exceeding burst consumes nothing", func(t *testing.T) {
		l := New(10, time.Second, 5, 100, 0)
		defer l.Stop()

		assert.False(t, l.allowN("key1", 6, 0))
		assert.Equal(t, 0, l.Len())
		assert.True(t, l.allowN("key1", 5, 0))
	})

	t.Run("denied batch on existing key keeps state", func(t *testing.T) {
		l := New(10, time.Second, 5, 100, 0)
		defer l.Stop()

		assert.True(t, l.allowN("key1", 3, 0))
		assert.False(t, l.allowN("key1", 3, 0))
		assert.True(t, l.allowN("key1", 2, 0))
	})

	t.Run("non-positive n always allowed", func(t *testing.T) {
		l := New(10, time.Second, 1, 100, 0)
		defer l.Stop()

		assert.True(t, l.AllowN("key1", 0))
		assert.True(t, l.AllowN("key1", -1))
		assert.Equal(t, 0, l.Len())
	})

	t.Run("public API", func(t *testing.T) {
		l := New(10, time.Second, 3, 100, 0)
		defer l.Stop()

		assert.True(t, l.AllowN("key1", 3))
		assert.False(t, l.AllowN("key1", 1))
	})
}

func TestRetryAfter(t *testing.T) {
	t.Run("untracked key", func(t *testing.T) {
		l := New(10, time.Second, 1, 100, 0)
		defer l.Stop()

		assert.Equal(t, time.Duration(0), l.retryAfter("key1", 0))
	})

	t.Run("after exhausting burst", func(t *testing.T) {
		l := New(10, time.Second, 1, 100, 0)
		defer l.Stop()

		assert.True(t, l.allowN("key1", 1, 0))
		assert.Equal(t, 100*time.Millisecond, l.retryAfter("key1", 0))
		assert.Equal(t, time.Duration(0), l.retryAfter("key1", int64(100*time.Millisecond)))
	})

	t.Run("zero while capacity remains", func(t *testing.T) {
		l := New(10, time.Second, 2, 100, 0)
		defer l.Stop()

		assert.True(t, l.allowN("key1", 1, 0))
		assert.Equal(t, time.Duration(0), l.retryAfter("key1", 0))
	})

	t.Run("public API", func(t *testing.T) {
		l := New(10, time.Second, 1, 100, 0)
		defer l.Stop()

		assert.True(t, l.Allow("key1"))
		assert.Greater(t, l.RetryAfter("key1"), time.Duration(0))
	})
}

func TestEviction(t *testing.T) {
	t.Run("drained entries removed first", func(t *testing.T) {
		l := New(10, time.Second, 1, 2, 0)
		defer l.Stop()

		assert.True(t, l.allowN("key1", 1, 0))
		assert.True(t, l.allowN("key2", 1, 0))

		later := int64(time.Second)
		assert.True(t, l.allowN("key3", 1, later))
		assert.Equal(t, 1, l.Len())
	})

	t.Run("active state retained at capacity", func(t *testing.T) {
		l := New(10, time.Second, 5, 2, 0)
		defer l.Stop()

		assert.True(t, l.allowN("key1", 1, 0))
		assert.True(t, l.allowN("key2", 5, 0))
		assert.False(t, l.allowN("key3", 1, 0))

		assert.Equal(t, 2, l.Len())
		_, key1Tracked := l.entries["key1"]
		assert.True(t, key1Tracked)
		_, key2Tracked := l.entries["key2"]
		assert.True(t, key2Tracked)
	})
}

func TestCleanup(t *testing.T) {
	t.Run("background cleanup removes drained entries", func(t *testing.T) {
		l := New(1000, time.Second, 1, 100, 10*time.Millisecond)
		defer l.Stop()

		assert.True(t, l.Allow("key1"))
		assert.Equal(t, 1, l.Len())

		assert.Eventually(t, func() bool {
			return l.Len() == 0
		}, time.Second, 5*time.Millisecond)
	})

	t.Run("removeDrained keeps active entries", func(t *testing.T) {
		l := New(1, time.Hour, 1, 100, 0)
		defer l.Stop()

		assert.True(t, l.allowN("key1", 1, 0))
		l.removeDrained(0)
		assert.Equal(t, 1, l.Len())
	})
}

func TestResetAndStop(t *testing.T) {
	t.Run("reset clears all keys", func(t *testing.T) {
		l := New(10, time.Second, 1, 100, 0)
		defer l.Stop()

		assert.True(t, l.Allow("key1"))
		assert.True(t, l.Allow("key2"))
		assert.Equal(t, 2, l.Len())

		l.Reset()
		assert.Equal(t, 0, l.Len())
		assert.True(t, l.Allow("key1"))
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		l := New(10, time.Second, 1, 100, time.Minute)
		assert.NotPanics(t, func() {
			l.Stop()
			l.Stop()
		})
	})
}

func TestConcurrency(t *testing.T) {
	l := New(1000, time.Second, 100, 1000, time.Millisecond)
	defer l.Stop()

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", n%3)
			for range 100 {
				l.Allow(key)
				l.RetryAfter(key)
				l.Len()
			}
		}(i)
	}
	wg.Wait()

	assert.LessOrEqual(t, l.Len(), 3)
}

func TestExportImport(t *testing.T) {
	t.Run("round trip preserves limited state", func(t *testing.T) {
		l := New(1, time.Hour, 1, 500, 0)
		defer l.Stop()

		assert.True(t, l.Allow("key1"))
		assert.False(t, l.Allow("key1"))

		data, err := l.Export()
		require.NoError(t, err)

		restored, err := Import(data)
		require.NoError(t, err)
		defer restored.Stop()

		assert.Equal(t, l.interval, restored.interval)
		assert.Equal(t, l.tau, restored.tau)
		assert.Equal(t, l.maxKeys, restored.maxKeys)
		assert.Equal(t, 1, restored.Len())

		assert.False(t, restored.Allow("key1"))
		assert.Greater(t, restored.RetryAfter("key1"), time.Duration(0))
		assert.True(t, restored.Allow("key2"))
	})

	t.Run("empty limiter round trip", func(t *testing.T) {
		l := New(10, time.Second, 5, 100, 0)
		defer l.Stop()

		data, err := l.Export()
		require.NoError(t, err)

		restored, err := Import(data)
		require.NoError(t, err)
		defer restored.Stop()

		assert.Equal(t, 0, restored.Len())
		assert.True(t, restored.Allow("key1"))
	})

	t.Run("drained entries dropped on import", func(t *testing.T) {
		l := New(10, time.Second, 1, 100, 0)
		defer l.Stop()

		l.mu.Lock()
		l.entries["old"] = -int64(time.Hour) // drained long ago
		l.mu.Unlock()

		data, err := l.Export()
		require.NoError(t, err)

		restored, err := Import(data)
		require.NoError(t, err)
		defer restored.Stop()

		assert.Equal(t, 0, restored.Len())
	})

	t.Run("oversized TAT clamped on import", func(t *testing.T) {
		l := New(10, time.Second, 1, 100, 0)
		defer l.Stop()

		l.mu.Lock()
		l.entries["key1"] = math.MaxInt64 / 2 // far beyond any valid lead
		l.mu.Unlock()

		data, err := l.Export()
		require.NoError(t, err)

		restored, err := Import(data)
		require.NoError(t, err)
		defer restored.Stop()

		restored.mu.Lock()
		tat := restored.entries["key1"]
		restored.mu.Unlock()
		assert.LessOrEqual(t, tat, restored.tau+restored.interval)
	})

	t.Run("cleanup goroutine restored", func(t *testing.T) {
		l := New(10, time.Second, 1, 100, time.Minute)
		defer l.Stop()

		data, err := l.Export()
		require.NoError(t, err)

		restored, err := Import(data)
		require.NoError(t, err)
		defer restored.Stop()

		assert.NotNil(t, restored.cleanup)
		assert.Equal(t, time.Minute, restored.cleanupInterval)
	})

	t.Run("garbage data", func(t *testing.T) {
		_, err := Import([]byte("not gob data"))
		assert.Error(t, err)
	})

	t.Run("invalid configuration rejected", func(t *testing.T) {
		for name, d := range map[string]limiterData{
			"zero interval":     {Interval: 0, Tau: 0, MaxKeys: 1},
			"negative tau":      {Interval: 1, Tau: -1, MaxKeys: 1},
			"zero maxKeys":      {Interval: 1, Tau: 0, MaxKeys: 0},
			"interval overflow": {Interval: math.MaxInt64, Tau: 0, MaxKeys: 1},
			"tau overflow":      {Interval: 1, Tau: math.MaxInt64, MaxKeys: 1},
		} {
			var buf bytes.Buffer
			require.NoError(t, gob.NewEncoder(&buf).Encode(d), name)

			_, err := Import(buf.Bytes())
			assert.ErrorIs(t, err, ErrInvalidData, name)
		}
	})
}

func FuzzLimiter_AllowN(f *testing.F) {
	f.Add("key", 10, int64(time.Second), 5, 3)
	f.Add("", 0, int64(0), 0, 0)
	f.Add("a", 1000000, int64(1), 1<<40, 1<<40)
	f.Add("b", 1, int64(math.MaxInt64), math.MaxInt, math.MaxInt)
	f.Add("c", math.MaxInt, int64(-1), math.MinInt, 1)

	f.Fuzz(func(t *testing.T, key string, limit int, periodNs int64, burst, n int) {
		l := New(limit, time.Duration(periodNs), burst, 100, 0)
		defer l.Stop()

		capacity := l.tau/l.interval + 1
		allowed := l.allowN(key, n, 0)

		if n < 1 {
			if !allowed {
				t.Error("non-positive n must be allowed")
			}
			if l.Len() != 0 {
				t.Error("non-positive n must not create state")
			}
			return
		}

		if want := int64(n) <= capacity; allowed != want {
			t.Errorf("allowN(%d) = %v, want %v (capacity %d)", n, allowed, want, capacity)
		}

		if !allowed && l.Len() != 0 {
			t.Error("denied request on fresh key must not create state")
		}

		if allowed {
			d := l.retryAfter(key, 0)
			if d < 0 {
				t.Error("RetryAfter must not be negative")
			}
			if !l.allowN(key, 1, int64(d)) {
				t.Error("request after RetryAfter must be allowed")
			}
		}
	})
}

func FuzzLimiter_RetryAfter(f *testing.F) {
	f.Add("key", 100, int64(time.Second), 1, uint8(10))
	f.Add("", 1, int64(time.Minute), 100, uint8(255))
	f.Add("a", 1, int64(math.MaxInt64), 2, uint8(5))
	f.Add("b", math.MaxInt, int64(1), 1, uint8(3))

	f.Fuzz(func(t *testing.T, key string, limit int, periodNs int64, burst int, requests uint8) {
		l := New(limit, time.Duration(periodNs), burst, 100, 0)
		defer l.Stop()

		var now int64
		for range requests {
			if l.allowN(key, 1, now) {
				continue
			}

			d := l.retryAfter(key, now)
			if d <= 0 {
				t.Fatal("denied request must have positive RetryAfter")
			}

			now += int64(d)
			if !l.allowN(key, 1, now) {
				t.Fatal("request after RetryAfter must be allowed")
			}
		}
	})
}

func BenchmarkAllow(b *testing.B) {
	l := New(1000000, time.Second, 1000, 10000, 0)
	defer l.Stop()

	keys := make([]string, 1000)
	for i := range keys {
		keys[i] = fmt.Sprintf("key%d", i)
	}

	b.Run("single key", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Allow("key")
		}
	})

	b.Run("many keys", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			l.Allow(keys[i%1000])
		}
	})

	b.Run("parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				l.Allow("key")
			}
		})
	})
}

func TestKeyChurnCannotResetActiveLimits(t *testing.T) {
	l := New(1, time.Hour, 2, 2, 0)
	defer l.Stop()
	assert.True(t, l.allowN("victim", 2, 0))
	assert.True(t, l.allowN("attacker", 2, 0))
	for range 10 {
		assert.False(t, l.allowN("new-key", 1, 0))
		assert.False(t, l.allowN("victim", 1, 0))
	}
	assert.Equal(t, 2, l.Len())
	assert.Equal(t, 2*time.Hour, l.retryAfter("new-key", 0))
	assert.Equal(t, time.Hour, l.retryAfter("victim", 0))
	assert.True(t, l.allowN("new-key", 1, int64(2*time.Hour)))
	assert.Equal(t, 1, l.Len())
}
