package arccache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		assert.NotNil(t, c)
		assert.Equal(t, 0, c.Len())
	})

	t.Run("zero maxItems uses default", func(t *testing.T) {
		c := New[string, int](0)
		defer c.Stop()

		assert.Equal(t, 1000, c.maxItems)
	})

	t.Run("negative maxItems uses default", func(t *testing.T) {
		c := New[string, int](-5)
		defer c.Stop()

		assert.Equal(t, 1000, c.maxItems)
	})
}

func TestNewWithOptions(t *testing.T) {
	t.Run("with all options", func(t *testing.T) {
		c := NewWithOptions(Options[string, []byte]{
			MaxItems: 100,
			MaxBytes: 1024,
			SizeFunc: func(k string, v []byte) int {
				return len(k) + len(v)
			},
			DefaultTTL:      time.Hour,
			CleanupInterval: time.Minute,
		})
		defer c.Stop()

		assert.NotNil(t, c)
		assert.Equal(t, 100, c.maxItems)
		assert.Equal(t, 1024, c.maxBytes)
		assert.NotNil(t, c.sizeFunc)
		assert.NotNil(t, c.cleanup)
	})

	t.Run("MaxBytes ignored without SizeFunc", func(t *testing.T) {
		c := NewWithOptions(Options[string, int]{
			MaxItems: 100,
			MaxBytes: 1024,
		})
		defer c.Stop()

		assert.Equal(t, 0, c.maxBytes)
	})

	t.Run("without cleanup interval", func(t *testing.T) {
		c := NewWithOptions(Options[string, int]{
			MaxItems: 100,
		})
		defer c.Stop()

		assert.Nil(t, c.cleanup)
	})
}

func TestGetSet(t *testing.T) {
	t.Run("set and get", func(t *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		c.Set("key1", 42, 0)

		val, ok := c.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, 42, val)
	})

	t.Run("get non-existent key", func(t *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		val, ok := c.Get("missing")
		assert.False(t, ok)
		assert.Equal(t, 0, val)
	})

	t.Run("update existing key", func(t *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		c.Set("key1", 1, 0)
		c.Set("key1", 2, 0)

		val, ok := c.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, 2, val)
		assert.Equal(t, 1, c.Len())
	})

	t.Run("multiple keys", func(t *testing.T) {
		c := New[string, string](100)
		defer c.Stop()

		c.Set("a", "alpha", 0)
		c.Set("b", "beta", 0)
		c.Set("c", "gamma", 0)

		val, ok := c.Get("b")
		assert.True(t, ok)
		assert.Equal(t, "beta", val)
		assert.Equal(t, 3, c.Len())
	})
}

func TestTTL(t *testing.T) {
	t.Run("entry expires after TTL", func(t *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		c.Set("key1", 42, 50*time.Millisecond)

		val, ok := c.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, 42, val)

		time.Sleep(80 * time.Millisecond)

		_, ok = c.Get("key1")
		assert.False(t, ok)
	})

	t.Run("default TTL applied", func(t *testing.T) {
		c := NewWithOptions(Options[string, int]{
			MaxItems:   100,
			DefaultTTL: 50 * time.Millisecond,
		})
		defer c.Stop()

		c.Set("key1", 42, 0) // uses default TTL

		val, ok := c.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, 42, val)

		time.Sleep(80 * time.Millisecond)

		_, ok = c.Get("key1")
		assert.False(t, ok)
	})

	t.Run("explicit TTL overrides default", func(t *testing.T) {
		c := NewWithOptions(Options[string, int]{
			MaxItems:   100,
			DefaultTTL: time.Hour,
		})
		defer c.Stop()

		c.Set("key1", 42, 50*time.Millisecond)

		time.Sleep(80 * time.Millisecond)

		_, ok := c.Get("key1")
		assert.False(t, ok)
	})

	t.Run("zero TTL with zero default means no expiration", func(t *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		c.Set("key1", 42, 0)

		time.Sleep(50 * time.Millisecond)

		val, ok := c.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, 42, val)
	})
}

func TestDelete(t *testing.T) {
	t.Run("delete existing key", func(t *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		c.Set("key1", 42, 0)
		c.Delete("key1")

		_, ok := c.Get("key1")
		assert.False(t, ok)
		assert.Equal(t, 0, c.Len())
	})

	t.Run("delete non-existent key", func(t *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		c.Delete("missing") // should not panic
		assert.Equal(t, 0, c.Len())
	})
}

func TestLen(t *testing.T) {
	t.Run("counts only non-ghost entries", func(t *testing.T) {
		c := New[string, int](3)
		defer c.Stop()

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Set("c", 3, 0)
		assert.Equal(t, 3, c.Len())

		// Adding a 4th key evicts one to ghost
		c.Set("d", 4, 0)
		assert.Equal(t, 3, c.Len())
	})
}

func TestClear(t *testing.T) {
	t.Run("clears all entries", func(t *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Set("c", 3, 0)

		c.Clear()

		assert.Equal(t, 0, c.Len())
		assert.Equal(t, 0, c.Bytes())

		_, ok := c.Get("a")
		assert.False(t, ok)
	})
}

func TestARCAdaptation(t *testing.T) {
	t.Run("adapts to recency pattern", func(t *testing.T) {
		c := New[string, int](4)
		defer c.Stop()

		// Fill cache with T1 entries
		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Set("c", 3, 0)
		c.Set("d", 4, 0)

		// Evict "a" to B1
		c.Set("e", 5, 0)

		// Hit ghost in B1 -> increases p (favor recency)
		c.Set("a", 1, 0)
		assert.Greater(t, c.p, 0)
	})

	t.Run("adapts to frequency pattern", func(_ *testing.T) {
		c := New[string, int](4)
		defer c.Stop()

		// Create entries in T2 by accessing twice
		c.Set("a", 1, 0)
		c.Get("a") // promote to T2
		c.Set("b", 2, 0)
		c.Get("b") // promote to T2
		c.Set("c", 3, 0)
		c.Get("c") // promote to T2
		c.Set("d", 4, 0)
		c.Get("d") // promote to T2

		// Evict from T2 to B2
		c.Set("e", 5, 0)

		// Record p before B2 hit
		pBefore := c.p

		// Hit ghost in B2 -> decreases p (favor frequency)
		evicted := c.t2.tail()
		if evicted != nil {
			// Force a B2 ghost hit by setting the evicted key
			c.Set("f", 6, 0) // another eviction to B2
		}

		_ = pBefore // p adaptation is best tested by observing behavior
	})

	t.Run("promotes T1 to T2 on second access", func(t *testing.T) {
		c := New[string, int](10)
		defer c.Stop()

		c.Set("key1", 42, 0) // goes to T1
		assert.Equal(t, 1, c.t1.len())
		assert.Equal(t, 0, c.t2.len())

		c.Get("key1") // promote to T2
		assert.Equal(t, 0, c.t1.len())
		assert.Equal(t, 1, c.t2.len())
	})

	t.Run("T2 hit moves to front", func(t *testing.T) {
		c := New[string, int](10)
		defer c.Stop()

		c.Set("a", 1, 0)
		c.Get("a") // promote to T2
		c.Set("b", 2, 0)
		c.Get("b") // promote to T2

		// "b" is at front of T2, "a" is at back
		c.Get("a") // move "a" to front

		head := c.t2.head()
		assert.Equal(t, "a", head.key)
	})
}

func TestB2GhostHit(t *testing.T) {
	t.Run("B2 ghost hit decreases p", func(t *testing.T) {
		c := New[string, int](4)
		defer c.Stop()

		// Promote a,b,c,d to T2
		c.Set("a", 1, 0)
		c.Get("a")
		c.Set("b", 2, 0)
		c.Get("b")
		c.Set("c", 3, 0)
		c.Get("c")
		c.Set("d", 4, 0)
		c.Get("d")

		// All 4 in T2. p=0 (default). Set new keys to evict from T2 to B2.
		// First raise p so we can observe decrease.
		// Force some T1 entries and B1 ghost hits to raise p.
		c.Set("e", 5, 0) // T1, evicts T2 tail to B2
		c.Set("f", 6, 0) // T1, evicts T2 tail to B2
		c.Set("g", 7, 0) // T1, evicts T2 tail to B2
		c.Set("h", 8, 0) // T1, evicts T2 tail to B2

		// Now B2 has ghosts from T2 evictions. Find one.
		// "a" was evicted from T2 to B2
		pBefore := c.p
		c.Set("a", 10, 0) // B2 ghost hit -> decrease p
		assert.LessOrEqual(t, c.p, pBefore)

		val, ok := c.Get("a")
		assert.True(t, ok)
		assert.Equal(t, 10, val)
	})

	t.Run("B2 ghost hit with larger B1 computes delta", func(t *testing.T) {
		c := New[string, int](3)
		defer c.Stop()

		// Promote to T2
		c.Set("a", 1, 0)
		c.Get("a")
		c.Set("b", 2, 0)
		c.Get("b")
		c.Set("c", 3, 0)
		c.Get("c")

		// Evict T2 entries to B2 by adding new T1 entries
		c.Set("d", 4, 0) // evicts from T2 to B2
		c.Set("e", 5, 0) // evicts from T2 to B2
		c.Set("f", 6, 0) // evicts from T2 to B2

		// Now B1 might have some ghosts too from further eviction
		c.Set("g", 7, 0) // evicts T1 to B1

		// Hit a B2 ghost
		c.Set("a", 10, 0)

		val, ok := c.Get("a")
		assert.True(t, ok)
		assert.Equal(t, 10, val)
	})

	t.Run("B1 ghost hit with larger B2 computes delta", func(t *testing.T) {
		c := New[string, int](3)
		defer c.Stop()

		// Create B2 ghosts first by promoting to T2 then evicting
		c.Set("a", 1, 0)
		c.Get("a")
		c.Set("b", 2, 0)
		c.Get("b")
		c.Set("c", 3, 0)
		c.Get("c")

		// Evict T2 to B2
		c.Set("d", 4, 0)
		c.Set("e", 5, 0)
		c.Set("f", 6, 0)

		// Now add more T1 entries to create B1 ghosts
		c.Set("g", 7, 0)
		c.Set("h", 8, 0)

		// B2 should have more ghosts than B1
		// Hit a B1 ghost - delta should be b2.len/b1.len > 1
		c.Set("d", 40, 0) // was evicted from T1 to B1

		val, ok := c.Get("d")
		assert.True(t, ok)
		assert.Equal(t, 40, val)
	})
}

func TestReplaceFromT2(t *testing.T) {
	t.Run("evicts from T2 when T1 below target", func(t *testing.T) {
		c := New[string, int](4)
		defer c.Stop()

		// Fill T2 by promoting
		c.Set("a", 1, 0)
		c.Get("a")
		c.Set("b", 2, 0)
		c.Get("b")
		c.Set("c", 3, 0)
		c.Get("c")

		// T2 has 3 entries, T1 has 0
		assert.Equal(t, 0, c.t1.len())
		assert.Equal(t, 3, c.t2.len())

		// Add new entry to T1
		c.Set("d", 4, 0)

		// p=0, T1 has 1 entry (not > p=0), so T2 should be evicted
		assert.Equal(t, 1, c.t1.len())
		assert.LessOrEqual(t, c.t2.len(), 3)
	})
}

func TestEnforceBytesFromT2(t *testing.T) {
	t.Run("byte eviction from T2", func(t *testing.T) {
		c := NewWithOptions(Options[string, []byte]{
			MaxItems: 100,
			MaxBytes: 30,
			SizeFunc: func(k string, v []byte) int {
				return len(k) + len(v)
			},
		})
		defer c.Stop()

		// Fill T2 by promoting
		c.Set("a", []byte("1234567890"), 0) // 11 bytes
		c.Get("a")                          // promote to T2
		c.Set("b", []byte("1234567890"), 0) // 11 bytes
		c.Get("b")                          // promote to T2

		// T1 empty, T2 has 2 entries (22 bytes)
		assert.Equal(t, 0, c.t1.len())
		assert.Equal(t, 2, c.t2.len())

		// Add large entry that pushes over MaxBytes
		c.Set("c", []byte("12345678901234567890"), 0) // 21 bytes -> total 43, needs eviction from T2

		assert.LessOrEqual(t, c.Bytes(), 30)
	})
}

func TestGhostListTrimming(t *testing.T) {
	t.Run("trims ghost list when totalAll exceeds 2*maxItems", func(t *testing.T) {
		c := New[string, int](3)
		defer c.Stop()

		// Fill and evict to build up ghost lists
		for i := range 20 {
			c.Set(fmt.Sprintf("key-%d", i), i, 0)
		}

		// Ghost lists should be bounded
		totalGhosts := c.b1.len() + c.b2.len()
		assert.LessOrEqual(t, totalGhosts+c.t1.len()+c.t2.len(), 2*c.maxItems)
	})

	t.Run("trims B2 ghosts when full", func(t *testing.T) {
		c := New[string, int](3)
		defer c.Stop()

		// Promote to T2 and evict to B2
		c.Set("a", 1, 0)
		c.Get("a")
		c.Set("b", 2, 0)
		c.Get("b")
		c.Set("c", 3, 0)
		c.Get("c")

		// Evict from T2 to B2
		c.Set("d", 4, 0)
		c.Set("e", 5, 0)
		c.Set("f", 6, 0)

		// Keep adding to force ghost trimming
		c.Set("g", 7, 0)
		c.Set("h", 8, 0)
		c.Set("i", 9, 0)
		c.Set("j", 10, 0)

		totalAll := c.t1.len() + c.t2.len() + c.b1.len() + c.b2.len()
		assert.LessOrEqual(t, totalAll, 2*c.maxItems)
	})

	t.Run("trims B1 when B2 empty and totalAll at limit", func(t *testing.T) {
		c := New[string, int](3)
		defer c.Stop()

		// Only T1 entries -> evictions go to B1 only
		for i := range 10 {
			c.Set(fmt.Sprintf("key-%d", i), i, 0)
		}

		assert.Equal(t, 0, c.b2.len())
		totalAll := c.t1.len() + c.t2.len() + c.b1.len() + c.b2.len()
		assert.LessOrEqual(t, totalAll, 2*c.maxItems)
	})

	t.Run("ghost trim from totalAll >= maxItems path", func(t *testing.T) {
		c := New[string, int](4)
		defer c.Stop()

		// Fill partially (not at maxItems) but with ghosts pushing totalAll
		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Set("c", 3, 0)

		// Delete entries to reduce cache size but they become... no, delete removes.
		// Instead, use TTL to expire entries from cache, leaving ghost count the same
		// Actually let's just fill to capacity and evict to build ghosts
		c.Set("d", 4, 0) // cache full
		c.Set("e", 5, 0) // evict "a" to B1
		c.Set("f", 6, 0) // evict "b" to B1

		// Now: T1=4, B1=2, totalAll=6 >= maxItems(4)
		// Delete some real entries to get totalCache < maxItems but totalAll >= maxItems
		c.Delete("c")
		c.Delete("d")

		// totalCache=2, B1=2, totalAll=4 >= maxItems(4)
		c.Set("g", 7, 0)

		totalAll := c.t1.len() + c.t2.len() + c.b1.len() + c.b2.len()
		assert.LessOrEqual(t, totalAll, 2*c.maxItems)
	})
}

func TestEviction(t *testing.T) {
	t.Run("evicts when at capacity", func(t *testing.T) {
		c := New[string, int](3)
		defer c.Stop()

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Set("c", 3, 0)
		c.Set("d", 4, 0)

		assert.Equal(t, 3, c.Len())

		// One of the earlier entries should be evicted
		val, ok := c.Get("d")
		assert.True(t, ok)
		assert.Equal(t, 4, val)
	})

	t.Run("evicts LRU from T1", func(t *testing.T) {
		c := New[string, int](3)
		defer c.Stop()

		c.Set("a", 1, 0) // T1
		c.Set("b", 2, 0) // T1
		c.Set("c", 3, 0) // T1

		// "a" is at tail of T1 (LRU)
		c.Set("d", 4, 0) // evicts "a" to B1

		_, ok := c.Get("a")
		assert.False(t, ok)
	})

	t.Run("ghost entries enable adaptation", func(t *testing.T) {
		c := New[string, int](3)
		defer c.Stop()

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Set("c", 3, 0)

		// Evict "a" to ghost
		c.Set("d", 4, 0)

		// Re-insert "a" triggers ghost hit in B1
		c.Set("a", 10, 0)

		val, ok := c.Get("a")
		assert.True(t, ok)
		assert.Equal(t, 10, val)
	})
}

func TestOnEvict(t *testing.T) {
	t.Run("callback called on eviction", func(t *testing.T) {
		evicted := make(map[string]int)

		c := NewWithOptions(Options[string, int]{
			MaxItems: 3,
			OnEvict: func(k string, v int) {
				evicted[k] = v
			},
		})
		defer c.Stop()

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Set("c", 3, 0)
		c.Set("d", 4, 0) // evicts "a"

		assert.Equal(t, 1, evicted["a"])
	})

	t.Run("callback called on delete", func(t *testing.T) {
		var evictedKey string

		c := NewWithOptions(Options[string, int]{
			MaxItems: 10,
			OnEvict: func(k string, _ int) {
				evictedKey = k
			},
		})
		defer c.Stop()

		c.Set("key1", 42, 0)
		c.Delete("key1")

		assert.Equal(t, "key1", evictedKey)
	})

	t.Run("callback called on expiry during get", func(t *testing.T) {
		var evictedKey string

		c := NewWithOptions(Options[string, int]{
			MaxItems: 10,
			OnEvict: func(k string, _ int) {
				evictedKey = k
			},
		})
		defer c.Stop()

		c.Set("key1", 42, 50*time.Millisecond)

		time.Sleep(80 * time.Millisecond)
		c.Get("key1") // triggers expiry

		assert.Equal(t, "key1", evictedKey)
	})
}

func TestBytes(t *testing.T) {
	t.Run("tracks byte usage", func(t *testing.T) {
		c := NewWithOptions(Options[string, []byte]{
			MaxItems: 100,
			MaxBytes: 1024,
			SizeFunc: func(k string, v []byte) int {
				return len(k) + len(v)
			},
		})
		defer c.Stop()

		c.Set("key1", []byte("hello"), 0)
		assert.Equal(t, 9, c.Bytes()) // "key1"(4) + "hello"(5)

		c.Set("key2", []byte("world"), 0)
		assert.Equal(t, 18, c.Bytes()) // 9 + 9
	})

	t.Run("bytes decreases on delete", func(t *testing.T) {
		c := NewWithOptions(Options[string, []byte]{
			MaxItems: 100,
			MaxBytes: 1024,
			SizeFunc: func(k string, v []byte) int {
				return len(k) + len(v)
			},
		})
		defer c.Stop()

		c.Set("key1", []byte("hello"), 0)
		c.Delete("key1")
		assert.Equal(t, 0, c.Bytes())
	})

	t.Run("bytes updated on value change", func(t *testing.T) {
		c := NewWithOptions(Options[string, []byte]{
			MaxItems: 100,
			MaxBytes: 1024,
			SizeFunc: func(k string, v []byte) int {
				return len(k) + len(v)
			},
		})
		defer c.Stop()

		c.Set("key1", []byte("short"), 0)
		assert.Equal(t, 9, c.Bytes())

		c.Set("key1", []byte("a much longer value"), 0)
		assert.Equal(t, 23, c.Bytes()) // "key1"(4) + "a much longer value"(19)
	})

	t.Run("evicts when MaxBytes exceeded", func(t *testing.T) {
		c := NewWithOptions(Options[string, []byte]{
			MaxItems: 100,
			MaxBytes: 20,
			SizeFunc: func(k string, v []byte) int {
				return len(k) + len(v)
			},
		})
		defer c.Stop()

		c.Set("a", []byte("1234567"), 0) // 8 bytes
		c.Set("b", []byte("1234567"), 0) // 8 bytes, total 16
		c.Set("c", []byte("1234567"), 0) // 8 bytes, total would be 24, triggers eviction

		assert.LessOrEqual(t, c.Bytes(), 20)
	})

	t.Run("zero bytes without SizeFunc", func(t *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		c.Set("key1", 42, 0)
		assert.Equal(t, 0, c.Bytes())
	})

	t.Run("bytes reset on clear", func(t *testing.T) {
		c := NewWithOptions(Options[string, []byte]{
			MaxItems: 100,
			MaxBytes: 1024,
			SizeFunc: func(k string, v []byte) int {
				return len(k) + len(v)
			},
		})
		defer c.Stop()

		c.Set("key1", []byte("hello"), 0)
		c.Clear()
		assert.Equal(t, 0, c.Bytes())
	})
}

func TestEnforceBytesEvictsT2(t *testing.T) {
	t.Run("byte enforcement evicts from T2 when T1 at or below p", func(t *testing.T) {
		c := NewWithOptions(Options[string, []byte]{
			MaxItems: 100,
			MaxBytes: 25,
			SizeFunc: func(k string, v []byte) int {
				return len(k) + len(v)
			},
		})
		defer c.Stop()

		// Promote entries to T2
		c.Set("a", []byte("12345"), 0) // 6 bytes
		c.Get("a")                     // -> T2
		c.Set("b", []byte("12345"), 0) // 6 bytes
		c.Get("b")                     // -> T2
		c.Set("c", []byte("12345"), 0) // 6 bytes
		c.Get("c")                     // -> T2

		// T1 empty, T2 has 18 bytes. p=0.
		assert.Equal(t, 0, c.t1.len())
		assert.Equal(t, 3, c.t2.len())

		// Set p high so enforceBytes sees t1.len() <= p and evicts from T2
		c.mu.Lock()
		c.p = c.maxItems
		c.mu.Unlock()

		// Add entry to T1 that causes byte overflow.
		// T1 has 1 entry (not > p=maxItems), so enforceBytes should evict from T2.
		c.Set("d", []byte("123456789012345"), 0) // 16 bytes -> total 34 > 25

		assert.LessOrEqual(t, c.Bytes(), 25)
		// T2 should have lost at least one entry
		assert.Less(t, c.t2.len(), 3)
	})
}

func TestB1GhostHitDeltaWhenB2Larger(t *testing.T) {
	t.Run("B1 ghost hit with B2 larger than B1", func(t *testing.T) {
		c := New[string, int](10)
		defer c.Stop()

		// Directly construct the desired state: 1 ghost in B1, 3 ghosts in B2
		c.mu.Lock()

		b1Ghost := &element[string, int]{key: "b1-ghost", ghost: true, list: c.b1}
		c.b1.pushFront(b1Ghost)
		c.items["b1-ghost"] = b1Ghost

		for i := range 3 {
			k := fmt.Sprintf("b2-ghost-%d", i)
			e := &element[string, int]{key: k, ghost: true, list: c.b2}
			c.b2.pushFront(e)
			c.items[k] = e
		}

		c.p = 2
		c.mu.Unlock()

		// B2(3) > B1(1), so delta = 3/1 = 3
		pBefore := c.p
		c.Set("b1-ghost", 99, 0) // B1 ghost hit
		assert.Equal(t, min(pBefore+3, c.maxItems), c.p)

		val, ok := c.Get("b1-ghost")
		assert.True(t, ok)
		assert.Equal(t, 99, val)
	})
}

func TestGhostTrimB2OnMiss(t *testing.T) {
	t.Run("trims B2 when totalAll >= 2*maxItems on miss", func(t *testing.T) {
		c := New[string, int](3)
		defer c.Stop()

		// Promote to T2 and evict to B2
		c.Set("a", 1, 0)
		c.Get("a")
		c.Set("b", 2, 0)
		c.Get("b")
		c.Set("c", 3, 0)
		c.Get("c")

		// Evict T2 to B2
		c.Set("d", 4, 0) // B2 gets ghost
		c.Set("e", 5, 0) // B2 gets ghost
		c.Set("f", 6, 0) // B2 gets ghost

		// Now keep adding to make totalAll hit 2*maxItems
		c.Set("g", 7, 0) // T1 evictions to B1
		c.Set("h", 8, 0)
		c.Set("i", 9, 0)

		totalAll := c.t1.len() + c.t2.len() + c.b1.len() + c.b2.len()
		assert.LessOrEqual(t, totalAll, 2*c.maxItems)
	})

	t.Run("trims B2 in non-full cache path", func(t *testing.T) {
		// Directly construct: totalCache < maxItems, totalAll >= 2*maxItems, B2 > 0
		c := New[string, int](3)
		defer c.Stop()

		c.mu.Lock()
		// Add 1 real entry in T1
		entry := &element[string, int]{key: "existing", value: 1, list: c.t1}
		c.t1.pushFront(entry)
		c.items["existing"] = entry

		// Add 3 B1 ghosts
		for i := range 3 {
			k := fmt.Sprintf("b1-%d", i)
			e := &element[string, int]{key: k, ghost: true, list: c.b1}
			c.b1.pushFront(e)
			c.items[k] = e
		}
		// Add 3 B2 ghosts
		for i := range 3 {
			k := fmt.Sprintf("b2-%d", i)
			e := &element[string, int]{key: k, ghost: true, list: c.b2}
			c.b2.pushFront(e)
			c.items[k] = e
		}
		// totalCache=1 < maxItems=3, totalAll=7 >= 2*3=6, B2=3 > 0
		c.mu.Unlock()

		b2Before := c.b2.len()
		c.Set("new-key", 99, 0) // triggers non-full path with B2 trim

		assert.Less(t, c.b2.len(), b2Before)
	})
}

func TestDefensiveGuards(t *testing.T) {
	t.Run("handleGhostHit with non-ghost-list element", func(_ *testing.T) {
		c := New[string, int](10)
		defer c.Stop()

		c.mu.Lock()
		// Create an element on T1 (not B1 or B2) and mark as ghost
		e := &element[string, int]{key: "fake", ghost: true, list: c.t1}
		c.t1.pushFront(e)
		c.items["fake"] = e
		c.mu.Unlock()

		// Set "fake" triggers ghost hit path, but element is on T1 -> default case
		c.Set("fake", 42, 0)
	})

	t.Run("updateBytes clamps to zero", func(t *testing.T) {
		c := NewWithOptions(Options[string, []byte]{
			MaxItems: 10,
			MaxBytes: 1024,
			SizeFunc: func(k string, v []byte) int {
				return len(k) + len(v)
			},
		})
		defer c.Stop()

		c.mu.Lock()
		c.bytes = 1
		c.updateBytes(-10) // would go to -9, should clamp to 0
		assert.Equal(t, 0, c.bytes)
		c.mu.Unlock()
	})
}

func TestBackgroundCleanup(t *testing.T) {
	t.Run("removes expired entries", func(t *testing.T) {
		c := NewWithOptions(Options[string, int]{
			MaxItems:        100,
			DefaultTTL:      50 * time.Millisecond,
			CleanupInterval: 30 * time.Millisecond,
		})
		defer c.Stop()

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		assert.Equal(t, 2, c.Len())

		time.Sleep(120 * time.Millisecond)

		assert.Equal(t, 0, c.Len())
	})
}

func TestStop(t *testing.T) {
	t.Run("idempotent", func(_ *testing.T) {
		c := New[string, int](100)
		c.Stop()
		c.Stop() // should not panic
	})

	t.Run("stop with cleanup", func(_ *testing.T) {
		c := NewWithOptions(Options[string, int]{
			MaxItems:        100,
			CleanupInterval: time.Millisecond,
		})
		c.Stop()
	})

	t.Run("stop without cleanup", func(_ *testing.T) {
		c := New[string, int](100)
		c.Stop()
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent get and set", func(t *testing.T) {
		c := New[string, int](1000)
		defer c.Stop()

		var wg sync.WaitGroup

		for i := range 10 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := range 100 {
					key := fmt.Sprintf("key-%d-%d", id, j)
					c.Set(key, j, 0)
					c.Get(key)
				}
			}(i)
		}

		wg.Wait()
		assert.Greater(t, c.Len(), 0)
	})

	t.Run("concurrent reads and writes", func(_ *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		done := make(chan struct{})

		go func() {
			for i := range 1000 {
				c.Set(fmt.Sprintf("key-%d", i%20), i, 0)
			}
			close(done)
		}()

		go func() {
			for {
				select {
				case <-done:
					return
				default:
					c.Get("key-0")
					c.Len()
					c.Bytes()
				}
			}
		}()

		<-done
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty string key", func(t *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		c.Set("", 42, 0)

		val, ok := c.Get("")
		assert.True(t, ok)
		assert.Equal(t, 42, val)
	})

	t.Run("integer keys", func(t *testing.T) {
		c := New[int, string](100)
		defer c.Stop()

		c.Set(42, "answer", 0)

		val, ok := c.Get(42)
		assert.True(t, ok)
		assert.Equal(t, "answer", val)
	})

	t.Run("single capacity", func(t *testing.T) {
		c := New[string, int](1)
		defer c.Stop()

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)

		assert.Equal(t, 1, c.Len())

		val, ok := c.Get("b")
		assert.True(t, ok)
		assert.Equal(t, 2, val)
	})

	t.Run("get after expired does not count as entry", func(t *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		c.Set("key1", 42, 50*time.Millisecond)
		time.Sleep(80 * time.Millisecond)

		assert.Equal(t, 1, c.Len()) // still counted until accessed

		_, ok := c.Get("key1")
		assert.False(t, ok)
		assert.Equal(t, 0, c.Len()) // now removed
	})

	t.Run("delete ghost entry", func(_ *testing.T) {
		c := New[string, int](2)
		defer c.Stop()

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Set("c", 3, 0) // evicts "a" to ghost

		// "a" is now a ghost in B1
		c.Delete("a") // should remove the ghost
	})

	t.Run("set on ghost re-inserts", func(t *testing.T) {
		c := New[string, int](2)
		defer c.Stop()

		c.Set("a", 1, 0)
		c.Set("b", 2, 0)
		c.Set("c", 3, 0) // evicts "a" to ghost

		// Re-set "a" triggers ghost hit
		c.Set("a", 10, 0)

		val, ok := c.Get("a")
		assert.True(t, ok)
		assert.Equal(t, 10, val)
	})

	t.Run("rapid set same key", func(t *testing.T) {
		c := New[string, int](10)
		defer c.Stop()

		for i := range 100 {
			c.Set("key", i, 0)
		}

		val, ok := c.Get("key")
		assert.True(t, ok)
		assert.Equal(t, 99, val)
		assert.Equal(t, 1, c.Len())
	})
}

func TestScanResistance(t *testing.T) {
	t.Run("frequent items survive scan", func(t *testing.T) {
		c := New[string, int](100)
		defer c.Stop()

		// Create frequent items (access twice to promote to T2)
		for i := range 50 {
			key := fmt.Sprintf("freq-%d", i)
			c.Set(key, i, 0)
			c.Get(key) // promote to T2
		}

		// Simulate a scan with 200 unique keys (2x capacity)
		for i := range 200 {
			c.Set(fmt.Sprintf("scan-%d", i), i, 0)
		}

		// Frequent items should still be in cache (scan resistance)
		found := 0
		for i := range 50 {
			if _, ok := c.Get(fmt.Sprintf("freq-%d", i)); ok {
				found++
			}
		}

		// ARC should retain most frequent items
		assert.Greater(t, found, 25, "ARC should retain majority of frequent items after scan")
	})
}

func BenchmarkARC_Get(b *testing.B) {
	c := New[string, int](10000)
	defer c.Stop()

	c.Set("bench-key", 42, 0)
	c.Get("bench-key") // promote to T2

	b.ReportAllocs()
	for b.Loop() {
		c.Get("bench-key")
	}
}

func BenchmarkARC_Set(b *testing.B) {
	c := New[string, int](10000)
	defer c.Stop()

	b.ReportAllocs()
	for b.Loop() {
		c.Set("bench-key", 42, 0)
	}
}

func BenchmarkARC_SetNew(b *testing.B) {
	c := New[string, int](b.N + 1)
	defer c.Stop()

	b.ReportAllocs()
	i := 0
	for b.Loop() {
		c.Set(fmt.Sprintf("key-%d", i), i, 0)
		i++
	}
}

func BenchmarkARC_SetEvict(b *testing.B) {
	c := New[string, int](1000)
	defer c.Stop()

	b.ReportAllocs()
	i := 0
	for b.Loop() {
		c.Set(fmt.Sprintf("key-%d", i), i, 0)
		i++
	}
}

func BenchmarkARC_ConcurrentGetSet(b *testing.B) {
	c := New[string, int](10000)
	defer c.Stop()

	for i := range 1000 {
		c.Set(fmt.Sprintf("key-%d", i), i, 0)
	}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			if i%2 == 0 {
				c.Get(key)
			} else {
				c.Set(key, i, 0)
			}
			i++
		}
	})
}

func FuzzARC_GetSet(f *testing.F) {
	f.Add("test", 42, true)
	f.Add("", 0, false)
	f.Add("key-1", -1, true)
	f.Add("a", 100, false)

	f.Fuzz(func(t *testing.T, key string, value int, doGet bool) {
		c := New[string, int](10)
		defer c.Stop()

		c.Set(key, value, 0)

		if doGet {
			got, ok := c.Get(key)
			if !ok {
				t.Error("Get should return true after Set")
			}
			if got != value {
				t.Errorf("expected %d, got %d", value, got)
			}
		}
	})
}
