package sievecache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("default capacity on invalid input", func(t *testing.T) {
		c := New[string, int](0)
		assert.Equal(t, 1000, c.maxItems)

		c = New[string, int](-5)
		assert.Equal(t, 1000, c.maxItems)
	})
}

func TestGetSet(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		c := New[string, int](10)

		c.Set("a", 1, 0)
		v, ok := c.Get("a")
		assert.True(t, ok)
		assert.Equal(t, 1, v)

		v, ok = c.Get("missing")
		assert.False(t, ok)
		assert.Zero(t, v)

		assert.Equal(t, 1, c.Len())
	})

	t.Run("update existing key", func(t *testing.T) {
		c := New[string, int](10)

		c.Set("a", 1, 0)
		c.Set("a", 2, 0)

		v, ok := c.Get("a")
		assert.True(t, ok)
		assert.Equal(t, 2, v)
		assert.Equal(t, 1, c.Len())
	})

	t.Run("delete and clear", func(t *testing.T) {
		c := New[string, int](10)

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)

		c.Delete("a")
		_, ok := c.Get("a")
		assert.False(t, ok)
		assert.Equal(t, 1, c.Len())

		c.Delete("missing") // no-op

		c.Clear()
		assert.Equal(t, 0, c.Len())
		_, ok = c.Get("b")
		assert.False(t, ok)
	})
}

func TestEviction(t *testing.T) {
	t.Run("visited entries survive the sweep", func(t *testing.T) {
		c := New[string, int](3)

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Set("c", 3, 0)

		_, ok := c.Get("a")
		require.True(t, ok)

		// a is visited, so the oldest unvisited entry b is evicted.
		c.Set("d", 4, 0)

		_, ok = c.Get("a")
		assert.True(t, ok)
		_, ok = c.Get("b")
		assert.False(t, ok)
		assert.Equal(t, 3, c.Len())
	})

	t.Run("hand resumes where the last sweep stopped", func(t *testing.T) {
		c := New[string, int](3)

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Set("c", 3, 0)
		c.Get("a")
		c.Set("d", 4, 0) // evicts b, hand now past a

		// a's visited bit was cleared during the sweep, but the hand
		// moved beyond it: the next eviction starts at c, not a.
		c.Set("e", 5, 0)

		_, ok := c.Get("a")
		assert.True(t, ok)
		_, ok = c.Get("c")
		assert.False(t, ok)
	})

	t.Run("all visited wraps and evicts the tail", func(t *testing.T) {
		c := New[string, int](2)

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Get("a")
		c.Get("b")

		c.Set("c", 3, 0)

		_, ok := c.Get("a")
		assert.False(t, ok)
		_, ok = c.Get("b")
		assert.True(t, ok)
	})

	t.Run("capacity one", func(t *testing.T) {
		c := New[string, int](1)

		c.Set("a", 1, 0)
		c.Get("a")
		c.Set("b", 2, 0)

		_, ok := c.Get("a")
		assert.False(t, ok)
		v, ok := c.Get("b")
		assert.True(t, ok)
		assert.Equal(t, 2, v)
		assert.Equal(t, 1, c.Len())
	})

	t.Run("capacity never exceeded", func(t *testing.T) {
		c := New[int, int](50)

		for i := range 500 {
			c.Set(i, i, 0)
			assert.LessOrEqual(t, c.Len(), 50)
		}
	})
}

func TestPeek(t *testing.T) {
	t.Run("does not protect from eviction", func(t *testing.T) {
		c := New[string, int](2)

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)

		v, ok := c.Peek("a")
		require.True(t, ok)
		require.Equal(t, 1, v)

		// Peek did not mark a visited, so it is still the oldest
		// unvisited entry and gets evicted.
		c.Set("c", 3, 0)

		_, ok = c.Get("a")
		assert.False(t, ok)
	})

	t.Run("misses on absent and expired keys", func(t *testing.T) {
		c := New[string, int](10)

		_, ok := c.Peek("missing")
		assert.False(t, ok)

		c.Set("a", 1, time.Millisecond)
		time.Sleep(5 * time.Millisecond)

		_, ok = c.Peek("a")
		assert.False(t, ok)
	})
}

func TestGetOrSet(t *testing.T) {
	t.Run("stores on miss, loads on hit", func(t *testing.T) {
		c := New[string, int](10)

		v, loaded := c.GetOrSet("a", 1, 0)
		assert.False(t, loaded)
		assert.Equal(t, 1, v)

		v, loaded = c.GetOrSet("a", 99, 0)
		assert.True(t, loaded)
		assert.Equal(t, 1, v)
	})

	t.Run("expired entry is replaced", func(t *testing.T) {
		c := New[string, int](10)

		c.Set("a", 1, time.Millisecond)
		time.Sleep(5 * time.Millisecond)

		v, loaded := c.GetOrSet("a", 2, 0)
		assert.False(t, loaded)
		assert.Equal(t, 2, v)
	})

	t.Run("load counts as access", func(t *testing.T) {
		c := New[string, int](2)

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.GetOrSet("a", 99, 0) // marks a visited

		c.Set("c", 3, 0) // must evict b, not a

		_, ok := c.Get("a")
		assert.True(t, ok)
		_, ok = c.Get("b")
		assert.False(t, ok)
	})

	t.Run("evicts at capacity", func(t *testing.T) {
		c := New[string, int](1)

		c.GetOrSet("a", 1, 0)
		c.GetOrSet("b", 2, 0)

		assert.Equal(t, 1, c.Len())
		_, ok := c.Get("b")
		assert.True(t, ok)
	})
}

func TestOnEvict(t *testing.T) {
	t.Run("fires on capacity eviction", func(t *testing.T) {
		c := New[string, int](2)

		var gotKey string
		var gotValue int
		calls := 0
		c.OnEvict(func(k string, v int) {
			gotKey, gotValue = k, v
			calls++
		})

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Set("c", 3, 0)

		assert.Equal(t, 1, calls)
		assert.Equal(t, "a", gotKey)
		assert.Equal(t, 1, gotValue)
	})

	t.Run("not fired by delete, clear, or update", func(t *testing.T) {
		c := New[string, int](10)

		calls := 0
		c.OnEvict(func(string, int) { calls++ })

		c.Set("a", 1, 0)
		c.Set("a", 2, 0)
		c.Delete("a")
		c.Set("b", 3, 0)
		c.Clear()

		assert.Zero(t, calls)
	})

	t.Run("callback may reenter the cache", func(t *testing.T) {
		c := New[int, int](2)

		evictions := 0
		c.OnEvict(func(k, _ int) {
			evictions++
			c.Len()
			c.Get(k)
		})

		for i := range 10 {
			c.Set(i, i, 0)
		}

		assert.Equal(t, 8, evictions)
	})

	t.Run("nil disables", func(t *testing.T) {
		c := New[int, int](1)

		calls := 0
		c.OnEvict(func(int, int) { calls++ })
		c.OnEvict(nil)

		c.Set(1, 1, 0)
		c.Set(2, 2, 0)

		assert.Zero(t, calls)
	})
}

func TestNodeReuse(t *testing.T) {
	t.Run("reused node resets visited bit", func(t *testing.T) {
		c := New[string, int](2)

		c.Set("a", 1, time.Millisecond)
		c.Get("a") // visited, then expires
		c.Set("b", 2, 0)

		time.Sleep(5 * time.Millisecond)

		c.Set("c", 3, 0) // evicts expired a despite visited bit; node reused

		c.mu.RLock()
		e := c.items["c"]
		c.mu.RUnlock()
		assert.False(t, e.visited.Load())
	})

	t.Run("steady state evictions allocate nothing", func(t *testing.T) {
		c := New[int, int](64)
		for i := range 64 {
			c.Set(i, i, 0)
		}

		next := 64
		allocs := testing.AllocsPerRun(1000, func() {
			c.Set(next, next, 0)
			next++
		})

		assert.Zero(t, allocs)
	})
}

func TestTTL(t *testing.T) {
	t.Run("expired entries miss", func(t *testing.T) {
		c := New[string, int](10)

		c.Set("a", 1, time.Millisecond)
		c.Set("b", 2, time.Hour)

		time.Sleep(5 * time.Millisecond)

		_, ok := c.Get("a")
		assert.False(t, ok)
		_, ok = c.Get("b")
		assert.True(t, ok)
	})

	t.Run("expired entries evicted first even if visited", func(t *testing.T) {
		c := New[string, int](2)

		c.Set("a", 1, time.Millisecond)
		c.Set("b", 2, 0)
		c.Get("a")
		c.Get("b")

		time.Sleep(5 * time.Millisecond)

		// Both are visited, but a is expired: the sweep takes it
		// without clearing b's bit twice around.
		c.Set("c", 3, 0)

		_, ok := c.Get("b")
		assert.True(t, ok)
		_, ok = c.Get("c")
		assert.True(t, ok)
	})

	t.Run("update refreshes ttl", func(t *testing.T) {
		c := New[string, int](10)

		c.Set("a", 1, time.Millisecond)
		c.Set("a", 2, time.Hour)

		time.Sleep(5 * time.Millisecond)

		v, ok := c.Get("a")
		assert.True(t, ok)
		assert.Equal(t, 2, v)
	})
}

func TestConcurrency(t *testing.T) {
	c := New[int, int](100)

	var wg sync.WaitGroup
	for w := range 10 {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			for i := range 1000 {
				k := (seed*31 + i) % 200
				switch i % 4 {
				case 0:
					c.Set(k, i, 0)
				case 1:
					c.Set(k, i, time.Minute)
				case 2:
					c.Delete(k)
				default:
					c.Get(k)
				}
			}
		}(w)
	}
	wg.Wait()

	assert.LessOrEqual(t, c.Len(), 100)
	c.checkInvariants(t)
}

// checkInvariants verifies the internal queue and index agree.
func (c *Cache[K, V]) checkInvariants(t *testing.T) {
	t.Helper()

	c.mu.RLock()
	defer c.mu.RUnlock()

	require.LessOrEqual(t, len(c.items), c.maxItems, "capacity exceeded")

	forward := 0
	var last *entry[K, V]
	handSeen := c.hand == nil

	for e := c.head; e != nil; e = e.next {
		forward++
		require.LessOrEqual(t, forward, len(c.items), "cycle in queue")

		if e.next != nil {
			require.Same(t, e, e.next.prev, "broken prev link")
		}

		indexed, ok := c.items[e.key]
		require.True(t, ok, "queue entry missing from index")
		require.Same(t, e, indexed, "index points at wrong entry")

		if e == c.hand {
			handSeen = true
		}

		last = e
	}

	require.Equal(t, len(c.items), forward, "queue and index disagree")
	require.True(t, handSeen, "hand points outside the queue")

	if c.head != nil {
		require.Nil(t, c.head.prev, "head has a newer neighbor")
		require.Same(t, last, c.tail, "tail mismatch")
		require.Nil(t, c.tail.next, "tail has an older neighbor")
	} else {
		require.Nil(t, c.tail)
		require.Nil(t, c.hand)
	}
}

func FuzzCache(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3}, 4)
	f.Add([]byte{0, 0, 0, 0, 0, 0}, 1)
	f.Add([]byte{3, 7, 11, 255, 0, 128, 64, 2}, 3)

	f.Fuzz(func(t *testing.T, ops []byte, capacity int) {
		c := New[byte, int]((capacity%8+8)%8 + 1) // capacity 1..8

		for i, op := range ops {
			key := op % 16
			switch op % 7 {
			case 0, 1:
				c.Set(key, i, 0)
			case 2:
				c.Set(key, i, time.Hour)
			case 3:
				c.Get(key)
			case 4:
				c.GetOrSet(key, i, 0)
			case 5:
				c.Peek(key)
			default:
				c.Delete(key)
			}
		}

		c.checkInvariants(t)
	})
}

func BenchmarkCache(b *testing.B) {
	b.Run("get hit", func(b *testing.B) {
		c := New[string, int](1000)
		c.Set("key", 1, 0)

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			c.Get("key")
		}
	})

	b.Run("get hit parallel", func(b *testing.B) {
		c := New[string, int](1000)
		c.Set("key", 1, 0)

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c.Get("key")
			}
		})
	})

	b.Run("set with eviction", func(b *testing.B) {
		c := New[int, int](1000)
		keys := make([]int, 10000)
		for i := range keys {
			keys[i] = i
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			c.Set(keys[i%10000], i, 0)
		}
	})

	b.Run("mixed parallel", func(b *testing.B) {
		c := New[string, int](1000)
		keys := make([]string, 2000)
		for i := range keys {
			keys[i] = fmt.Sprintf("key%d", i)
			c.Set(keys[i], i, 0)
		}

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i%10 == 0 {
					c.Set(keys[i%2000], i, 0)
				} else {
					c.Get(keys[i%2000])
				}
				i++
			}
		})
	})
}
